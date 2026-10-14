// WHY: This file keeps the registry package independent while legacy operation plans migrate to contractedit.

package operation

import (
	"context"

	"github.com/spockz/semantic-editor/internal/contractedit"
)

func loadContractSnapshot(ctx context.Context, root string, selectedPaths []string) (*contractSnapshot, error) {
	return contractedit.LoadSnapshot(ctx, root, selectedPaths)
}

func adaptContractResult(result contractedit.Result) contractOutput {
	adapted := contractOutput{
		Status:            result.Status,
		NoOp:              result.NoOp,
		File:              result.File,
		Symbol:            result.Symbol,
		Contract:          result.Contract,
		ContractFile:      result.ContractFile,
		Detail:            result.Detail,
		Changes:           result.Changes,
		ReadSet:           result.ReadSet,
		ReceiptCandidates: make([]SymbolTarget, 0, len(result.ReceiptCandidates)),
		Signatures:        adaptMethodShapes(result.Signatures),
	}
	for _, candidate := range result.ReceiptCandidates {
		adapted.ReceiptCandidates = append(adapted.ReceiptCandidates, SymbolTarget{
			File:             candidate.File,
			Symbol:           candidate.Symbol,
			ExpectedRevision: candidate.ExpectedRevision,
		})
	}
	return adapted
}

func adaptMethodShapes(shapes []contractedit.MethodShape) []contractMethodShape {
	adapted := make([]contractMethodShape, 0, len(shapes))
	for _, shape := range shapes {
		method := contractMethodShape{
			Name:       shape.Name,
			Parameters: make([]contractParameterShape, 0, len(shape.Parameters)),
			Results:    make([]contractResultShape, 0, len(shape.Results)),
		}
		for _, parameter := range shape.Parameters {
			method.Parameters = append(method.Parameters, contractParameterShape{
				Name:     parameter.Name,
				Type:     parameter.Type,
				Variadic: parameter.Variadic,
			})
		}
		for _, result := range shape.Results {
			method.Results = append(method.Results, contractResultShape{Name: result.Name, Type: result.Type})
		}
		adapted = append(adapted, method)
	}
	return adapted
}
