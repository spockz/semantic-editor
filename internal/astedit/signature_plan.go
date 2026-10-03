// WHY: Provides pure plans for signature changes while keeping caller discovery and publication outside AST editing.

package astedit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"strings"
)

// SignatureSourceFile holds one explicit source snapshot for planning.
type SignatureSourceFile struct {
	File   string
	Source []byte
}

// SignatureCallSite identifies a proven direct call in one source snapshot.
type SignatureCallSite struct {
	File  string
	Start int
	End   int
}

// SignatureCallSetProof carries the complete call inventory supplied by semantic analysis.
type SignatureCallSetProof struct {
	Complete bool
	Files    []SignatureSourceFile
	Calls    []SignatureCallSite
}

// SignatureParameter records the original callable parameter shape.
type SignatureParameter struct {
	Name     string
	Type     string
	Variadic bool
}

// SignatureParameterMapping preserves an old parameter or appends one parameter.
type SignatureParameterMapping struct {
	SourceIndex *int
	Name        string
	Type        string
	Variadic    bool
	Argument    string
}

// SignatureChangeRequest describes an append-only signature change.
type SignatureChangeRequest struct {
	File             string
	Symbol           string
	ExpectedRevision string
	Parameters       []SignatureParameterMapping
}

// SignatureConstantEvidence records the semantic equality and timing proof for a moved constant.
type SignatureConstantEvidence struct {
	Type           string
	SelectedValue  string
	ArgumentValue  string
	ArgumentSource string
	MutableRead    bool
	HasSideEffects bool
}

// IntroduceParameterRequest describes replacing one constant body expression with a parameter.
type IntroduceParameterRequest struct {
	Signature     SignatureChangeRequest
	Expression    string
	ParameterName string
	Constant      SignatureConstantEvidence
}

// SignaturePlannedFile contains the validated preimage and proposed final bytes.
type SignaturePlannedFile struct {
	File     string
	Before   []byte
	After    []byte
	Revision string
}

// SignaturePlan contains every changed file produced by a pure signature planner.
type SignaturePlan struct{ Files []SignaturePlannedFile }

// PlanChangeSignature plans an append-only signature and its explicitly proven direct calls.
func PlanChangeSignature(request SignatureChangeRequest, proof SignatureCallSetProof) (SignaturePlan, error) {
	if !proof.Complete {
		return SignaturePlan{}, fmt.Errorf("change signature: complete caller-set proof is required")
	}
	if request.File == "" || request.Symbol == "" {
		return SignaturePlan{}, fmt.Errorf("change signature: file and symbol are required")
	}
	files, err := signatureSourceMap(proof.Files)
	if err != nil {
		return SignaturePlan{}, err
	}
	target, ok := files[request.File]
	if !ok {
		return SignaturePlan{}, fmt.Errorf("change signature: target file %q is absent from the read set", request.File)
	}
	if request.ExpectedRevision != "" && request.ExpectedRevision != signatureRevision(target) {
		return SignaturePlan{}, fmt.Errorf("change signature: selected file revision is stale")
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, request.File, target, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return SignaturePlan{}, fmt.Errorf("change signature: parse target file: %w", err)
	}
	fn, err := signatureTarget(file, request.Symbol)
	if err != nil {
		return SignaturePlan{}, err
	}
	oldParameters, err := signatureParameters(fset, fn.Type.Params)
	if err != nil {
		return SignaturePlan{}, err
	}
	if len(request.Parameters) == len(oldParameters) {
		for index, old := range oldParameters {
			mapping := request.Parameters[index]
			if mapping.SourceIndex == nil || *mapping.SourceIndex != index || mapping.Name != old.Name || mapping.Type != old.Type || mapping.Variadic != old.Variadic || mapping.Argument != "" {
				return SignaturePlan{}, fmt.Errorf("change signature: same-signature mappings must preserve every original parameter without arguments")
			}
		}
		return SignaturePlan{}, nil
	}
	if len(request.Parameters) != len(oldParameters)+1 {
		return SignaturePlan{}, fmt.Errorf("change signature: release one requires every existing parameter and exactly one appended parameter, or the unchanged signature")
	}
	for _, parameter := range oldParameters {
		if parameter.Variadic {
			return SignaturePlan{}, fmt.Errorf("change signature: appending after a variadic parameter is unsupported")
		}
	}
	for index, old := range oldParameters {
		mapping := request.Parameters[index]
		if mapping.SourceIndex == nil || *mapping.SourceIndex != index || mapping.Name != old.Name || mapping.Type != old.Type || mapping.Variadic != old.Variadic || mapping.Argument != "" {
			return SignaturePlan{}, fmt.Errorf("change signature: parameter %d must preserve its original index, name, type, and variadic status", index)
		}
	}
	added := request.Parameters[len(oldParameters)]
	if added.SourceIndex != nil || !signatureIdentifier(added.Name) || added.Name == "_" || added.Type == "" || added.Variadic {
		return SignaturePlan{}, fmt.Errorf("change signature: one valid non-variadic appended parameter is required")
	}
	if err := signatureParseType(added.Type); err != nil {
		return SignaturePlan{}, fmt.Errorf("change signature: invalid appended parameter type: %w", err)
	}
	if err := signatureClosedArgument(added.Argument); err != nil {
		return SignaturePlan{}, fmt.Errorf("change signature: appended argument is unsupported: %w", err)
	}
	for _, parameter := range oldParameters {
		if parameter.Name == added.Name {
			return SignaturePlan{}, fmt.Errorf("change signature: appended parameter %q duplicates an existing name", added.Name)
		}
	}
	targetTokenFile := fset.File(file.Pos())
	if targetTokenFile == nil {
		return SignaturePlan{}, fmt.Errorf("change signature: target token positions are unavailable")
	}
	paramClose := targetTokenFile.Offset(fn.Type.Params.Closing)
	paramOpen := targetTokenFile.Offset(fn.Type.Params.Opening)
	parameterInsertion, parameterText, err := signatureAppendInsertion(target, paramOpen+1, paramClose, added.Name+" "+added.Type, fn.Type.Params.NumFields() > 0)
	if err != nil {
		return SignaturePlan{}, fmt.Errorf("change signature: append parameter: %w", err)
	}
	edits := map[string][]signatureSourceEdit{request.File: {{Start: parameterInsertion, End: parameterInsertion, Replacement: parameterText}}}
	for _, callSite := range proof.Calls {
		source, exists := files[callSite.File]
		if !exists {
			return SignaturePlan{}, fmt.Errorf("change signature: caller %q is absent from the explicit read set", callSite.File)
		}
		if callSite.Start < 0 || callSite.End <= callSite.Start || callSite.End > len(source) {
			return SignaturePlan{}, fmt.Errorf("change signature: caller range in %q is invalid", callSite.File)
		}
		callSource := source[callSite.Start:callSite.End]
		call, err := parser.ParseExpr(string(callSource))
		if err != nil {
			return SignaturePlan{}, fmt.Errorf("change signature: parse proven caller in %q: %w", callSite.File, err)
		}
		callExpr, ok := call.(*ast.CallExpr)
		if !ok || !callExpr.Rparen.IsValid() || !signatureCallsTarget(callExpr.Fun, fn.Name.Name) {
			return SignaturePlan{}, fmt.Errorf("change signature: proof contains a non-direct call to %q in %q", request.Symbol, callSite.File)
		}
		if len(callExpr.Args) != len(oldParameters) {
			return SignaturePlan{}, fmt.Errorf("change signature: call in %q has %d arguments; expected %d", callSite.File, len(callExpr.Args), len(oldParameters))
		}
		if bytes.Contains(callSource, []byte("//")) || bytes.Contains(callSource, []byte("/*")) {
			return SignaturePlan{}, fmt.Errorf("change signature: calls containing comments are outside the supported scope")
		}
		end := callSite.End - 1
		if source[end] != ')' {
			return SignaturePlan{}, fmt.Errorf("change signature: caller range in %q must end at the call's closing parenthesis", callSite.File)
		}
		callOpen := fset.Position(callExpr.Lparen).Offset
		insertion, insertionText, err := signatureAppendInsertion(source, callOpen+1, end, added.Argument, len(callExpr.Args) > 0)
		if err != nil {
			return SignaturePlan{}, fmt.Errorf("change signature: append caller argument in %q: %w", callSite.File, err)
		}
		edits[callSite.File] = append(edits[callSite.File], signatureSourceEdit{Start: insertion, End: insertion, Replacement: insertionText})
	}
	return signatureBuildPlan(proof.Files, edits)
}

// PlanIntroduceParameter plans a typed constant move and its caller updates.
func PlanIntroduceParameter(request IntroduceParameterRequest, proof SignatureCallSetProof) (SignaturePlan, error) {
	if request.Expression == "" || request.ParameterName == "" {
		return SignaturePlan{}, fmt.Errorf("introduce parameter: selected expression and parameter name are required")
	}
	evidence := request.Constant
	if evidence.Type == "" || evidence.SelectedValue == "" || evidence.SelectedValue != evidence.ArgumentValue || evidence.MutableRead || evidence.HasSideEffects {
		return SignaturePlan{}, fmt.Errorf("introduce parameter: equal typed constant values and side-effect-free timing proof are required")
	}
	if err := signatureClosedArgument(evidence.ArgumentSource); err != nil {
		return SignaturePlan{}, fmt.Errorf("introduce parameter: caller argument is unsupported: %w", err)
	}
	added := false
	for _, mapping := range request.Signature.Parameters {
		if mapping.SourceIndex == nil && mapping.Name == request.ParameterName {
			if mapping.Argument != evidence.ArgumentSource || mapping.Type != evidence.Type {
				return SignaturePlan{}, fmt.Errorf("introduce parameter: appended mapping must match the typed constant evidence")
			}
			added = true
		}
	}
	if !added {
		return SignaturePlan{}, fmt.Errorf("introduce parameter: signature must append the evidenced parameter")
	}
	plan, err := PlanChangeSignature(request.Signature, proof)
	if err != nil {
		return SignaturePlan{}, err
	}
	var target *SignaturePlannedFile
	for index := range plan.Files {
		if plan.Files[index].File == request.Signature.File {
			target = &plan.Files[index]
			break
		}
	}
	if target == nil {
		return SignaturePlan{}, fmt.Errorf("introduce parameter: signature planner omitted target file")
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, target.File, target.After, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return SignaturePlan{}, fmt.Errorf("introduce parameter: parse planned target: %w", err)
	}
	fn, err := signatureTarget(file, request.Signature.Symbol)
	if err != nil {
		return SignaturePlan{}, err
	}
	matches := make([]signatureSourceEdit, 0, 1)
	tokenFile := fset.File(file.Pos())
	if tokenFile == nil {
		return SignaturePlan{}, fmt.Errorf("introduce parameter: target token positions are unavailable")
	}
	for _, statement := range fn.Body.List {
		ast.Inspect(statement, func(candidate ast.Node) bool {
			expr, ok := candidate.(ast.Expr)
			if !ok {
				return true
			}
			formatted, formatErr := signatureFormatNode(fset, expr)
			if formatErr == nil && formatted == strings.TrimSpace(request.Expression) {
				matches = append(matches, signatureSourceEdit{Start: tokenFile.Offset(expr.Pos()), End: tokenFile.Offset(expr.End()), Replacement: request.ParameterName})
			}
			return true
		})
	}
	if len(matches) != 1 {
		return SignaturePlan{}, fmt.Errorf("introduce parameter: selected expression matched %d body nodes; expected exactly one", len(matches))
	}
	updated, err := signatureApplyEdits(target.After, matches)
	if err != nil {
		return SignaturePlan{}, fmt.Errorf("introduce parameter: apply body replacement: %w", err)
	}
	target.After = updated
	target.Revision = signatureRevision(updated)
	return plan, nil
}

type signatureSourceEdit struct {
	Start       int
	End         int
	Replacement string
}

func signatureSourceMap(inputs []SignatureSourceFile) (map[string][]byte, error) {
	files := make(map[string][]byte, len(inputs))
	for _, input := range inputs {
		if input.File == "" || len(input.Source) == 0 {
			return nil, fmt.Errorf("signature plan: read-set file and source are required")
		}
		if _, exists := files[input.File]; exists {
			return nil, fmt.Errorf("signature plan: duplicate read-set file %q", input.File)
		}
		files[input.File] = bytes.Clone(input.Source)
	}
	return files, nil
}

func signatureTarget(file *ast.File, symbol string) (*ast.FuncDecl, error) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name != symbol {
			continue
		}
		if ast.IsExported(fn.Name.Name) {
			return nil, fmt.Errorf("signature plan: exported function %q is outside the supported scope", symbol)
		}
		if fn.Type.TypeParams != nil && len(fn.Type.TypeParams.List) != 0 {
			return nil, fmt.Errorf("signature plan: generic function %q is outside the supported scope", symbol)
		}
		return fn, nil
	}
	return nil, fmt.Errorf("signature plan: private free function %q was not found", symbol)
}

func signatureParameters(fset *token.FileSet, fields *ast.FieldList) ([]SignatureParameter, error) {
	if fields == nil {
		return nil, fmt.Errorf("signature plan: target has no parameter list")
	}
	parameters := make([]SignatureParameter, 0, fields.NumFields())
	for _, field := range fields.List {
		if len(field.Names) == 0 {
			return nil, fmt.Errorf("signature plan: unnamed parameters are unsupported")
		}
		typeSource, err := signatureFormatNode(fset, field.Type)
		if err != nil {
			return nil, err
		}
		_, variadic := field.Type.(*ast.Ellipsis)
		for _, name := range field.Names {
			parameters = append(parameters, SignatureParameter{Name: name.Name, Type: typeSource, Variadic: variadic})
		}
	}
	return parameters, nil
}

func signatureFormatNode(fset *token.FileSet, node ast.Node) (string, error) {
	var output bytes.Buffer
	if err := format.Node(&output, fset, node); err != nil {
		return "", fmt.Errorf("format signature node: %w", err)
	}
	return output.String(), nil
}

func signatureParseType(source string) error {
	if _, err := parser.ParseExpr(source); err != nil {
		return err
	}
	return nil
}

func signatureClosedArgument(source string) error {
	node, err := parser.ParseExpr(source)
	if err != nil {
		return fmt.Errorf("parse expression: %w", err)
	}
	switch value := node.(type) {
	case *ast.BasicLit:
		return nil
	case *ast.Ident:
		if value.Name == "nil" || value.Name == "true" || value.Name == "false" {
			return nil
		}
	}
	return fmt.Errorf("only literals, nil, true, or false are accepted")
}

func signatureCallsTarget(expr ast.Expr, name string) bool {
	switch target := expr.(type) {
	case *ast.Ident:
		return target.Name == name
	case *ast.SelectorExpr:
		return target.Sel.Name == name
	default:
		return false
	}
}

func signatureApplyEdits(source []byte, edits []signatureSourceEdit) ([]byte, error) {
	slices.SortFunc(edits, func(left, right signatureSourceEdit) int {
		if left.Start > right.Start {
			return -1
		}
		if left.Start < right.Start {
			return 1
		}
		if left.End > right.End {
			return -1
		}
		if left.End < right.End {
			return 1
		}
		return 0
	})
	output := bytes.Clone(source)
	lastStart := len(source) + 1
	for _, edit := range edits {
		if edit.Start < 0 || edit.End < edit.Start || edit.End > len(source) || edit.End > lastStart {
			return nil, fmt.Errorf("signature plan: source edits overlap or fall outside the input")
		}
		output = bytes.Join([][]byte{output[:edit.Start], []byte(edit.Replacement), output[edit.End:]}, nil)
		lastStart = edit.Start
	}
	return output, nil
}

func signatureBuildPlan(inputs []SignatureSourceFile, edits map[string][]signatureSourceEdit) (SignaturePlan, error) {
	plan := SignaturePlan{Files: make([]SignaturePlannedFile, 0, len(edits))}
	for _, input := range inputs {
		fileEdits := edits[input.File]
		if len(fileEdits) == 0 {
			continue
		}
		after, err := signatureApplyEdits(input.Source, fileEdits)
		if err != nil {
			return SignaturePlan{}, err
		}
		if bytes.Equal(input.Source, after) {
			continue
		}
		plan.Files = append(plan.Files, SignaturePlannedFile{File: input.File, Before: bytes.Clone(input.Source), After: after, Revision: signatureRevision(after)})
	}
	slices.SortFunc(plan.Files, func(left, right SignaturePlannedFile) int { return strings.Compare(left.File, right.File) })
	return plan, nil
}

func signatureRevision(source []byte) string {
	sum := sha256.Sum256(source)
	return hex.EncodeToString(sum[:])
}

func signatureIdentifier(value string) bool {
	return value != "" && token.IsIdentifier(value) && !token.Lookup(value).IsKeyword()
}

func signatureNoPackageNameCollision(inputs []SignatureSourceFile, name string) error {
	for _, input := range inputs {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, input.File, input.Source, parser.ParseComments|parser.AllErrors)
		if err != nil {
			return fmt.Errorf("introduce parameter object: parse read-set file %q: %w", input.File, err)
		}
		for _, decl := range file.Decls {
			switch value := decl.(type) {
			case *ast.FuncDecl:
				if value.Name.Name == name {
					return fmt.Errorf("introduce parameter object: package name %q already exists", name)
				}
			case *ast.GenDecl:
				for _, spec := range value.Specs {
					switch item := spec.(type) {
					case *ast.TypeSpec:
						if item.Name.Name == name {
							return fmt.Errorf("introduce parameter object: package name %q already exists", name)
						}
					case *ast.ValueSpec:
						for _, identifier := range item.Names {
							if identifier.Name == name {
								return fmt.Errorf("introduce parameter object: package name %q already exists", name)
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func signatureAppendDeclaration(source []byte, declaration string) string {
	var prefix string
	if len(source) > 0 && source[len(source)-1] != '\n' {
		prefix = "\n"
	}
	return prefix + "\n" + declaration
}

func signatureAppendInsertion(source []byte, interiorStart, close int, item string, hasExisting bool) (int, string, error) {
	if !hasExisting {
		return close, item, nil
	}
	if interiorStart < 0 || close < interiorStart || close > len(source) {
		return 0, "", fmt.Errorf("signature list bounds are invalid")
	}
	trimmed := bytes.TrimRight(source[interiorStart:close], " \t\r\n")
	if len(trimmed) == 0 {
		return 0, "", fmt.Errorf("signature list has no existing element")
	}
	if trimmed[len(trimmed)-1] != ',' {
		return close, ", " + item, nil
	}
	comma := interiorStart + len(trimmed) - 1
	gap := source[comma+1 : close]
	if bytes.Contains(gap, []byte("//")) || bytes.Contains(gap, []byte("/*")) {
		return 0, "", fmt.Errorf("trailing-comma list comments are unsupported")
	}
	insertion := " " + item
	if bytes.ContainsAny(gap, "\r\n") {
		insertion += ","
	}
	return comma + 1, insertion, nil
}

// ParameterObjectField maps one original parameter to a generated struct field.
type ParameterObjectField struct {
	SourceIndex int
	Name        string
	Type        string
	Tag         string
}

// ParameterObjectRequest describes an all-parameter conversion to one private struct.
type ParameterObjectRequest struct {
	File          string
	Symbol        string
	ParameterName string
	TypeName      string
	Fields        []ParameterObjectField
}

// SignatureBodyBinding identifies a parameter-bound name occurrence in the selected body.
type SignatureBodyBinding struct {
	File        string
	Start       int
	End         int
	SourceIndex int
}

// ParameterObjectProof carries the complete call and body-binding evidence.
type ParameterObjectProof struct {
	Complete     bool
	Files        []SignatureSourceFile
	Calls        []SignatureCallSite
	BodyBindings []SignatureBodyBinding
}

// PlanIntroduceParameterObject plans an all-parameter conversion and keyed caller literals.
func PlanIntroduceParameterObject(request ParameterObjectRequest, proof ParameterObjectProof) (SignaturePlan, error) {
	if !proof.Complete {
		return SignaturePlan{}, fmt.Errorf("introduce parameter object: complete caller and body-binding proof is required")
	}
	if request.File == "" || request.Symbol == "" || !signatureIdentifier(request.ParameterName) || !signatureIdentifier(request.TypeName) {
		return SignaturePlan{}, fmt.Errorf("introduce parameter object: file, symbol, and valid object names are required")
	}
	if request.ParameterName == "_" || request.TypeName == "_" || ast.IsExported(request.ParameterName) || ast.IsExported(request.TypeName) {
		return SignaturePlan{}, fmt.Errorf("introduce parameter object: the callable parameter and object type must be private identifiers")
	}
	files, err := signatureSourceMap(proof.Files)
	if err != nil {
		return SignaturePlan{}, err
	}
	source, ok := files[request.File]
	if !ok {
		return SignaturePlan{}, fmt.Errorf("introduce parameter object: target file is absent from the explicit read set")
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, request.File, source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return SignaturePlan{}, fmt.Errorf("introduce parameter object: parse target file: %w", err)
	}
	fn, err := signatureTarget(file, request.Symbol)
	if err != nil {
		return SignaturePlan{}, err
	}
	parameters, err := signatureParameters(fset, fn.Type.Params)
	if err != nil {
		return SignaturePlan{}, err
	}
	if len(parameters) == 0 || len(parameters) != len(request.Fields) {
		return SignaturePlan{}, fmt.Errorf("introduce parameter object: every original parameter must map to one field")
	}
	fieldByIndex := make(map[int]ParameterObjectField, len(request.Fields))
	fieldNames := make(map[string]bool, len(request.Fields))
	var structSource strings.Builder
	fmt.Fprintf(&structSource, "type %s struct {\n", request.TypeName)
	for _, field := range request.Fields {
		if field.SourceIndex < 0 || field.SourceIndex >= len(parameters) || field.SourceIndex != len(fieldByIndex) {
			return SignaturePlan{}, fmt.Errorf("introduce parameter object: fields must map every source index exactly once in original order")
		}
		if !signatureIdentifier(field.Name) || field.Name == "_" || ast.IsExported(field.Name) || fieldNames[field.Name] {
			return SignaturePlan{}, fmt.Errorf("introduce parameter object: field names must be unique private identifiers")
		}
		original := parameters[field.SourceIndex]
		if original.Variadic || field.Type != original.Type {
			return SignaturePlan{}, fmt.Errorf("introduce parameter object: field %d must preserve the original non-variadic type", field.SourceIndex)
		}
		if err := signatureParseType(field.Type); err != nil {
			return SignaturePlan{}, fmt.Errorf("introduce parameter object: invalid field type: %w", err)
		}
		fieldByIndex[field.SourceIndex] = field
		fieldNames[field.Name] = true
		fmt.Fprintf(&structSource, "\t%s %s", field.Name, field.Type)
		if field.Tag != "" {
			fmt.Fprintf(&structSource, " %s", strconv.Quote(field.Tag))
		}
		structSource.WriteByte('\n')
	}
	structSource.WriteString("}\n")
	if len(fieldByIndex) != len(parameters) {
		return SignaturePlan{}, fmt.Errorf("introduce parameter object: one or more original parameters are unmapped")
	}
	if err := signatureNoPackageNameCollision(proof.Files, request.TypeName); err != nil {
		return SignaturePlan{}, err
	}
	targetTokenFile := fset.File(file.Pos())
	if targetTokenFile == nil {
		return SignaturePlan{}, fmt.Errorf("introduce parameter object: target token positions are unavailable")
	}
	parameterEdits := []signatureSourceEdit{{Start: targetTokenFile.Offset(fn.Type.Params.Opening) + 1, End: targetTokenFile.Offset(fn.Type.Params.Closing), Replacement: request.ParameterName + " " + request.TypeName}}
	edits := map[string][]signatureSourceEdit{request.File: parameterEdits}
	bodyStart := targetTokenFile.Offset(fn.Body.Pos())
	bodyEnd := targetTokenFile.Offset(fn.Body.End())
	for _, binding := range proof.BodyBindings {
		if binding.File != request.File || binding.SourceIndex < 0 || binding.SourceIndex >= len(parameters) || binding.Start < bodyStart || binding.End > bodyEnd || binding.End <= binding.Start {
			return SignaturePlan{}, fmt.Errorf("introduce parameter object: body binding is outside the target body or has an invalid source index")
		}
		original := parameters[binding.SourceIndex].Name
		if binding.End-binding.Start != len(original) || string(source[binding.Start:binding.End]) != original {
			return SignaturePlan{}, fmt.Errorf("introduce parameter object: body binding preimage does not match its original parameter")
		}
		field := fieldByIndex[binding.SourceIndex]
		edits[request.File] = append(edits[request.File], signatureSourceEdit{Start: binding.Start, End: binding.End, Replacement: request.ParameterName + "." + field.Name})
	}
	edits[request.File] = append(edits[request.File], signatureSourceEdit{Start: len(source), End: len(source), Replacement: signatureAppendDeclaration(source, structSource.String())})
	for _, callSite := range proof.Calls {
		callSource, exists := files[callSite.File]
		if !exists {
			return SignaturePlan{}, fmt.Errorf("introduce parameter object: caller is absent from the explicit read set")
		}
		if callSite.Start < 0 || callSite.End <= callSite.Start || callSite.End > len(callSource) {
			return SignaturePlan{}, fmt.Errorf("introduce parameter object: caller range is invalid")
		}
		callSet := token.NewFileSet()
		call, err := parser.ParseExprFrom(callSet, callSite.File, callSource[callSite.Start:callSite.End], parser.ParseComments)
		if err != nil {
			return SignaturePlan{}, fmt.Errorf("introduce parameter object: parse caller: %w", err)
		}
		callExpr, ok := call.(*ast.CallExpr)
		if !ok || !signatureCallsTarget(callExpr.Fun, fn.Name.Name) || len(callExpr.Args) != len(parameters) {
			return SignaturePlan{}, fmt.Errorf("introduce parameter object: caller is not a direct call with the original argument count")
		}
		if bytes.Contains(callSource[callSite.Start:callSite.End], []byte("//")) || bytes.Contains(callSource[callSite.Start:callSite.End], []byte("/*")) {
			return SignaturePlan{}, fmt.Errorf("introduce parameter object: calls containing comments are outside the supported scope")
		}
		callTokenFile := callSet.File(call.Pos())
		if callTokenFile == nil {
			return SignaturePlan{}, fmt.Errorf("introduce parameter object: caller token positions are unavailable")
		}
		var literal strings.Builder
		fmt.Fprintf(&literal, "%s{", request.TypeName)
		for index, field := range request.Fields {
			if index > 0 {
				literal.WriteString(", ")
			}
			arg := callExpr.Args[field.SourceIndex]
			start := callSite.Start + callTokenFile.Offset(arg.Pos())
			end := callSite.Start + callTokenFile.Offset(arg.End())
			if start < callSite.Start || end > callSite.End || end <= start {
				return SignaturePlan{}, fmt.Errorf("introduce parameter object: caller argument range is invalid")
			}
			fmt.Fprintf(&literal, "%s: %s", field.Name, string(callSource[start:end]))
		}
		literal.WriteByte('}')
		lparen := callTokenFile.Offset(callExpr.Lparen)
		rparen := callTokenFile.Offset(callExpr.Rparen)
		edits[callSite.File] = append(edits[callSite.File], signatureSourceEdit{Start: callSite.Start + lparen + 1, End: callSite.Start + rparen, Replacement: literal.String()})
	}
	return signatureBuildPlan(proof.Files, edits)
}
