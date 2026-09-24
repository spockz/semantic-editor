// Package astedit provides AST-level code transformation routines including declaration insertion, body replacement, and visibility management.
package astedit

import (
	"errors"
	"fmt"
	"go/scanner"
	"go/token"
	"strings"
)

var (
	// ErrSwitchAmbiguous indicates that more than one switch matches the selector.
	ErrSwitchAmbiguous = errors.New("ambiguous switch selector")

	// ErrUnsupportedModifier indicates an access modifier is not supported by the active language backend.
	ErrUnsupportedModifier = errors.New("unsupported access modifier")

	// ErrVisibilityMismatch indicates identifier casing conflicts with the explicit access modifier or visibility constraint.
	ErrVisibilityMismatch = errors.New("visibility mismatch")

	// ErrNilBackend indicates an uninitialized language backend.
	ErrNilBackend = errors.New("nil language backend")

	// ErrSectionViolation indicates a declaration was attempted in an incompatible section.
	ErrSectionViolation = errors.New("section placement violation")

	// ErrEmptySnippet indicates an empty code snippet was supplied.
	ErrEmptySnippet = errors.New("empty snippet")

	// ErrSyntax indicates a code snippet has syntax errors.
	ErrSyntax = errors.New("syntax error")

	// ErrNoDeclarations indicates no declarations were found in the snippet.
	ErrNoDeclarations = errors.New("no declarations found in snippet")

	// ErrMultipleDeclarations indicates multiple declarations were found when a single declaration was expected.
	ErrMultipleDeclarations = errors.New("multiple declarations found in snippet")

	// ErrUnexpectedDeclType indicates the declaration node was of an unexpected AST type.
	ErrUnexpectedDeclType = errors.New("unexpected declaration type")

	// ErrNoTypeSpecs indicates a type declaration contains no type specifications.
	ErrNoTypeSpecs = errors.New("no type specs found in declaration")

	// ErrSymbolNotFound indicates the target symbol for relative placement was not found.
	ErrSymbolNotFound = errors.New("target symbol not found")

	// ErrUnsupportedPlacement indicates an unknown placement qualifier was provided.
	ErrUnsupportedPlacement = errors.New("unsupported placement")

	// ErrMissingIdentifier indicates an identifier required for resolution was omitted.
	ErrMissingIdentifier = errors.New("missing identifier")

	// ErrSwitchNotFound indicates a target switch statement was not found.
	ErrSwitchNotFound = errors.New("switch statement not found")

	// ErrAnchorNotFound indicates the anchor case clause for relative placement was not found.
	ErrAnchorNotFound = errors.New("anchor case not found")

	// ErrInferNoSiblings indicates no non-test Go files exist to infer package name.
	ErrInferNoSiblings = errors.New("cannot infer package: no non-test sibling Go files found")

	// ErrFileExists indicates target file already exists and overwrite was not specified.
	ErrFileExists = errors.New("file already exists")

	// ErrNoBody indicates the target function does not have a body.
	ErrNoBody = errors.New("function has no body")
)

// VisibilityMismatchError represents an access modifier or casing visibility constraint violation.
type VisibilityMismatchError struct {
	File       string
	Pos        token.Position
	Identifier string
	Requested  AccessModifier
	Effective  AccessModifier
	Err        error
}

func (e *VisibilityMismatchError) Error() string {
	return fmt.Sprintf("%v: identifier %q has %s casing but access modifier was explicitly specified as %q",
		e.Err, e.Identifier, e.Effective, e.Requested)
}

func (e *VisibilityMismatchError) Unwrap() error { return e.Err }

// SyntaxError represents a syntax or parse failure within an AST snippet or source file with coordinates.
type SyntaxError struct {
	File    string
	Snippet string
	Pos     token.Position
	Cause   error
	Err     error
}

func (e *SyntaxError) Error() string {
	if e.Cause != nil {
		causeMsg := e.Cause.Error()
		if parts := strings.SplitN(causeMsg, ": ", 2); len(parts) == 2 && strings.Count(parts[0], ":") >= 2 {
			causeMsg = parts[1]
		}
		return fmt.Sprintf("%v: %s", e.Err, causeMsg)
	}
	return e.Err.Error()
}

func (e *SyntaxError) Unwrap() error { return e.Err }

// PlacementError represents a failure to place a declaration at a specified target or boundary.
type PlacementError struct {
	File         string
	Strategy     Placement
	TargetSymbol string
	Pos          token.Position
	Err          error
}

func (e *PlacementError) Error() string {
	var details []string
	if e.Strategy != "" {
		details = append(details, fmt.Sprintf("strategy %q", e.Strategy))
	}
	if e.TargetSymbol != "" {
		details = append(details, fmt.Sprintf("target symbol %q", e.TargetSymbol))
	}
	if e.File != "" {
		details = append(details, fmt.Sprintf("in %s", e.File))
	}
	if len(details) > 0 {
		return fmt.Sprintf("%v: %s", e.Err, strings.Join(details, ", "))
	}
	return e.Err.Error()
}

func (e *PlacementError) Unwrap() error { return e.Err }

func extractSyntaxPosition(fset *token.FileSet, err error) token.Position {
	var errList scanner.ErrorList
	if errors.As(err, &errList) && len(errList) > 0 {
		if fset != nil {
			if f := fset.File(token.Pos(1)); f != nil {
				return fset.Position(f.Pos(errList[0].Pos.Offset))
			}
		}
		return errList[0].Pos
	}
	if sErr, ok := errors.AsType[*scanner.Error](err); ok {
		if fset != nil {
			if f := fset.File(token.Pos(1)); f != nil {
				return fset.Position(f.Pos(sErr.Pos.Offset))
			}
		}
		return sErr.Pos
	}
	return token.Position{}
}
