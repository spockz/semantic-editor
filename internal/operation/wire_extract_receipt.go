// Package operation contains bounded extraction receipt projection because receipts must be finalized at the shared operation output boundary.
package operation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"

	"semedit/internal/astedit"
	"semedit/internal/sourcebuf"
	symbols "semedit/internal/symbol"
)

const (
	sourceExtractFunctionReceiptFile   = "semantic_extract_function.result.continuations[].file"
	sourceExtractFunctionReceiptSymbol = "semantic_extract_function.result.continuations[].symbol"
	sourceExtractConstantReceiptFile   = "semantic_extract_constant.result.continuations[].file"
	sourceExtractConstantReceiptSymbol = "semantic_extract_constant.result.continuations[].symbol"
)

// ExtractionResult combines human CLI detail with the typed public edit receipt.
type ExtractionResult struct {
	Edit    FileEditRes
	Draft   EditReceiptDraft
	Receipt EditReceipt
}

func beginExtractionReceipt(ctx context.Context, root, file, operationName string, targets ...SymbolTarget) (EditReceiptDraft, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return EditReceiptDraft{}, fmt.Errorf("resolve extraction receipt root: %w", err)
	}
	selected := filepath.Join(root, filepath.FromSlash(file))
	dir := filepath.Dir(selected)
	if err := sourcebuf.ObserveTree(ctx, dir); err != nil {
		return EditReceiptDraft{}, fmt.Errorf("capture extraction package read set: %w", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return EditReceiptDraft{}, fmt.Errorf("read extraction package directory: %w", err)
	}
	draft := EditReceiptDraft{Operation: operationName, Status: EditPersisted, Continuations: append([]SymbolTarget(nil), targets...)}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		content, readErr := sourcebuf.ReadFile(ctx, path)
		if readErr != nil {
			return EditReceiptDraft{}, fmt.Errorf("read extraction analysis input %q: %w", path, readErr)
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil || !filepath.IsLocal(relative) {
			return EditReceiptDraft{}, fmt.Errorf("extraction read-set path %q is outside the workspace", path)
		}
		draft.ReadSet = append(draft.ReadSet, extractionRevision(filepath.ToSlash(relative), content))
	}
	return draft, nil
}

func completeExtractionReceipt(ctx context.Context, root, file string, draft EditReceiptDraft) (EditReceiptDraft, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return EditReceiptDraft{}, fmt.Errorf("resolve extraction receipt root: %w", err)
	}
	path := filepath.Join(root, filepath.FromSlash(file))
	content, err := sourcebuf.ReadFile(ctx, path)
	if err != nil {
		return EditReceiptDraft{}, fmt.Errorf("read extraction postimage: %w", err)
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || !filepath.IsLocal(relative) {
		return EditReceiptDraft{}, fmt.Errorf("extraction postimage %q is outside the workspace", path)
	}
	draft.PlannedChanges = []FileRevision{extractionRevision(filepath.ToSlash(relative), content)}
	return draft, nil
}

func extractionRevision(path string, content []byte) FileRevision {
	digest := sha256.Sum256(content)
	return FileRevision{Path: path, Revision: hex.EncodeToString(digest[:])}
}

func finalizeExtractionOutput(cc CallContext, result ExtractionResult) (ExtractionResult, error) {
	resolver := func(ctx context.Context, candidate SymbolTarget) (SymbolTarget, bool, error) {
		path := filepath.Join(cc.WorkDir, filepath.FromSlash(candidate.File))
		content, err := sourcebuf.ReadFile(ctx, path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return SymbolTarget{}, false, nil
			}
			return SymbolTarget{}, false, fmt.Errorf("read extraction continuation file: %w", err)
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, content, parser.ParseComments|parser.AllErrors)
		if err != nil {
			return SymbolTarget{}, false, fmt.Errorf("parse extraction continuation file: %w", err)
		}
		if extractionDeclarationLive(file, candidate.Symbol) {
			return candidate, true, nil
		}
		return SymbolTarget{}, false, nil
	}
	receipt, err := FinalizeEditReceipt(cc.Ctx, cc.WorkDir, cc.InBatch, result.Draft, resolver)
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("finalize extraction receipt: %w", err)
	}
	result.Receipt = receipt
	return result, nil
}

func extractionDeclarationLive(file *ast.File, symbol string) bool {
	receiver, name, err := symbols.ParseIdentifier(symbol)
	if err != nil {
		return false
	}
	for _, declaration := range file.Decls {
		switch node := declaration.(type) {
		case *ast.FuncDecl:
			if node.Name.Name != name {
				continue
			}
			if receiver == "" && node.Recv == nil {
				return true
			}
			if receiver != "" && node.Recv != nil && len(node.Recv.List) == 1 {
				_, receiverName, supported := extractionReceiverQualifier(node.Recv.List[0].Type)
				if supported && receiverName == receiver {
					return true
				}
			}
		case *ast.GenDecl:
			if receiver != "" || node.Tok != token.CONST {
				continue
			}
			for _, spec := range node.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, declared := range value.Names {
					if declared.Name == name {
						return true
					}
				}
			}
		}
	}
	return false
}

func extractionReceiptOutput(continuationTools ...string) *OperationOutputContract {
	schema := EditReceiptSchema()
	contract := &OperationOutputContract{ResultSchema: &schema}
	if len(continuationTools) == 0 {
		return contract
	}
	text := nonEmptyStringSchema()
	contract.Continuations = []OutputContinuation{
		{Path: "result.continuations[].file", Schema: text, Role: RoleSelector, Consumers: continuationTools},
		{Path: "result.continuations[].symbol", Schema: text, Role: RoleTarget, Consumers: continuationTools},
	}
	return contract
}

func projectExtractionResult(result ExtractionResult) (any, error) {
	return result.Receipt, nil
}

func formatExtractionResult(result ExtractionResult) (string, error) {
	text, err := formatFileEdit(result.Edit)
	if err != nil {
		return "", err
	}
	if result.Receipt.SchemaVersion != "" {
		encoded, marshalErr := json.Marshal(result.Receipt)
		if marshalErr != nil {
			return "", fmt.Errorf("format extraction receipt: %w", marshalErr)
		}
		text += "\nreceipt: " + string(encoded)
	}
	return text, nil
}

func runExtractFunctionWithReceipt(ctx context.Context, cc CallContext, req ExtractFunctionReq) (ExtractionResult, error) {
	draft, err := beginExtractionReceipt(ctx, cc.WorkDir, req.File, "extract_function")
	if err != nil {
		return ExtractionResult{}, err
	}
	edit, err := runExtractFunction(ctx, cc, req)
	if err != nil {
		return ExtractionResult{}, err
	}
	target, err := extractionFunctionTarget(ctx, cc.WorkDir, req.File, req.Symbol, req.InFunction, req.Kind)
	if err != nil {
		return ExtractionResult{}, err
	}
	draft.Continuations = []SymbolTarget{target}
	draft, err = completeExtractionReceipt(ctx, cc.WorkDir, req.File, draft)
	if err != nil {
		return ExtractionResult{}, err
	}
	return ExtractionResult{Edit: edit, Draft: draft}, nil
}

func runExtractConstantWithReceipt(ctx context.Context, cc CallContext, req ExtractConstantReq) (ExtractionResult, error) {
	draft, err := beginExtractionReceipt(ctx, cc.WorkDir, req.File, "extract_constant", SymbolTarget{File: req.File, Symbol: req.Symbol})
	if err != nil {
		return ExtractionResult{}, err
	}
	edit, err := runExtractConstant(ctx, cc, req)
	if err != nil {
		return ExtractionResult{}, err
	}
	draft, err = completeExtractionReceipt(ctx, cc.WorkDir, req.File, draft)
	if err != nil {
		return ExtractionResult{}, err
	}
	return ExtractionResult{Edit: edit, Draft: draft}, nil
}

func runExtractVariableWithReceipt(ctx context.Context, cc CallContext, req ExtractVariableReq) (ExtractionResult, error) {
	draft, err := beginExtractionReceipt(ctx, cc.WorkDir, req.File, "extract_variable")
	if err != nil {
		return ExtractionResult{}, err
	}
	edit, err := runExtractVariable(ctx, cc, req)
	if err != nil {
		return ExtractionResult{}, err
	}
	draft, err = completeExtractionReceipt(ctx, cc.WorkDir, req.File, draft)
	if err != nil {
		return ExtractionResult{}, err
	}
	return ExtractionResult{Edit: edit, Draft: draft}, nil
}

func runInlineVariableWithReceipt(ctx context.Context, cc CallContext, req InlineVariableReq) (ExtractionResult, error) {
	draft, err := beginExtractionReceipt(ctx, cc.WorkDir, req.File, "inline_variable")
	if err != nil {
		return ExtractionResult{}, err
	}
	edit, err := runInlineVariable(ctx, cc, req)
	if err != nil {
		return ExtractionResult{}, err
	}
	draft, err = completeExtractionReceipt(ctx, cc.WorkDir, req.File, draft)
	if err != nil {
		return ExtractionResult{}, err
	}
	return ExtractionResult{Edit: edit, Draft: draft}, nil
}

func extractionReceiverQualifier(expression ast.Expr) (string, string, bool) {
	switch receiver := expression.(type) {
	case *ast.Ident:
		return receiver.Name, receiver.Name, true
	case *ast.StarExpr:
		base, ok := receiver.X.(*ast.Ident)
		if !ok {
			return "", "", false
		}
		return "(*" + base.Name + ")", base.Name, true
	default:
		return "", "", false
	}
}

func extractionFunctionTarget(ctx context.Context, root, relativeFile, helper, owner string, kind astedit.ExtractKind) (SymbolTarget, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return SymbolTarget{}, fmt.Errorf("resolve extraction target root: %w", err)
	}
	path := filepath.Join(root, filepath.FromSlash(relativeFile))
	content, err := sourcebuf.ReadFile(ctx, path)
	if err != nil {
		return SymbolTarget{}, fmt.Errorf("read extracted helper target: %w", err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), path, content, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return SymbolTarget{}, fmt.Errorf("parse extracted helper target: %w", err)
	}
	if kind == astedit.ExtractKindFunction {
		if !extractionDeclarationLive(file, helper) {
			return SymbolTarget{}, fmt.Errorf("extracted function %q is not live in the result", helper)
		}
		return SymbolTarget{File: filepath.ToSlash(relativeFile), Symbol: helper}, nil
	}
	ownerReceiver, _, err := symbols.ParseIdentifier(owner)
	if err != nil || ownerReceiver == "" {
		return SymbolTarget{}, fmt.Errorf("resolve extracted method receiver from owner %q", owner)
	}
	var target string
	for _, declaration := range file.Decls {
		method, ok := declaration.(*ast.FuncDecl)
		if !ok || method.Recv == nil || method.Name.Name != helper || len(method.Recv.List) != 1 {
			continue
		}
		qualifier, receiverName, supported := extractionReceiverQualifier(method.Recv.List[0].Type)
		if supported && receiverName == ownerReceiver {
			if target != "" {
				return SymbolTarget{}, fmt.Errorf("extracted method %q has ambiguous receiver identity", helper)
			}
			target = qualifier + "." + helper
		}
	}
	if target == "" {
		return SymbolTarget{}, fmt.Errorf("extracted method %q is not live on receiver %q", helper, ownerReceiver)
	}
	return SymbolTarget{File: filepath.ToSlash(relativeFile), Symbol: target}, nil
}
