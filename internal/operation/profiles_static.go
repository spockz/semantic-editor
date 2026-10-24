// WHY: This file keeps each language's static capability notes together so the registry profile selector stays small.

package operation

import (
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/capability"
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
			{Title: "Bounded Promote Local to Const",
				Description: "Promotion accepts one explicitly typed immutable local with a range-valid decimal fixed-width integer literal when its only use is the sole direct value argument of the immediately following call. Macros, attributes, imports, comments, custom or shadowed types, extra uses, mutation, destructuring, aggregates, and any non-exact native edit refuse.",
				Severity:    "info"},
			{Title: "Trusted Explicit Workspace",
				Description: "Lookup requires a selected .rs file, a deterministic Cargo root, explicit workspace trust, and a preinstalled rust-analyzer. Rust-analyzer may run Cargo metadata while loading the project; the backend disables build scripts, procedural macros, and check-on-save.",
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
			{Title: "Trusted Explicit Workspace",
				Description: "Java edits and JDT LS reads require an explicit or unambiguous workspace root, explicit workspace trust, a preinstalled JDT LS distribution, and Java 21 or newer; JDT LS operations never invoke build tools. Bounded Maven process actions are exposed separately by the central operation registry.",
				Severity:    "error"},
			{Title: "Hierarchical Document Symbols",
				Description: "Hierarchical package, type, field, method, and constructor symbols are resolved; overloaded methods accept signature-qualified selectors. Locals, generated symbols, and malformed-source fallback are not promised.",
				Severity:    "info"},
		},
	}
}

func scalaProfile() Profile {
	return Profile{
		Language: backend.LanguageScala, DisplayName: "Scala", Maturity: "Bounded refactoring preview",
		Limitations: []capability.Constraint{
			{Title: "Trusted Explicit Tools",
				Description: "Lookup and extraction require a selected .scala file, an explicit workspace root for project markers, explicit workspace trust, a pinned preinstalled Metals distribution, and recorded Java 21 or newer; no build tool is invoked.",
				Severity:    "error"},
			{Title: "Bounded Refactorings",
				Description: "Extract-value accepts only integer literal arithmetic as the sole argument in a two-method object: a zero-argument expression-bodied Int owner calls one strict Int identity callee. Extract-callable accepts one or more contiguous direct-call statements to the same callee in a selected method and creates a local Unit helper. Inline-value remains unavailable until the configured Metals runtime returns the native command action and its applyEdit callback is verified. Other syntax, overloads, by-name parameters, and compiler diagnostics are unavailable.",
				Severity:    "info"},
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
			{Title: "Explicit Standalone Trust",
				Description: "Lookup requires --haskell-standalone (or standalone_haskell=true), a selected .hs file, explicit workspace trust, preinstalled GHC, and a matching preinstalled haskell-language-server-wrapper; hie.yaml, stack.yaml, cabal.project, *.cabal, and package.yaml project markers are rejected.",
				Severity:    "error"},
			{Title: "Hierarchical Document Symbols",
				Description: "Only module, top-level values, types, classes, constructors, fields, and instances returned hierarchically by HLS are resolved; locals, pattern synonyms, duplicate record fields, reexports, generated or Template Haskell symbols, and malformed-source fallback are not promised.",
				Severity:    "info"},
		},
	}
}
