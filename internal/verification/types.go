// Package verification owns project-plan execution and language-server diagnostics.
// WHY: Verification requires a cohesive plan, trust, normalization, and diagnostic lifecycle outside operation ingress.
package verification

import (
	"semedit/internal/backend"
	"semedit/internal/projectverify"
)

// Request carries typed project execution inputs from an ingress adapter.
type Request struct {
	Project              backend.ProjectContext
	Path                 string
	FormatSelectedFile   bool
	OrganizeImports      bool
	CheckOnly            bool
	SkipHooks            bool
	AllowedLanguages     []backend.LanguageID
	GoFormattingPaths    []string
	GoPackagePatterns    map[string][]string
	GoPackageScopeDetail string
	Service              *backend.Service
}

// Result carries final diagnostics and each completed verification phase.
type Result struct {
	Diagnostics     []backend.Diagnostic `json:"diagnostics,omitempty"`
	Formatted       bool                 `json:"formatted,omitempty"`
	NormalizeReport projectverify.Report `json:"normalize"`
	CheckReport     projectverify.Report `json:"checks"`
	Scope           projectverify.Scope  `json:"scope"`
	Failure         string               `json:"failure,omitempty"`
	Coverage        []LanguageCoverage   `json:"coverage,omitempty"`
}
