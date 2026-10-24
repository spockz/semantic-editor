// WHY: This file adapts exact Go type plans to current-overlay analysis and preflighted operation results.

package operation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/contractedit"
	"github.com/spockz/semantic-editor/internal/contractplan"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"strings"
)

const (
	contractFileField             = "contract_file"
	contractImplementTool         = "refactor_implement_contract"
	contractInspectTool           = "semantic_inspect_symbol"
	contractRenameTool            = "refactor_rename"
	replaceBodyTool               = "semantic_replace_body"
	sourceContractExtractFile     = "refactor_extract_interface.result.receipt.continuations[].file"
	sourceContractExtractSymbol   = "refactor_extract_interface.result.receipt.continuations[].symbol"
	sourceContractImplementFile   = "refactor_implement_contract.result.receipt.continuations[].file"
	sourceContractImplementSymbol = "refactor_implement_contract.result.receipt.continuations[].symbol"
	sourceExtractedContractFile   = "refactor_extract_interface.contract_file"
	sourceExtractedContract       = "refactor_extract_interface.contract"
	sourceConcreteTargetFile      = "refactor_implement_contract.result.type_target.file"
	sourceConcreteTargetSymbol    = "refactor_implement_contract.result.type_target.symbol"
	sourceGeneratedMethodFile     = "refactor_implement_contract.result.generated_methods[].file"
	sourceGeneratedMethodSymbol   = "refactor_implement_contract.result.generated_methods[].symbol"
)

var extractInterfaceParams = append([]ParameterContract{
	{Name: contractDestinationFileField, CLIName: "destination-file", JSONName: contractDestinationFileField, Type: ParamString, Description: "Target: existing same-package source file receiving the interface; Java currently requires the concrete type file", Role: RoleTarget},
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Selector: existing Go or Java source file declaring the concrete type", Required: true, Role: RoleSelector},
	{Name: "in_type", CLIName: "in-type", JSONName: "in_type", Type: ParamString, Description: "Selector: exact named concrete type whose direct methods are selected", Required: true, Role: RoleSelector},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: new simple interface identifier chosen by the caller", Required: true, Role: RoleTarget},
	{Name: contractReceiverKindField, CLIName: "receiver-kind", JSONName: contractReceiverKindField, Type: ParamString, Description: "Policy: for Go, select the pointer or value method set of in_type; Java extracts public instance methods", Enums: []string{"pointer", "value"}, Role: RolePolicy},
	{Name: "members", CLIName: "members", JSONName: "members", Type: ParamArray, Description: "Selector: nonempty closed array of exact direct method targets {file, symbol}", Required: true, Role: RoleSelector, Schema: new(contractMemberArraySchema())},
	{Name: contractExpectedRevisionField, CLIName: wireCLIExpectedRevision, JSONName: contractExpectedRevisionField, Type: ParamString, Description: "Selector precondition: optional full-source digest for file", Role: RoleSelector, Schema: new(nonEmptyStringSchema())},
}, readProjectParams...)

var implementContractParams = append([]ParameterContract{
	{Name: "file", CLIName: "file", JSONName: "file", Type: ParamString, Description: "Selector: existing Go source file declaring the concrete type", Required: true, Role: RoleSelector},
	{Name: "symbol", CLIName: "symbol", JSONName: "symbol", Type: ParamString, Description: "Target: exact named concrete type to receive missing methods; this is an existing declaration", Required: true, Role: RoleTarget},
	{Name: "contract", CLIName: "contract", JSONName: "contract", Type: ParamString, Description: "Selector: exact interface name copied from refactor_extract_interface.contract", Required: true, Role: RoleSelector, SourceFields: []string{sourceExtractedContract}},
	{Name: contractFileField, CLIName: "contract-file", JSONName: contractFileField, Type: ParamString, Description: "Selector: interface file copied from refactor_extract_interface.contract_file; defaults to file", Role: RoleSelector, SourceFields: []string{sourceExtractedContractFile}},
	{Name: contractDestinationFileField, CLIName: "destination-file", JSONName: contractDestinationFileField, Type: ParamString, Description: "Target: existing same-package Go file receiving generated methods; defaults to file", Role: RoleTarget},
	{Name: contractReceiverKindField, CLIName: "receiver-kind", JSONName: contractReceiverKindField, Type: ParamString, Description: "Policy: exact receiver method set for generated methods", Required: true, Enums: []string{"pointer", "value"}, Role: RolePolicy},
	{Name: "body_policy", CLIName: "body-policy", JSONName: "body_policy", Type: ParamString, Description: "Policy: generated missing methods use explicit panic_stub bodies", Required: true, Enums: []string{"panic_stub"}, Role: RolePolicy},
	{Name: contractExpectedRevisionField, CLIName: wireCLIExpectedRevision, JSONName: contractExpectedRevisionField, Type: ParamString, Description: "Selector precondition: optional full-source digest for file", Role: RoleSelector, Schema: new(nonEmptyStringSchema())},
}, readProjectParams...)

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
	javaService       *backend.Service
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
	Project          backend.ProjectContext
	File             string
	Symbol           string
	Contract         string
	ContractFile     string
	DestinationFile  string
	ReceiverKind     contractplan.ReceiverKind
	BodyPolicy       contractplan.BodyPolicy
	ExpectedRevision *string
}

func (r implementContractRequest) GetProjectContext() backend.ProjectContext { return r.Project }

func (r *implementContractRequest) SetProjectContext(project backend.ProjectContext) {
	r.Project = project
}

type extractInterfaceRequest struct {
	Project          backend.ProjectContext
	File             string
	InType           string
	Symbol           string
	DestinationFile  string
	ReceiverKind     contractplan.ReceiverKind
	Members          []contractMemberTarget
	ExpectedRevision *string
}

func (r extractInterfaceRequest) GetProjectContext() backend.ProjectContext { return r.Project }

func (r *extractInterfaceRequest) SetProjectContext(project backend.ProjectContext) {
	r.Project = project
}

type contractSnapshot = contractedit.Snapshot
type contractRevision = contractedit.Revision
type contractMemberTarget = contractedit.MemberTarget

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
	if request.Project, err = parseReadProject(raw, request.File); err != nil {
		return request, err
	}
	if request.Project.Language == "" {
		request.Project.Language = backend.LanguageGo
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
	java := request.Project.Language == backend.LanguageJava
	receiver, err := ParseEnum(raw, contractReceiverKindField, "receiver-kind", []string{"pointer", "value"}, !java, "pointer")
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
	if request.Project, err = parseReadProject(raw, request.File); err != nil {
		return request, err
	}
	if request.Project.Language == "" {
		request.Project.Language = backend.LanguageGo
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
	return contractedit.Digest(data)
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
	consumers := []string{contractInspectTool, contractRenameTool, semanticLookupToolName}
	return &OperationOutputContract{ResultSchema: new(contractResultSchema()), Continuations: []OutputContinuation{
		{Path: "result.receipt.continuations[].file", Schema: text, Role: RoleTarget, Consumers: consumers},
		{Path: "result.receipt.continuations[].symbol", Schema: text, Role: RoleTarget, Consumers: consumers},
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
	receipt, err := finalizeCallReceipt(ctx, cc, root, EditReceiptDraft{Operation: result.Operation, Status: status, PlannedChanges: planned, ReadSet: readSet, Continuations: result.ReceiptCandidates}, func(resolveCtx context.Context, candidate SymbolTarget) (SymbolTarget, bool, error) {
		return resolveContractContinuation(resolveCtx, root, candidate)
	})
	if err != nil {
		return contractOutput{}, fmt.Errorf("finalize contract edit receipt: %w", err)
	}
	result.Receipt = receipt
	if result.Operation == canonicalImplementContractName {
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
	path, err := contractedit.CanonicalPath(root, candidate.File)
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
	pkg := snapshot.PackageForFile(path)
	if pkg == nil || pkg.Types == nil {
		return SymbolTarget{}, false, nil
	}
	name := candidate.Symbol
	if before, after, ok0 := strings.CutLast(name, "."); ok0 {
		receiverName := strings.Trim(before, "() *")
		methodName := after
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
	return contractedit.ParseSource(ctx, root, fset, filename, source, captured, onCapture)
}
