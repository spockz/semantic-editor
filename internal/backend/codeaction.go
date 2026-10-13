// Package backend defines neutral, request-scoped contracts for compiler-backed operations.
// WHY: It keeps the operation layer independent of language-specific code-action implementations.
package backend

import (
	"context"
	"errors"
	"fmt"
)

// CodeActionRequest selects one server-provided refactoring over an exact source range.
type CodeActionRequest struct {
	Operation                Operation
	Project                  ProjectContext
	File                     string
	DestinationFile          string
	DestinationType          string
	DestinationPackage       string
	Range                    Range
	Kind                     string
	NewName                  string
	Symbol                   string
	Parameters               []CodeActionParameter
	Expression               string
	ExpressionPath           string
	ParameterName            string
	ExpectedRevision         string
	ParameterObjectType      string
	ParameterObjectFields    []CodeActionParameterField
	ParameterObjectCallSites []CodeActionCallSite
	ParameterObjectMethod    string
}

// CodeActionParameter maps one existing method parameter or adds a new one.
type CodeActionParameter struct {
	SourceIndex *int
	Name        string
	Type        string
	Argument    string
}

// CodeActionResult reports the resulting selected-file source and diagnostics.
type CodeActionResult struct {
	File        string
	Before      []byte
	After       []byte
	Files       []CodeActionFileEdit
	Diagnostics []Diagnostic
}

// CodeActionFileEdit reports one affected file from a compiler-backed workspace refactoring.
type CodeActionFileEdit struct {
	File   string
	Before []byte
	After  []byte
}

// CodeActionBackend delegates a selected refactoring to the language server.
type CodeActionBackend interface {
	ApplyCodeAction(context.Context, CodeActionRequest) (*CodeActionResult, error)
}

// ApplyCodeAction runs one selected compiler-backed refactoring.
func (s *Service) ApplyCodeAction(ctx context.Context, request CodeActionRequest) (*CodeActionResult, error) {
	if request.Operation == "" {
		return nil, fmt.Errorf("code action operation is required")
	}
	b, err := s.backendFor(request.Project, request.Operation)
	if err != nil {
		return nil, err
	}
	refactor, ok := b.(CodeActionBackend)
	if !ok {
		return nil, &Error{Operation: request.Operation, Language: b.Language(), Err: errors.New("backend does not implement CodeActionBackend")}
	}
	return refactor.ApplyCodeAction(ctx, request)
}

// CodeActionParameterField describes one parameter that a code action can add.
type CodeActionParameterField struct {
	SourceIndex   int
	ParameterName string
	Name          string
	Type          string
}

// CodeActionCallSite identifies a resolved invocation rewritten by a code action.
type CodeActionCallSite struct {
	File      string
	Arguments []string
}
