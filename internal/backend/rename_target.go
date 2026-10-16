// Package backend exposes native rename eligibility without allowing failure
// recovery to publish a refactoring.
package backend

import "context"

type renameTargetValidator interface {
	ValidateRenameTarget(context.Context, RenameRequest) error
}

// ValidateRenameTarget proves that the selected backend accepts a rename without applying it.
func (s *Service) ValidateRenameTarget(ctx context.Context, request RenameRequest) error {
	request.Symbol = NormalizeRenameInput(request.Symbol)
	request.To = NormalizeRenameInput(request.To)
	b, err := s.backendFor(request.Project, OperationRename)
	if err != nil {
		return err
	}
	validator, ok := b.(renameTargetValidator)
	if !ok {
		return &Error{Operation: OperationRename, Language: b.Language(), Err: ErrUnsupportedOperation}
	}
	return validator.ValidateRenameTarget(ctx, request)
}
