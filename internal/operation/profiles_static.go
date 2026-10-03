// WHY: This file keeps each language's static capability notes together so the registry profile selector stays small.

package operation

import (
	"semedit/internal/backend"
	"semedit/internal/capability"
)

func goProfile() Profile {
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
	}
}

func rustProfile() Profile {
	return Profile{
		Language: backend.LanguageRust, DisplayName: "Rust", Maturity: "Selected-file refactoring preview",
		Limitations: []capability.Constraint{
			{Title: "Selected-File Rename Only",
				Description: "Rust supports selected-file rename only; formatting, imports, verification, extraction, inline, move, and other structural edit/refactoring capabilities are unavailable.",
				Severity:    "error"},
			{Title: "Trusted Explicit Workspace",
				Description: "Lookup requires a selected .rs file, a deterministic Cargo root, explicit workspace trust, and a preinstalled rust-analyzer; Cargo is never invoked.",
				Severity:    "error"},
			{Title: "Hierarchical Document Symbols",
				Description: "Only exact hierarchical document symbols and associated items are resolved; macros, generated symbols, locals, ambiguous trait methods, and multiple impl resolution are not promised.",
				Severity:    "info"},
		},
	}
}

func javaProfile() Profile {
	return Profile{
		Language: backend.LanguageJava, DisplayName: "Java", Maturity: "Selected-file refactoring preview",
		Limitations: []capability.Constraint{
			{Title: "Selected-File Rename Only",
				Description: "Java supports selected-file rename only; formatting, imports, verification, extraction, inline, move, and hierarchy refactoring capabilities are unavailable.",
				Severity:    "error"},
			{Title: "Trusted Explicit Workspace",
				Description: "Lookup requires a selected .java file, an explicit or unambiguous Maven/Gradle root, explicit workspace trust, a preinstalled JDT LS distribution, and Java 21 or newer; JDT LS operations never invoke build tools. Bounded Maven process actions are exposed separately by the central operation registry.",
				Severity:    "error"},
			{Title: "Hierarchical Document Symbols",
				Description: "Only exact hierarchical package, type, field, method, and constructor document symbols are resolved; overload signatures, locals, generated symbols, and malformed-source fallback are not promised.",
				Severity:    "info"},
		},
	}
}

func scalaProfile() Profile {
	return Profile{
		Language: backend.LanguageScala, DisplayName: "Scala", Maturity: readOnlyPreview,
		Limitations: []capability.Constraint{
			{Title: "Lookup Only",
				Description: "Scala rename, formatting, imports, verification, build import, and structural edits are unavailable.",
				Severity:    "error"},
			{Title: "Trusted Explicit Tools",
				Description: "Lookup requires a selected .scala file, an explicit workspace root for project markers, explicit workspace trust, a pinned preinstalled Metals distribution, and recorded Java 21 or newer; no build tool is invoked.",
				Severity:    "error"},
			{Title: "Hierarchical Document Symbols",
				Description: "Only exact hierarchical classes, objects, traits, enums, methods, fields, and nested types are resolved; overload signatures, givens, extensions, package objects, generated symbols, and cross-file SemanticDB search are not promised.",
				Severity:    "info"},
		},
	}
}

func kotlinProfile() Profile {
	return Profile{
		Language: backend.LanguageKotlin, DisplayName: "Kotlin", Maturity: readOnlyPreview,
		SupportedModifiers: []string{"public", "private", "protected", "internal"},
		Limitations: []capability.Constraint{
			{Title: "Read Only",
				Description: "Kotlin supports trusted lookup and selected-file diagnostics only; rename, formatting, imports, and structural edits are unavailable.",
				Severity:    "error"},
			{Title: "Trusted Server",
				Description: "A selected .kt or .kts file and request-scoped workspace trust are required before the preinstalled fwcd/kotlin-language-server is resolved or launched. Upstream may execute project classpath scripts or Maven/Gradle while resolving dependencies.",
				Severity:    "error"},
			{Title: "Selected File Diagnostics",
				Description: "Verification requires a matching publishDiagnostics notification for the selected isolated file; no report before timeout is an error, not a clean result. Formatting and import modes are rejected.",
				Severity:    "info"},
		},
	}
}

func bashProfile() Profile {
	return Profile{
		Language: backend.LanguageBash, DisplayName: "Bash", Maturity: readOnlyPreview,
		Limitations: []capability.Constraint{
			{Title: "Read Only",
				Description: "Bash supports trusted selected-file lookup and diagnostics only; rename, formatting, imports, and structural edits are unavailable.",
				Severity:    "error"},
			{Title: "Selected File",
				Description: "Only .sh and .bash files are accepted. The server runs in a source-only scratch workspace after request-scoped workspace trust.",
				Severity:    "error"},
			{Title: "Optional Shell Tools",
				Description: "bash-language-server may invoke an installed ShellCheck against the copied selected source; the backend does not execute the shell script.",
				Severity:    "info"},
			{Title: "Diagnostics Receipt",
				Description: "Verification requires an explicit matching publishDiagnostics report. Silence and malformed reports are errors.",
				Severity:    "info"},
		},
	}
}

func makeProfile() Profile {
	return Profile{
		Language: backend.LanguageMake, DisplayName: "Makefile", Maturity: readOnlyPreview,
		Limitations: []capability.Constraint{
			{Title: "Read Only",
				Description: "Makefile supports trusted selected-file target and assignment-variable lookup only; conditional lookup, verification, rename, formatting, imports, and structural edits are unavailable.",
				Severity:    "error"},
			{Title: "Selected File",
				Description: "Only Makefile, makefile, GNUmakefile, and .mk files are accepted. make-ls runs in an isolated source workspace after request-scoped trust.",
				Severity:    "error"},
			{Title: "Include Reads",
				Description: "make-ls resolves includes; absolute paths and relative paths traversing outside the copied source may read files after trust is granted.",
				Severity:    "info"},
		},
	}
}

func haskellProfile() Profile {
	return Profile{
		Language: backend.LanguageHaskell, DisplayName: "Haskell", Maturity: readOnlyPreview,
		Limitations: []capability.Constraint{
			{Title: "Lookup Only",
				Description: "Haskell rename, formatting, imports, verification, compilation, diagnostics, and structural edits are unavailable.",
				Severity:    "error"},
			{Title: "Explicit Standalone Trust",
				Description: "Lookup requires --haskell-standalone (or standalone_haskell=true), a selected .hs file, explicit workspace trust, preinstalled GHC, and a matching preinstalled haskell-language-server-wrapper; hie.yaml, stack.yaml, cabal.project, *.cabal, and package.yaml project markers are rejected.",
				Severity:    "error"},
			{Title: "Hierarchical Document Symbols",
				Description: "Only module, top-level values, types, classes, constructors, fields, and instances returned hierarchically by HLS are resolved; locals, pattern synonyms, duplicate record fields, reexports, generated or Template Haskell symbols, and malformed-source fallback are not promised.",
				Severity:    "info"},
		},
	}
}
