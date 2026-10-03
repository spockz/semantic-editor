// WHY: This file adapts exact Go type plans to current-overlay analysis and preflighted operation results.

package operation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"semedit/internal/contractplan"
	"semedit/internal/sourcebuf"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

const (
	contractFileField             = "contract_file"
	contractImplementTool         = "semantic_implement_contract"
	contractInspectTool           = "semantic_inspect_symbol"
	contractRenameTool            = "semantic_rename"
	replaceBodyTool               = "semantic_replace_body"
	sourceContractExtractFile     = "semantic_extract_interface.result.receipt.continuations[].file"
	sourceContractExtractSymbol   = "semantic_extract_interface.result.receipt.continuations[].symbol"
	sourceContractImplementFile   = "semantic_implement_contract.result.receipt.continuations[].file"
	sourceContractImplementSymbol = "semantic_implement_contract.result.receipt.continuations[].symbol"
	sourceExtractedContractFile   = "semantic_extract_interface.contract_file"
	sourceExtractedContract       = "semantic_extract_interface.contract"
	sourceConcreteTargetFile      = "semantic_implement_contract.result.type_target.file"
	sourceConcreteTargetSymbol    = "semantic_implement_contract.result.type_target.symbol"
	sourceGeneratedMethodFile     = "semantic_implement_contract.result.generated_methods[].file"
	sourceGeneratedMethodSymbol   = "semantic_implement_contract.result.generated_methods[].symbol"
)

var extractInterfaceParams = []ParameterContract{
	{Name: contractDestinationFileField, CLIName: "destination-file", JSONName: contractDestinationFileField, Type: ParamString, Description: "Target: existing same-package Go file receiving the interface; defaults to file", Role: RoleTarget},
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Selector: existing Go source file declaring the concrete type", Required: true, Role: RoleSelector},
	{Name: "in_type", CLIName: "in-type", JSONName: "in_type", Type: ParamString, Description: "Selector: exact named concrete type whose direct methods are selected", Required: true, Role: RoleSelector},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: new simple interface identifier chosen by the caller", Required: true, Role: RoleTarget},
	{Name: contractReceiverKindField, CLIName: "receiver-kind", JSONName: contractReceiverKindField, Type: ParamString, Description: "Policy: select the pointer or value method set of in_type", Required: true, Enums: []string{"pointer", "value"}, Role: RolePolicy},
	{Name: "members", CLIName: "members", JSONName: "members", Type: ParamArray, Description: "Selector: nonempty closed array of exact direct method targets {file, symbol}", Required: true, Role: RoleSelector, Schema: new(contractMemberArraySchema())},
	{Name: contractExpectedRevisionField, CLIName: wireCLIExpectedRevision, JSONName: contractExpectedRevisionField, Type: ParamString, Description: "Selector precondition: optional full-source digest for file", Role: RoleSelector, Schema: new(nonEmptyStringSchema())},
}

var implementContractParams = []ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Selector: existing Go source file declaring the concrete type", Required: true, Role: RoleSelector},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: exact named concrete type to receive missing methods; this is an existing declaration", Required: true, Role: RoleTarget},
	{Name: "contract", CLIName: "contract", JSONName: "contract", Type: ParamString, Description: "Selector: exact interface name copied from semantic_extract_interface.contract", Required: true, Role: RoleSelector, SourceFields: []string{sourceExtractedContract}},
	{Name: contractFileField, CLIName: "contract-file", JSONName: contractFileField, Type: ParamString, Description: "Selector: interface file copied from semantic_extract_interface.contract_file; defaults to file", Role: RoleSelector, SourceFields: []string{sourceExtractedContractFile}},
	{Name: contractDestinationFileField, CLIName: "destination-file", JSONName: contractDestinationFileField, Type: ParamString, Description: "Target: existing same-package Go file receiving generated methods; defaults to file", Role: RoleTarget},
	{Name: contractReceiverKindField, CLIName: "receiver-kind", JSONName: contractReceiverKindField, Type: ParamString, Description: "Policy: exact receiver method set for generated methods", Required: true, Enums: []string{"pointer", "value"}, Role: RolePolicy},
	{Name: "body_policy", CLIName: "body-policy", JSONName: "body_policy", Type: ParamString, Description: "Policy: generated missing methods use explicit panic_stub bodies", Required: true, Enums: []string{"panic_stub"}, Role: RolePolicy},
	{Name: contractExpectedRevisionField, CLIName: wireCLIExpectedRevision, JSONName: contractExpectedRevisionField, Type: ParamString, Description: "Selector precondition: optional full-source digest for file", Role: RoleSelector, Schema: new(nonEmptyStringSchema())},
}

const contractReceiverKindField = "receiver_kind"

type contractOutput struct {
	Operation         string                 `json:"-"`
	Status            string                 `json:"status"`
	NoOp              bool                   `json:"no_op"`
	File              string                 `json:"file"`
	Symbol            string                 `json:"symbol"`
	Contract          string                 `json:"contract,omitempty"`
	ContractFile      string                 `json:"contract_file,omitempty"`
	TypeTarget        *SymbolTarget          `json:"type_target,omitempty"`
	GeneratedMethods  []SymbolTarget         `json:"generated_methods,omitempty"`
	Detail            string                 `json:"detail"`
	Changes           []contractRevision     `json:"changes,omitempty"`
	ReadSet           []contractRevision     `json:"read_set,omitempty"`
	Continuations     []contractMemberTarget `json:"continuations,omitempty"`
	Signatures        []contractMethodShape  `json:"signatures,omitempty"`
	Receipt           EditReceipt            `json:"receipt"`
	ReceiptCandidates []SymbolTarget         `json:"-"`
}

const contractExpectedRevisionField = "expected_revision"

const contractDestinationFileField = "destination_file"

type contractResultShape struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type contractParameterShape struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Variadic bool   `json:"variadic"`
}

type contractMethodShape struct {
	Name       string                   `json:"name"`
	Parameters []contractParameterShape `json:"parameters"`
	Results    []contractResultShape    `json:"results"`
}

type implementContractRequest struct {
	File             string
	Symbol           string
	Contract         string
	ContractFile     string
	DestinationFile  string
	ReceiverKind     contractplan.ReceiverKind
	BodyPolicy       contractplan.BodyPolicy
	ExpectedRevision *string
}

type extractInterfaceRequest struct {
	File             string
	InType           string
	Symbol           string
	DestinationFile  string
	ReceiverKind     contractplan.ReceiverKind
	Members          []contractMemberTarget
	ExpectedRevision *string
}

func (snapshot *contractSnapshot) packageForFile(path string) *packages.Package {
	if snapshot == nil {
		return nil
	}
	for _, pkg := range snapshot.ByPath {
		for _, file := range pkg.GoFiles {
			absolute, err := filepath.Abs(file)
			if err != nil {
				continue
			}
			canonical, err := filepath.EvalSymlinks(absolute)
			if err == nil && canonical == path {
				return pkg
			}
		}
	}
	return nil
}

type contractSnapshot struct {
	Context          context.Context
	Buffer           *sourcebuf.Buffer
	Root             string
	ModuleRoot       string
	Env              []string
	Packages         []*packages.Package
	ByPath           map[string]*packages.Package
	RequiredPkgPaths map[string]bool
	Syntax           map[string]*ast.File
	Contents         map[string][]byte
	ReadSet          []contractRevision
}

type contractRevision struct {
	Path     string
	Revision string
}

type contractMemberTarget struct {
	File   string
	Symbol string
}

func contractImportBindings(pkg *packages.Package, file *ast.File) ([]contractplan.ImportBinding, error) {
	if pkg == nil || pkg.Types == nil || file == nil {
		return nil, fmt.Errorf("destination package and syntax are required")
	}
	bindings := make([]contractplan.ImportBinding, 0, len(file.Imports))
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, fmt.Errorf("decode destination import path: %w", err)
		}
		name := ""
		if spec.Name != nil {
			name = spec.Name.Name
		} else if imported := pkg.Imports[path]; imported != nil {
			name = imported.Name
		}
		if name == "" {
			return nil, fmt.Errorf("resolve destination import name for %q", path)
		}
		bindings = append(bindings, contractplan.ImportBinding{Path: path, Name: name})
	}
	return bindings, nil
}

func renderContractCandidate(source []byte, destination *ast.File, fset *token.FileSet, plan contractplan.Plan) ([]byte, error) {
	if destination == nil || fset == nil {
		return nil, fmt.Errorf("destination syntax and file set are required")
	}
	if plan.Declaration == "" {
		return bytes.Clone(source), nil
	}
	packageFile := fset.File(destination.Pos())
	if packageFile == nil {
		return nil, fmt.Errorf("destination package position has no source file")
	}
	packageEnd := packageFile.Offset(destination.Name.End())
	if packageEnd < 0 || packageEnd > len(source) {
		return nil, fmt.Errorf("destination package clause is outside its source bytes")
	}
	var output bytes.Buffer
	output.Grow(len(source) + len(plan.Declaration) + len(plan.Imports)*48 + 4)
	output.Write(source[:packageEnd])
	if len(plan.Imports) > 0 {
		output.WriteString("\n\nimport (\n")
		for _, imported := range plan.Imports {
			fmt.Fprintf(&output, "\t%s %s\n", imported.Name, strconv.Quote(imported.Path))
		}
		output.WriteString(")\n")
	}
	output.Write(source[packageEnd:])
	output.WriteString("\n\n")
	output.WriteString(plan.Declaration)
	output.WriteByte('\n')
	candidate, err := format.Source(output.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated contract destination: %w", err)
	}
	if _, err := parser.ParseFile(fset, packageFile.Name(), candidate, parser.ParseComments|parser.AllErrors); err != nil {
		return nil, fmt.Errorf("parse generated destination source: %w", err)
	}
	return bytes.Clone(candidate), nil
}

func preflightContractCandidate(ctx context.Context, snapshot *contractSnapshot, destination string, candidate []byte) (*packages.Package, error) {
	state, err := newContractPreflightState(snapshot, destination, candidate)
	if err != nil {
		return nil, err
	}
	if validationErr := validateContractTestSymlinks(snapshot.Root, state.destinationDirectory); validationErr != nil {
		return nil, fmt.Errorf("validate contract test sources: %w", validationErr)
	}
	loaded, err := loadContractPreflightPackages(ctx, state)
	if err != nil {
		return nil, err
	}
	if validationErr := validateContractTestSymlinks(snapshot.Root, state.destinationDirectory); validationErr != nil {
		return nil, fmt.Errorf("revalidate contract test sources: %w", validationErr)
	}
	finalPackages, finalPackage, err := validateContractPreflightPackages(snapshot, state, loaded)
	if err != nil {
		return nil, err
	}
	if err := recordContractPreflightTests(snapshot, state.parsedTests); err != nil {
		return nil, err
	}
	snapshot.ByPath = finalPackages
	return finalPackage, nil
}

func validateContractTestVariants(loaded []*packages.Package, destinationDirectory string) error {
	var validationErr error
	packages.Visit(loaded, func(pkg *packages.Package) bool {
		if pkg == nil {
			return true
		}
		for _, ignored := range pkg.IgnoredFiles {
			if !strings.HasSuffix(ignored, "_test.go") {
				continue
			}
			sameDirectory, err := sameContractDirectory(ignored, destinationDirectory)
			if err != nil {
				validationErr = fmt.Errorf("resolve ignored test source %q: %w", ignored, err)
				return false
			}
			if sameDirectory {
				validationErr = fmt.Errorf("contract preflight cannot prove inactive test source %q under the current build tags", ignored)
				return false
			}
		}
		hasTestSource := false
		for _, filename := range pkg.CompiledGoFiles {
			if !strings.HasSuffix(filename, "_test.go") {
				continue
			}
			sameDirectory, err := sameContractDirectory(filename, destinationDirectory)
			if err != nil {
				validationErr = fmt.Errorf("resolve compiled test source %q: %w", filename, err)
				return false
			}
			if sameDirectory {
				hasTestSource = true
				break
			}
		}
		if !hasTestSource {
			return true
		}
		if pkg.Types == nil {
			validationErr = fmt.Errorf("contract test variant %q has incomplete type information", pkg.ID)
			return false
		}
		if len(pkg.Errors) > 0 {
			validationErr = fmt.Errorf("contract test variant %q has type errors: %s", pkg.ID, pkg.Errors[0].Msg)
			return false
		}
		return true
	}, nil)
	return validationErr
}

func sameContractDirectory(path, directory string) (bool, error) {
	pathDirectory, err := canonicalContractDirectory(filepath.Dir(path))
	if err != nil {
		return false, err
	}
	canonicalDirectory, err := canonicalContractDirectory(directory)
	if err == nil {
		return pathDirectory == canonicalDirectory, nil
	}
	return false, err
}

func canonicalContractDirectory(directory string) (string, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("resolve directory %q: %w", directory, err)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("canonicalize directory %q: %w", absolute, err)
	}
	return canonical, nil
}

func validateContractTestSymlinks(root, directory string) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("read destination package directory %q: %w", directory, err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		if err := contractTestSymlinkError(root, path); err != nil {
			return err
		}
	}
	return nil
}

func contractTestSymlinkError(root, path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("inspect test source %q: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return nil
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolve test source symlink %q: %w", path, err)
	}
	relative, err := filepath.Rel(root, canonical)
	if err != nil {
		return fmt.Errorf("check test source symlink %q against snapshot root: %w", path, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("test source symlink %q targets %q outside the snapshot root; refusing untracked source", path, canonical)
	}
	return fmt.Errorf("destination-local test symlink %q targets %q; contract preflight does not support test symlinks", path, canonical)
}

func contractTypeInFile(pkg *packages.Package, path, name string) (*types.Named, error) {
	if pkg == nil || pkg.Types == nil || pkg.TypesInfo == nil {
		return nil, fmt.Errorf("selected package has incomplete type information")
	}
	object, ok := pkg.Types.Scope().Lookup(name).(*types.TypeName)
	if !ok {
		return nil, fmt.Errorf("type %q is not declared in package %q", name, pkg.PkgPath)
	}
	named, ok := object.Type().(*types.Named)
	if !ok {
		return nil, fmt.Errorf("type %q is not a named type", name)
	}
	file := findContractFile(pkg, path)
	if file == nil {
		return nil, fmt.Errorf("source file %q is not in the selected package", path)
	}
	for _, declaration := range file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.TYPE {
			continue
		}
		for _, spec := range group.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if ok && typeSpec.Name.Name == name && pkg.TypesInfo.Defs[typeSpec.Name] == object {
				return named, nil
			}
		}
	}
	return nil, fmt.Errorf("type %q is not declared by selected file %q", name, path)
}

func findContractFile(pkg *packages.Package, path string) *ast.File {
	if pkg == nil {
		return nil
	}
	for _, file := range pkg.Syntax {
		position := pkg.Fset.Position(file.Pos())
		absolute, err := filepath.Abs(position.Filename)
		if err != nil {
			continue
		}
		canonical, err := filepath.EvalSymlinks(absolute)
		if err == nil && canonical == path {
			return file
		}
	}
	return nil
}

func contractMethodInFile(snapshot *contractSnapshot, pkg *packages.Package, owner *types.Named, member contractMemberTarget) (*types.Func, error) {
	path, err := canonicalContractPath(snapshot.Root, member.File)
	if err != nil {
		return nil, err
	}
	if snapshot.packageForFile(path) != pkg {
		return nil, fmt.Errorf("member file %q does not resolve to the exact owner package", member.File)
	}
	file := findContractFile(pkg, path)
	if file == nil || pkg.TypesInfo == nil {
		return nil, fmt.Errorf("member file %q has no resolved syntax", member.File)
	}
	separator := strings.LastIndex(member.Symbol, ".")
	if separator <= 0 || separator == len(member.Symbol)-1 {
		return nil, fmt.Errorf("member symbol %q must be a receiver-qualified method", member.Symbol)
	}
	wantReceiver := member.Symbol[:separator]
	wantName := member.Symbol[separator+1:]
	for _, declaration := range file.Decls {
		method, ok := declaration.(*ast.FuncDecl)
		if !ok || method.Recv == nil || method.Name.Name != wantName || len(method.Recv.List) != 1 {
			continue
		}
		var rendered bytes.Buffer
		if err := format.Node(&rendered, pkg.Fset, method.Recv.List[0].Type); err != nil {
			return nil, fmt.Errorf("render method receiver %q: %w", member.Symbol, err)
		}
		receiver := rendered.String()
		if strings.HasPrefix(receiver, "*") {
			receiver = "(" + receiver + ")"
		}
		if receiver+"."+wantName != wantReceiver+"."+wantName {
			continue
		}
		object, ok := pkg.TypesInfo.Defs[method.Name].(*types.Func)
		if !ok || object.Pkg() != pkg.Types || object.Origin() != object {
			return nil, fmt.Errorf("method %q has no exact direct object identity", member.Symbol)
		}
		signature, ok := object.Type().(*types.Signature)
		if !ok || signature.Recv() == nil {
			return nil, fmt.Errorf("member %q is not a receiver method", member.Symbol)
		}
		base := types.Unalias(signature.Recv().Type())
		if pointer, ok := base.(*types.Pointer); ok {
			base = types.Unalias(pointer.Elem())
		}
		if !types.Identical(base, owner) {
			return nil, fmt.Errorf("member %q does not belong to the exact owner type", member.Symbol)
		}
		return object, nil
	}
	return nil, fmt.Errorf("direct method %q was not found in %q", member.Symbol, member.File)
}

func resolveContractMembers(snapshot *contractSnapshot, pkg *packages.Package, owner *types.Named, members []contractMemberTarget) ([]*types.Func, error) {
	if len(members) == 0 {
		return nil, fmt.Errorf("members must select at least one direct method")
	}
	resolved := make([]*types.Func, 0, len(members))
	seen := make(map[string]bool, len(members))
	for _, member := range members {
		key := member.File + "\x00" + member.Symbol
		if seen[key] {
			return nil, fmt.Errorf("member %q in %q was selected more than once", member.Symbol, member.File)
		}
		seen[key] = true
		method, err := contractMethodInFile(snapshot, pkg, owner, member)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, method)
	}
	return resolved, nil
}

func parseContractMembers(value any) ([]contractMemberTarget, error) {
	values, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("members must be an array of closed member objects")
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("members must select at least one direct method")
	}
	members := make([]contractMemberTarget, 0, len(values))
	for index, raw := range values {
		object, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("members[%d] must be an object", index)
		}
		for key := range object {
			if key != "file" && key != "symbol" {
				return nil, fmt.Errorf("members[%d] has unknown field %q", index, key)
			}
		}
		file, ok := object["file"].(string)
		if !ok || strings.TrimSpace(file) == "" {
			return nil, fmt.Errorf("members[%d].file must be a nonempty string", index)
		}
		symbol, ok := object["symbol"].(string)
		if !ok || strings.TrimSpace(symbol) == "" {
			return nil, fmt.Errorf("members[%d].symbol must be a nonempty string", index)
		}
		members = append(members, contractMemberTarget{File: file, Symbol: symbol})
	}
	return members, nil
}

func parseExpectedContractRevision(raw map[string]any) (*string, error) {
	value, present := raw[contractExpectedRevisionField]
	if !present {
		return new(string), nil
	}
	revision, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("expected_revision must be a string")
	}
	if strings.TrimSpace(revision) == "" {
		return nil, fmt.Errorf("expected_revision must not be empty when supplied")
	}
	return &revision, nil
}

func parseExtractInterfaceRequest(raw map[string]any) (extractInterfaceRequest, error) {
	var request extractInterfaceRequest
	if err := CheckParams(raw, extractInterfaceParams); err != nil {
		return request, err
	}
	var err error
	if request.File, err = ParseString(raw, "file", "file", true); err != nil {
		return request, err
	}
	if request.InType, err = ParseString(raw, "in_type", "in-type", true); err != nil {
		return request, err
	}
	if request.Symbol, err = ParseString(raw, "symbol", "symbol", true); err != nil {
		return request, err
	}
	if request.DestinationFile, err = ParseString(raw, contractDestinationFileField, "destination-file", false); err != nil {
		return request, err
	}
	receiver, err := ParseEnum(raw, contractReceiverKindField, "receiver-kind", []string{"pointer", "value"}, true, "")
	if err != nil {
		return request, err
	}
	request.ReceiverKind = contractplan.ReceiverKind(receiver)
	request.Members, err = parseContractMembers(raw["members"])
	if err != nil {
		return request, err
	}
	request.ExpectedRevision, err = parseExpectedContractRevision(raw)
	if err != nil {
		return request, err
	}
	return request, nil
}

func parseImplementContractRequest(raw map[string]any) (implementContractRequest, error) {
	var request implementContractRequest
	if err := CheckParams(raw, implementContractParams); err != nil {
		return request, err
	}
	var err error
	if request.File, err = ParseString(raw, "file", "file", true); err != nil {
		return request, err
	}
	if request.Symbol, err = ParseString(raw, "symbol", "symbol", true); err != nil {
		return request, err
	}
	if request.Contract, err = ParseString(raw, "contract", "contract", true); err != nil {
		return request, err
	}
	if request.ContractFile, err = ParseString(raw, contractFileField, "contract-file", false); err != nil {
		return request, err
	}
	if request.DestinationFile, err = ParseString(raw, contractDestinationFileField, "destination-file", false); err != nil {
		return request, err
	}
	receiver, err := ParseEnum(raw, contractReceiverKindField, "receiver-kind", []string{"pointer", "value"}, true, "")
	if err != nil {
		return request, err
	}
	request.ReceiverKind = contractplan.ReceiverKind(receiver)
	bodyPolicy, err := ParseEnum(raw, "body_policy", "body-policy", []string{"panic_stub"}, true, "")
	if err != nil {
		return request, err
	}
	request.BodyPolicy = contractplan.BodyPolicy(bodyPolicy)
	request.ExpectedRevision, err = parseExpectedContractRevision(raw)
	if err != nil {
		return request, err
	}
	return request, nil
}

func contractDigest(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func checkContractRevision(snapshot *contractSnapshot, selected string, expected *string) error {
	if expected == nil || *expected == "" {
		return nil
	}
	data, exists := snapshot.Contents[selected]
	if !exists {
		return fmt.Errorf("selected source %q is missing from the current source snapshot", selected)
	}
	actual := contractDigest(data)
	if actual != *expected {
		return fmt.Errorf("selected source revision changed: expected %q, found %q", *expected, actual)
	}
	return nil
}

func findContractType(snapshot *contractSnapshot, selected, name string) (*packages.Package, *types.Named, error) {
	pkg := snapshot.packageForFile(selected)
	if pkg == nil {
		return nil, nil, fmt.Errorf("selected file %q has no package in the current overlay", selected)
	}
	if pkg.Types == nil || pkg.TypesInfo == nil {
		return nil, nil, fmt.Errorf("selected package %q has incomplete Go type information", pkg.PkgPath)
	}
	if len(pkg.Errors) > 0 {
		return nil, nil, fmt.Errorf("type-check selected package %q: %s", pkg.PkgPath, pkg.Errors[0].Msg)
	}
	named, err := contractTypeInFile(pkg, selected, name)
	if err != nil {
		return nil, nil, err
	}
	return pkg, named, nil
}

func contractPlanSignatures(plan contractplan.Plan) []contractMethodShape {
	signatures := make([]contractMethodShape, 0, len(plan.Methods))
	for _, method := range plan.Methods {
		shape := contractMethodShape{
			Name:       method.Signature.Name,
			Parameters: make([]contractParameterShape, 0, len(method.Signature.Parameters)),
			Results:    make([]contractResultShape, 0, len(method.Signature.Results)),
		}
		for _, parameter := range method.Signature.Parameters {
			shape.Parameters = append(shape.Parameters, contractParameterShape{Name: parameter.Name, Type: parameter.Type, Variadic: parameter.Variadic})
		}
		for _, result := range method.Signature.Results {
			shape.Results = append(shape.Results, contractResultShape{Name: result.Name, Type: result.Type})
		}
		signatures = append(signatures, shape)
	}
	return signatures
}

func finishContractPlan(ctx context.Context, snapshot *contractSnapshot, destination, selected, symbol string, expected *string, plan contractplan.Plan, validate func(*packages.Package) error) (contractOutput, error) {
	if err := checkContractRevision(snapshot, selected, expected); err != nil {
		return contractOutput{}, err
	}
	if plan.Declaration == "" {
		if err := snapshot.Buffer.AssertDisk(); err != nil {
			return contractOutput{}, fmt.Errorf("validate contract no-op read set: %w", err)
		}
		for _, revision := range snapshot.ReadSet {
			path := filepath.Join(snapshot.Root, filepath.FromSlash(revision.Path))
			data, err := sourcebuf.ReadFile(snapshot.Context, path)
			if err != nil {
				return contractOutput{}, fmt.Errorf("validate contract no-op read %q: %w", revision.Path, err)
			}
			if actual := contractDigest(data); actual != revision.Revision {
				return contractOutput{}, fmt.Errorf("contract no-op read %q changed: expected %q, found %q", revision.Path, revision.Revision, actual)
			}
		}
		if validate != nil {
			if err := validate(snapshot.packageForFile(destination)); err != nil {
				return contractOutput{}, fmt.Errorf("validate existing contract no-op: %w", err)
			}
		}
		destinationRevision := contractDigest(snapshot.Contents[destination])
		return contractOutput{Status: "no_op", NoOp: true, File: displayWorkPath(snapshot.Root, destination), Symbol: symbol, Detail: "The requested contract is already satisfied.", ReadSet: append([]contractRevision(nil), snapshot.ReadSet...), ReceiptCandidates: []SymbolTarget{{File: displayWorkPath(snapshot.Root, destination), Symbol: symbol, ExpectedRevision: &destinationRevision}}, Signatures: contractPlanSignatures(plan)}, nil
	}
	destinationFile := snapshot.Syntax[destination]
	if destinationFile == nil {
		return contractOutput{}, fmt.Errorf("destination file %q has no syntax in the current overlay", destination)
	}
	before, exists := snapshot.Contents[destination]
	if !exists {
		return contractOutput{}, fmt.Errorf("destination file %q is absent from the source snapshot", destination)
	}
	candidate, err := renderContractCandidate(before, destinationFile, findPackageFileSet(snapshot, destination), plan)
	if err != nil {
		return contractOutput{}, err
	}
	finalPackage, err := preflightContractCandidate(ctx, snapshot, destination, candidate)
	if err != nil {
		return contractOutput{}, err
	}
	if validate != nil {
		if err := validate(finalPackage); err != nil {
			return contractOutput{}, fmt.Errorf("validate generated contract output: %w", err)
		}
	}
	change := sourcebuf.Change{Path: destination, Before: before, After: candidate}
	if sourcebuf.FromContext(ctx) != nil {
		if err := snapshot.Buffer.Apply(ctx, []sourcebuf.Change{change}); err != nil {
			return contractOutput{}, fmt.Errorf("apply generated contract to working buffer: %w", err)
		}
	} else if err := sourcebuf.WriteFileChecked(sourcebuf.WithoutBuffer(ctx), destination, candidate, snapshot.Buffer.AssertDisk); err != nil {
		return contractOutput{}, fmt.Errorf("publish generated contract: %w", err)
	}
	readSet := append([]contractRevision(nil), snapshot.ReadSet...)
	destinationRevision := contractDigest(candidate)
	return contractOutput{Status: map[bool]string{true: "working_buffer", false: "persisted"}[sourcebuf.FromContext(ctx) != nil], File: displayWorkPath(snapshot.Root, destination), Symbol: symbol, Detail: "Generated and type-checked the requested contract declarations.", Changes: []contractRevision{{Path: displayWorkPath(snapshot.Root, destination), Revision: destinationRevision}}, ReadSet: readSet, ReceiptCandidates: []SymbolTarget{{File: displayWorkPath(snapshot.Root, destination), Symbol: symbol, ExpectedRevision: &destinationRevision}}, Signatures: contractPlanSignatures(plan)}, nil
}

func findPackageFileSet(snapshot *contractSnapshot, destination string) *token.FileSet {
	pkg := snapshot.packageForFile(destination)
	if pkg == nil {
		return nil
	}
	return pkg.Fset
}

func requirePanicBuiltin(pkg *packages.Package, imports []contractplan.ImportBinding) error {
	if pkg == nil || pkg.Types == nil {
		return fmt.Errorf("concrete package type information is required to resolve builtin panic")
	}
	if _, ok := types.Universe.Lookup("panic").(*types.Builtin); !ok {
		return fmt.Errorf("go universe does not define the panic builtin")
	}
	if shadow := pkg.Types.Scope().Lookup("panic"); shadow != nil {
		return fmt.Errorf("panic_stub would resolve panic to package declaration %q", shadow.Name())
	}
	for _, imported := range imports {
		if imported.Name == "panic" {
			return fmt.Errorf("panic_stub would resolve panic to file import %q", imported.Path)
		}
	}
	return nil
}

func contractMemberArraySchema() ValueSchema {
	minimum := 1
	member := MemberTargetSchema()
	return ValueSchema{Kind: SchemaArray, Items: &member, MinItems: &minimum}
}

func contractSignaturesSchema() ValueSchema {
	text := ValueSchema{Kind: SchemaString}
	boolean := ValueSchema{Kind: SchemaBoolean}
	parameter := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{"name": text, "type": text, "variadic": boolean}, Required: []string{"name", "type", "variadic"}}
	result := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{"name": text, "type": text}, Required: []string{"name", "type"}}
	parameters := ValueSchema{Kind: SchemaArray, Items: &parameter}
	results := ValueSchema{Kind: SchemaArray, Items: &result}
	method := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{"name": text, "parameters": parameters, "results": results}, Required: []string{"name", "parameters", "results"}}
	return ValueSchema{Kind: SchemaArray, Items: &method}
}

func contractResultSchema() ValueSchema {
	signatures := contractSignaturesSchema()
	receipt := EditReceiptSchema()
	text := nonEmptyStringSchema()
	typeTarget := SymbolTargetSchema()
	generated := ValueSchema{Kind: SchemaArray, Items: &typeTarget}
	return ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"receipt": receipt, "signatures": signatures, "contract": text, contractFileField: text,
		"type_target": typeTarget, "generated_methods": generated,
	}, Required: []string{"receipt", "signatures"}}
}

func contractOutputContract() *OperationOutputContract {
	text := nonEmptyStringSchema()
	consumers := []string{contractInspectTool, contractRenameTool}
	return &OperationOutputContract{ResultSchema: new(contractResultSchema()), Continuations: []OutputContinuation{
		{Path: "result.receipt.continuations[].file", Schema: text, Role: RoleSelector, Consumers: consumers},
		{Path: "result.receipt.continuations[].symbol", Schema: text, Role: RoleSelector, Consumers: consumers},
	}}
}

func extractInterfaceOutputContract() *OperationOutputContract {
	contract := contractOutputContract()
	text := nonEmptyStringSchema()
	contract.Continuations = append(contract.Continuations,
		OutputContinuation{Field: "contract", Schema: text, Role: RoleSelector, Consumers: []string{contractImplementTool}},
		OutputContinuation{Field: contractFileField, Schema: text, Role: RoleSelector, Consumers: []string{contractImplementTool}},
	)
	return contract
}

func implementContractOutputContract() *OperationOutputContract {
	contract := contractOutputContract()
	text := nonEmptyStringSchema()
	contract.Continuations = append(contract.Continuations,
		OutputContinuation{Path: "result.type_target.file", Schema: text, Role: RoleTarget, Consumers: []string{contractInspectTool, contractRenameTool}},
		OutputContinuation{Path: "result.type_target.symbol", Schema: text, Role: RoleTarget, Consumers: []string{contractInspectTool, contractRenameTool}},
		OutputContinuation{Path: "result.generated_methods[].file", Schema: text, Role: RoleTarget, Consumers: []string{contractInspectTool, contractRenameTool, replaceBodyTool}},
		OutputContinuation{Path: "result.generated_methods[].symbol", Schema: text, Role: RoleTarget, Consumers: []string{contractInspectTool, contractRenameTool, replaceBodyTool}},
	)
	return contract
}

func projectContractOutput(result contractOutput) (any, error) {
	if err := ValidateEditReceipt(result.Receipt); err != nil {
		return nil, fmt.Errorf("project contract receipt: %w", err)
	}
	signatures := result.Signatures
	if signatures == nil {
		signatures = []contractMethodShape{}
	}
	return struct {
		Receipt         EditReceipt           `json:"receipt"`
		Signatures      []contractMethodShape `json:"signatures"`
		Contract        string                `json:"contract,omitempty"`
		ContractFile    string                `json:"contract_file,omitempty"`
		TypeTarget      *SymbolTarget         `json:"type_target,omitempty"`
		GeneratedMethod []SymbolTarget        `json:"generated_methods,omitempty"`
	}{Receipt: result.Receipt, Signatures: signatures, Contract: result.Contract, ContractFile: result.ContractFile, TypeTarget: result.TypeTarget, GeneratedMethod: result.GeneratedMethods}, nil
}

func finalizeContractOutput(cc CallContext, _ map[string]any, result contractOutput) (contractOutput, error) {
	root := cc.WorkDir
	if root == "" {
		root = cc.Project.RootDir
	}
	status := EditPersisted
	switch result.Status {
	case "refused":
		status = EditRefused
	case "no_op":
		status = EditNoOp
	}
	readSet := make([]FileRevision, 0, len(result.ReadSet))
	for _, revision := range result.ReadSet {
		readSet = append(readSet, FileRevision(revision))
	}
	planned := make([]FileRevision, 0, len(result.Changes))
	for _, revision := range result.Changes {
		planned = append(planned, FileRevision(revision))
	}
	ctx := cc.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	receipt, err := FinalizeEditReceipt(ctx, root, cc.InBatch, EditReceiptDraft{Operation: result.Operation, Status: status, PlannedChanges: planned, ReadSet: readSet, Continuations: result.ReceiptCandidates}, func(resolveCtx context.Context, candidate SymbolTarget) (SymbolTarget, bool, error) {
		return resolveContractContinuation(resolveCtx, root, candidate)
	})
	if err != nil {
		return contractOutput{}, fmt.Errorf("finalize contract edit receipt: %w", err)
	}
	result.Receipt = receipt
	if result.Operation == contractImplementTool {
		for _, continuation := range receipt.Continuations {
			if continuation.Symbol == result.Symbol {
				target := continuation
				result.TypeTarget = &target
				continue
			}
			result.GeneratedMethods = append(result.GeneratedMethods, continuation)
		}
	}
	return result, nil
}

func resolveContractContinuation(ctx context.Context, root string, candidate SymbolTarget) (SymbolTarget, bool, error) {
	path, err := canonicalContractPath(root, candidate.File)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SymbolTarget{}, false, nil
		}
		return SymbolTarget{}, false, err
	}
	snapshot, err := loadContractSnapshot(ctx, root, []string{path})
	if err != nil {
		return SymbolTarget{}, false, err
	}
	pkg := snapshot.packageForFile(path)
	if pkg == nil || pkg.Types == nil {
		return SymbolTarget{}, false, nil
	}
	name := candidate.Symbol
	if separator := strings.LastIndex(name, "."); separator >= 0 {
		receiverName := strings.Trim(name[:separator], "() *")
		methodName := name[separator+1:]
		typeName, ok := pkg.Types.Scope().Lookup(receiverName).(*types.TypeName)
		if !ok {
			return SymbolTarget{}, false, nil
		}
		named, ok := typeName.Type().(*types.Named)
		if !ok {
			return SymbolTarget{}, false, nil
		}
		var receiver types.Type = named
		if strings.HasPrefix(name, "(*") {
			receiver = types.NewPointer(named)
		}
		object, _, _ := types.LookupFieldOrMethod(receiver, false, pkg.Types, methodName)
		method, ok := object.(*types.Func)
		if !ok || method.Type().(*types.Signature).Recv() == nil {
			return SymbolTarget{}, false, nil
		}
		if method.Pkg() != pkg.Types || method.Origin() != method {
			return SymbolTarget{}, false, nil
		}
		position := pkg.Fset.Position(method.Pos())
		return SymbolTarget{File: displayWorkPath(root, position.Filename), Symbol: candidate.Symbol}, true, nil
	}
	typeName, ok := pkg.Types.Scope().Lookup(name).(*types.TypeName)
	if !ok || typeName.Pkg() != pkg.Types {
		return SymbolTarget{}, false, nil
	}
	position := pkg.Fset.Position(typeName.Pos())
	return SymbolTarget{File: displayWorkPath(root, position.Filename), Symbol: candidate.Symbol}, true, nil
}

func formatContractOutput(result contractOutput) (string, error) {
	projected, err := projectContractOutput(result)
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(projected, "", "  ")
	if err != nil {
		return "", fmt.Errorf("format contract output: %w", err)
	}
	return string(data), nil
}

func parseContractSource(ctx context.Context, root string, fset *token.FileSet, filename string, source []byte, captured map[string][]byte, onCapture func(string)) (*ast.File, error) {
	absolute, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("resolve parsed contract file %q: %w", filename, err)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fmt.Errorf("canonicalize parsed contract file %q: %w", filename, err)
	}
	data := bytes.Clone(source)
	if source == nil {
		data, err = sourcebuf.ReadFile(ctx, canonical)
		if err != nil {
			return nil, fmt.Errorf("read parsed contract file %q: %w", canonical, err)
		}
	}
	if relative, relErr := filepath.Rel(root, canonical); relErr == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		captured[canonical] = bytes.Clone(data)
		if onCapture != nil {
			onCapture(canonical)
		}
	}
	file, err := parser.ParseFile(fset, filename, data, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("parse current contract source %q: %w", filename, err)
	}
	return file, nil
}

func validateContractPackageGraph(packageSet map[string]*packages.Package, required map[string]bool) error {
	queue := make([]string, 0, len(required))
	for path := range required {
		queue = append(queue, path)
	}
	slices.Sort(queue)
	visited := make(map[string]bool, len(queue))
	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		if visited[path] {
			continue
		}
		visited[path] = true
		pkg := packageSet[path]
		if pkg == nil {
			return fmt.Errorf("required package %q is absent from the current compiler graph", path)
		}
		if pkg.Types == nil || (required[path] && pkg.TypesInfo == nil) {
			return fmt.Errorf("required package %q has incomplete type information", path)
		}
		if len(pkg.Errors) > 0 {
			return fmt.Errorf("required package %q has type errors: %s", path, pkg.Errors[0].Msg)
		}
		imports := make([]string, 0, len(pkg.Imports))
		for imported := range pkg.Imports {
			imports = append(imports, imported)
		}
		slices.Sort(imports)
		for _, imported := range imports {
			if !visited[imported] {
				queue = append(queue, imported)
			}
		}
	}
	return nil
}
