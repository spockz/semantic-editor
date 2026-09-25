// Package bash exposes trusted selected-file Bash lookup and diagnostics.
package bash

import (
	"semedit/internal/backend"
	"semedit/internal/backend/textlsp"
)

// NewBackend constructs the Bash selected-file backend.
func NewBackend() backend.Backend {
	return &textlsp.Adapter{Config: textlsp.Config{Language: backend.LanguageBash, Binary: "bash-language-server", Args: []string{"start"}, Extensions: []string{".sh", ".bash"}, LanguageID: "shell", Diagnostics: true, SymbolSyntax: "bash"}}
}
