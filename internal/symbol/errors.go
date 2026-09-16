// Package symbol provides AST symbol resolution and coordinate translation for Go source files.
package symbol

import (
	"errors"
	"fmt"
	"go/token"
)

var (
	// ErrNotFound indicates the queried symbol does not exist in scope.
	ErrNotFound = errors.New("symbol not found")
	// ErrAmbiguous indicates multiple matching symbols were discovered.
	ErrAmbiguous = errors.New("ambiguous symbol query")
	// ErrInvalidIdentifier indicates an identifier is empty or improperly formatted.
	ErrInvalidIdentifier = errors.New("invalid identifier")
)

// SymbolError represents a structured error condition when resolving or operating on symbols.
//
//nolint:revive // SymbolError stuttering is explicitly specified by ADR-0013 contract
type SymbolError struct {
	Op     string
	File   string
	Symbol string
	Pos    token.Position
	Err    error
}

func (e *SymbolError) Error() string {
	var prefix string
	if e.Pos.IsValid() {
		prefix = e.Pos.String() + ": "
	}
	opPart := e.Op
	if opPart != "" {
		opPart += " "
	}
	if e.File != "" && !e.Pos.IsValid() {
		return fmt.Sprintf("%s%s%s in %s: %v", prefix, opPart, e.Symbol, e.File, e.Err)
	}
	return fmt.Sprintf("%s%s%s: %v", prefix, opPart, e.Symbol, e.Err)
}

func (e *SymbolError) Unwrap() error { return e.Err }
