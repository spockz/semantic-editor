// WHY: Share indexed parameter and argument projection while keeping permutation and unused-scalar removal admission distinct.

package astedit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"slices"
	"strings"
)

// SignatureArgumentEvidence binds compiler evidence to one original argument.
type SignatureArgumentEvidence struct {
	Index  int
	Start  int
	End    int
	Source string
	Value  string
	Type   string
}

// SignatureProjectionProof retains the original typed inputs for candidate validation.
type SignatureProjectionProof struct {
	File           string
	Symbol         string
	Order          []int
	Calls          []SignatureCallSite
	ParameterTypes []string
	OriginalCount  int
	RemovedIndex   *int
}

func signaturePermutationOrder(request SignatureChangeRequest, original []SignatureParameter) ([]int, bool, error) {
	if len(request.Parameters) != len(original) {
		return nil, false, nil
	}
	order := make([]int, len(original))
	seen := make(map[int]bool, len(original))
	changed := false
	for index, mapping := range request.Parameters {
		if mapping.SourceIndex == nil || *mapping.SourceIndex < 0 || *mapping.SourceIndex >= len(original) || seen[*mapping.SourceIndex] {
			return nil, false, fmt.Errorf("change signature: existing parameter mapping must be a complete bijection")
		}
		source := *mapping.SourceIndex
		if !signatureMappingPreserves(mapping, original[source], source) || mapping.Argument != "" {
			return nil, false, fmt.Errorf("change signature: permutation must preserve each original name, type, and variadic status without arguments")
		}
		seen[source] = true
		order[index] = source
		changed = changed || source != index
	}
	if changed {
		for _, parameter := range original {
			if parameter.Variadic {
				return nil, false, fmt.Errorf("change signature: variadic permutation is unsupported")
			}
		}
	}
	return order, changed, nil
}

func planSignatureProjection(request SignatureChangeRequest, proof SignatureCallSetProof, target signatureTargetContext, order []int, removed *int) (SignaturePlan, error) {
	if target.function.Body == nil {
		return SignaturePlan{}, fmt.Errorf("change signature: projection requires a source-owned function body")
	}
	tokenFile := target.fset.File(target.file.Pos())
	open, close := tokenFile.Offset(target.function.Type.Params.Opening)+1, tokenFile.Offset(target.function.Type.Params.Closing)
	if signatureHasComments(target.source[open:close]) || signatureHasDirectives(target.file) {
		return SignaturePlan{}, fmt.Errorf("change signature: parameter comments and source directives are unsupported")
	}
	declarations := make([]string, 0, len(target.parameters))
	for _, field := range target.function.Type.Params.List {
		typeSource := string(target.source[tokenFile.Offset(field.Type.Pos()):tokenFile.Offset(field.Type.End())])
		for _, name := range field.Names {
			declarations = append(declarations, name.Name+" "+typeSource)
		}
	}
	permuted := make([]string, len(order))
	for index, source := range order {
		permuted[index] = declarations[source]
	}
	edits := map[string][]signatureSourceEdit{request.File: {{Start: open, End: close, Replacement: strings.Join(permuted, ", ")}}}
	seen := make(map[string]bool, len(proof.Calls))
	for _, call := range proof.Calls {
		key := fmt.Sprintf("%s:%d:%d", call.File, call.Start, call.End)
		if seen[key] {
			return SignaturePlan{}, fmt.Errorf("change signature: duplicate projection call evidence")
		}
		seen[key] = true
		source, exists := target.files[call.File]
		if !exists {
			return SignaturePlan{}, fmt.Errorf("change signature: projection caller is absent from the explicit read set")
		}
		callEdits, err := signatureProjectionCallEdits(call, source, request.Symbol, order, len(target.parameters), removed)
		if err != nil {
			return SignaturePlan{}, err
		}
		edits[call.File] = append(edits[call.File], callEdits...)
	}
	plan, err := signatureBuildPlan(proof.Files, edits)
	if err != nil {
		return SignaturePlan{}, err
	}
	calls := slices.Clone(proof.Calls)
	for index := range calls {
		calls[index].Arguments = slices.Clone(calls[index].Arguments)
	}
	plan.Projection = &SignatureProjectionProof{File: request.File, Symbol: request.Symbol, Order: slices.Clone(order), Calls: calls, OriginalCount: len(target.parameters), RemovedIndex: removed}
	return plan, nil
}

func signatureProjectionCallEdits(call SignatureCallSite, source []byte, symbol string, order []int, originalCount int, removed *int) ([]signatureSourceEdit, error) {
	if call.Start < 0 || call.End <= call.Start || call.End > len(source) || call.Revision != signatureRevision(source) {
		return nil, fmt.Errorf("change signature: projection caller revision or range is stale")
	}
	callSource := source[call.Start:call.End]
	fset := token.NewFileSet()
	expression, err := parser.ParseExprFrom(fset, call.File, callSource, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("change signature: parse projection caller: %w", err)
	}
	node, ok := expression.(*ast.CallExpr)
	if !ok || !signatureCallsTarget(node.Fun, symbol) || node.Ellipsis.IsValid() || len(node.Args) != originalCount || len(call.Arguments) != originalCount {
		return nil, fmt.Errorf("change signature: projection requires one-to-one direct-call argument evidence")
	}
	if signatureHasComments(callSource) {
		return nil, fmt.Errorf("change signature: projection call comments are unsupported")
	}
	file, parseErr := parser.ParseFile(token.NewFileSet(), call.File, source, parser.ParseComments)
	if parseErr != nil {
		return nil, fmt.Errorf("change signature: parse projection source: %w", parseErr)
	}
	if signatureHasDirectives(file) {
		return nil, fmt.Errorf("change signature: projection source directives are unsupported")
	}
	tokenFile := fset.File(node.Pos())
	for index, argument := range node.Args {
		evidence := call.Arguments[index]
		start, end := call.Start+tokenFile.Offset(argument.Pos()), call.Start+tokenFile.Offset(argument.End())
		if evidence.Index != index || evidence.Start != start || evidence.End != end || evidence.Source != string(source[start:end]) || evidence.Type == "" || ((removed == nil || index == *removed) && evidence.Value == "") {
			return nil, fmt.Errorf("change signature: missing, stale, or mismatched constant argument evidence")
		}
	}
	if removed != nil {
		arguments := make([]string, len(order))
		for index, original := range order {
			arguments[index] = call.Arguments[original].Source
		}
		return []signatureSourceEdit{{Start: call.Start + tokenFile.Offset(node.Lparen) + 1, End: call.Start + tokenFile.Offset(node.Rparen), Replacement: strings.Join(arguments, ", ")}}, nil
	}
	edits := make([]signatureSourceEdit, 0, len(order))
	for index, original := range order {
		evidence := call.Arguments[index]
		edits = append(edits, signatureSourceEdit{Start: evidence.Start, End: evidence.End, Replacement: call.Arguments[original].Source})
	}
	return edits, nil
}

func signatureRemovalOrder(request SignatureChangeRequest, target signatureTargetContext, evidence *SignatureParameterRemovalEvidence) ([]int, int, error) {
	if len(request.Parameters) != len(target.parameters)-1 {
		return nil, 0, fmt.Errorf("change signature: remove exactly one unused scalar parameter")
	}
	order := make([]int, 0, len(request.Parameters))
	next := 0
	for _, mapping := range request.Parameters {
		if mapping.SourceIndex == nil || *mapping.SourceIndex < next || *mapping.SourceIndex >= len(target.parameters) || !signatureMappingPreserves(mapping, target.parameters[*mapping.SourceIndex], *mapping.SourceIndex) || mapping.Argument != "" {
			return nil, 0, fmt.Errorf("change signature: removal must preserve surviving parameter indices, names, types, and order without additions")
		}
		next = *mapping.SourceIndex + 1
		order = append(order, *mapping.SourceIndex)
	}
	removed := 0
	for _, index := range order {
		if index == removed {
			removed++
		}
	}
	if evidence == nil || !evidence.Scalar || !evidence.Unused || evidence.File != request.File || evidence.Revision != signatureRevision(target.source) || evidence.Index != removed || evidence.Name != target.parameters[removed].Name {
		return nil, 0, fmt.Errorf("change signature: removal requires revision-bound proof of one unused scalar parameter")
	}
	return order, removed, nil
}

func signatureHasComments(source []byte) bool {
	fset := token.NewFileSet()
	file := fset.AddFile("", -1, len(source))
	var scan scanner.Scanner
	scan.Init(file, source, nil, scanner.ScanComments)
	for {
		_, kind, _ := scan.Scan()
		if kind == token.COMMENT {
			return true
		}
		if kind == token.EOF {
			return false
		}
	}
}

func signatureHasDirectives(file *ast.File) bool {
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if strings.HasPrefix(comment.Text, "//go:") || strings.HasPrefix(comment.Text, "//line ") || strings.HasPrefix(comment.Text, "/*line ") {
				return true
			}
		}
	}
	return false
}
