// Package makefile exposes trusted selected-file Makefile symbol lookup.
package makefile

import (
	"semedit/internal/backend"
	"semedit/internal/backend/textlsp"
)

// NewBackend constructs the Makefile selected-file backend.
func NewBackend() backend.Backend {
	return &textlsp.Adapter{Config: textlsp.Config{Language: backend.LanguageMake, Binary: "make-ls", Basenames: []string{"Makefile", "makefile", "GNUmakefile"}, Extensions: []string{".mk"}, LanguageID: "makefile", SymbolSyntax: "make"}}
}
