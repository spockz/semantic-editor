// Package operation derives per-language capability views from registered operations and static profiles.
package operation

import (
	"fmt"
	"slices"

	"semedit/internal/astedit"
	"semedit/internal/backend"
	"semedit/internal/capability"
)

// Profile carries the static language metadata backing a capability matrix view.
type Profile struct {
	Language           backend.LanguageID
	DisplayName        string
	Maturity           string
	SupportedModifiers []string
	Limitations        []capability.Constraint
}

func goModifiers() []string {
	modifiers := make([]string, 0, 3)
	for _, modifier := range (astedit.GolangBackend{}).SupportedAccessModifiers() {
		modifiers = append(modifiers, string(modifier))
	}
	return modifiers
}

func profileFor(language backend.LanguageID) (Profile, bool) {
	switch language {
	case backend.LanguageGo:
		return Profile{
			Language:           backend.LanguageGo,
			DisplayName:        "Go (Golang)",
			Maturity:           "Production",
			SupportedModifiers: goModifiers(),
			Limitations: []capability.Constraint{
				{
					Title:       "Unsupported Modifiers Rejection",
					Description: "Go lacks 'protected' and 'package-private' scopes. The engine rejects these modifiers with ErrUnsupportedModifier.",
					Severity:    "error",
				},
				{
					Title:       "Casing & Visibility Invariant",
					Description: "Identifier capitalization governs visibility. Specifying 'public' for a lowercase symbol or 'private' for an uppercase symbol returns VisibilityMismatchError.",
					Severity:    "error",
				},
				{
					Title:       "Strict Public-Precedes-Private Ordering",
					Description: "All public declarations precede private declarations within generated or updated source files.",
					Severity:    "info",
				},
				{
					Title:       "Receiver Method Clustering",
					Description: "Methods sharing a common receiver type cluster near each other while maintaining public vs private partitioning.",
					Severity:    "info",
				},
				{
					Title:       "Declaration Block Merging",
					Description: "Constants and variables automatically merge into existing 'const (...)' or 'var (...)' blocks instead of creating duplicate blocks.",
					Severity:    "info",
				},
			},
		}, true
	case backend.LanguageRust:
		return Profile{
			Language:    backend.LanguageRust,
			DisplayName: "Rust",
			Maturity:    "Selected-file refactoring preview",
			Limitations: []capability.Constraint{
				{
					Title:       "Selected-File Rename Only",
					Description: "Rust supports selected-file rename only; formatting, imports, verification, extraction, inline, move, and other structural edit/refactoring capabilities are unavailable.",
					Severity:    "error",
				},
				{
					Title:       "Trusted Explicit Workspace",
					Description: "Lookup requires a selected .rs file, a deterministic Cargo root, explicit workspace trust, and a preinstalled rust-analyzer; Cargo is never invoked.",
					Severity:    "error",
				},
				{
					Title:       "Hierarchical Document Symbols",
					Description: "Only exact hierarchical document symbols and associated items are resolved; macros, generated symbols, locals, ambiguous trait methods, and multiple impl resolution are not promised.",
					Severity:    "info",
				},
			},
		}, true
	case backend.LanguageJava:
		return Profile{
			Language:    backend.LanguageJava,
			DisplayName: "Java",
			Maturity:    "Selected-file refactoring preview",
			Limitations: []capability.Constraint{
				{
					Title:       "Selected-File Rename Only",
					Description: "Java supports selected-file rename only; formatting, imports, verification, extraction, inline, move, and hierarchy refactoring capabilities are unavailable.",
					Severity:    "error",
				},
				{
					Title:       "Trusted Explicit Workspace",
					Description: "Lookup requires a selected .java file, an explicit or unambiguous Maven/Gradle root, explicit workspace trust, a preinstalled JDT LS distribution, and Java 21 or newer; build tools are never invoked.",
					Severity:    "error",
				},
				{
					Title:       "Hierarchical Document Symbols",
					Description: "Only exact hierarchical package, type, field, method, and constructor document symbols are resolved; overload signatures, locals, generated symbols, and malformed-source fallback are not promised.",
					Severity:    "info",
				},
			},
		}, true
	case backend.LanguageScala:
		return Profile{
			Language:    backend.LanguageScala,
			DisplayName: "Scala",
			Maturity:    "Read-only preview",
			Limitations: []capability.Constraint{
				{
					Title:       "Lookup Only",
					Description: "Scala rename, formatting, imports, verification, build import, and structural edits are unavailable.",
					Severity:    "error",
				},
				{
					Title:       "Trusted Explicit Tools",
					Description: "Lookup requires a selected .scala file, an explicit workspace root for project markers, explicit workspace trust, a pinned preinstalled Metals distribution, and recorded Java 21 or newer; no build tool is invoked.",
					Severity:    "error",
				},
				{
					Title:       "Hierarchical Document Symbols",
					Description: "Only exact hierarchical classes, objects, traits, enums, methods, fields, and nested types are resolved; overload signatures, givens, extensions, package objects, generated symbols, and cross-file SemanticDB search are not promised.",
					Severity:    "info",
				},
			},
		}, true
	case backend.LanguageHaskell:
		return Profile{
			Language:    backend.LanguageHaskell,
			DisplayName: "Haskell",
			Maturity:    "Read-only preview",
			Limitations: []capability.Constraint{
				{
					Title:       "Lookup Only",
					Description: "Haskell rename, formatting, imports, verification, compilation, diagnostics, and structural edits are unavailable.",
					Severity:    "error",
				},
				{
					Title:       "Explicit Standalone Trust",
					Description: "Lookup requires --haskell-standalone (or standalone_haskell=true), a selected .hs file, explicit workspace trust, preinstalled GHC, and a matching preinstalled haskell-language-server-wrapper; hie.yaml, stack.yaml, cabal.project, *.cabal, and package.yaml project markers are rejected.",
					Severity:    "error",
				},
				{
					Title:       "Hierarchical Document Symbols",
					Description: "Only module, top-level values, types, classes, constructors, fields, and instances returned hierarchically by HLS are resolved; locals, pattern synonyms, duplicate record fields, reexports, generated or Template Haskell symbols, and malformed-source fallback are not promised.",
					Severity:    "info",
				},
			},
		}, true
	default:
		return Profile{}, false
	}
}

// Matrix builds the capability view for language from registered operations and its static profile.
// Operation support derives from handler presence; descriptions, commands, and tools derive from defs.
func (r *Registry) Matrix(language backend.LanguageID) (capability.LanguageMatrix, error) {
	profile, ok := profileFor(language)
	if !ok {
		return capability.LanguageMatrix{}, fmt.Errorf("capability matrix for language %q: %w", language, ErrUnsupportedLanguage)
	}
	operations := make(map[string]capability.OpCapability, len(r.byKey))
	for _, entry := range r.All() {
		supported := slices.Contains(entry.Languages, language)
		command := ""
		if entry.CLIName != "" {
			command = "semedit " + entry.CLIName
		}
		operations[entry.Key] = capability.OpCapability{
			Supported:    supported,
			Description:  entry.Summary,
			CLICommand:   command,
			MCPTool:      entry.MCPName,
			PlacementKey: entry.PlacementKey,
			Level:        string(entry.Level),
			ReadOnly:     entry.ReadOnly,
		}
	}
	return capability.LanguageMatrix{
		Language:           string(language),
		DisplayName:        profile.DisplayName,
		Maturity:           profile.Maturity,
		SupportedModifiers: profile.SupportedModifiers,
		Operations:         operations,
		Limitations:        profile.Limitations,
	}, nil
}
