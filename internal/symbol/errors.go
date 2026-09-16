// Package symbol provides AST symbol resolution and coordinate translation for Go source files.
package symbol

import (
	"errors"
	"fmt"
	"go/token"
	"strings"
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
	opPart := e.Op
	if opPart != "" {
		opPart += " "
	}
	errMsg := ""
	if e.Err != nil {
		errMsg = e.Err.Error()
		if e.Pos.IsValid() {
			prefix := e.Pos.String() + ": "
			if after, ok := strings.CutPrefix(errMsg, prefix); ok {
				errMsg = after
			}
		}
	}
	if e.File != "" && !e.Pos.IsValid() {
		return fmt.Sprintf("%s%s in %s: %s", opPart, e.Symbol, e.File, errMsg)
	}
	if e.Symbol != "" {
		return fmt.Sprintf("%s%s: %s", opPart, e.Symbol, errMsg)
	}
	return fmt.Sprintf("%s: %s", e.Op, errMsg)
}

func (e *SymbolError) Unwrap() error { return e.Err }
