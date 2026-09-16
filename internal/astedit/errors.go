// Package astedit provides AST-level code transformation routines including declaration insertion, body replacement, and visibility management.
package astedit

import "errors"

var (
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
)
