// Package operation_test verifies backend-wired operation contracts without invoking language backends.
package operation_test

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"semedit/internal/backend"
	"semedit/internal/operation"
)

func TestJavaVerifyRegistryRequestAndDispatch(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupCLI("verify")
	if !ok {
		t.Fatal("verify registry entry missing")
	}
	request, err := entry.Parse(map[string]any{"file": "src/Widget.java", "language": "java", "trust_workspace": true, "jdtls_home": "/jdtls", "java_bin": "/java", "import_maven": true, "format_selected_file": true, "organize_imports": true})
	if err != nil {
		t.Fatal(err)
	}
	verify, ok := request.(operation.VerifyReq)
	if !ok || verify.Path != "src/Widget.java" || !verify.FormatSelectedFile || !verify.OrganizeImports || !verify.Project.Java.ImportMaven {
		t.Fatalf("parsed Java verify request = %#v", request)
	}
}

func TestAutomaticMutationDescriptionsDiscourageRedundantVerification(t *testing.T) {
	t.Parallel()

	registry := operation.DefaultRegistry()
	for _, name := range []string{
		"semantic_rename",
		"semantic_insert_declaration",
		"semantic_insert_function",
		"semantic_insert_type",
		"semantic_insert_decl",
		"semantic_insert_structure",
		"semantic_organize_imports",
		"semantic_replace_body",
	} {
		entry, ok := registry.LookupMCP(name)
		if !ok {
			t.Errorf("missing %s", name)
			continue
		}
		if !strings.Contains(entry.Summary, "do not call semantic_verify separately") {
			t.Errorf("%s does not discourage redundant verification: %q", name, entry.Summary)
		}
	}
}

func cloneExample(raw map[string]any) map[string]any {
	cloned := make(map[string]any, len(raw))
	maps.Copy(cloned, raw)
	return cloned
}

func TestRegisteredDefsHonorContracts(t *testing.T) {
	t.Parallel()

	registry := operation.DefaultRegistry()
	entries := registry.All()
	if len(entries) != 23 {
		t.Errorf("registered operations = %d, want 23", len(entries))
	}
	for _, entry := range entries {
		t.Run(entry.Key, func(t *testing.T) {
			t.Parallel()

			if entry.Key == "" {
				t.Error("operation key is empty")
			}
			if entry.Summary == "" {
				t.Error("operation summary is empty")
			}
			if entry.MCPName == "" {
				t.Error("operation MCP name is empty")
			}
			if len(entry.Params) == 0 {
				t.Error("operation params are empty")
			}
			if _, err := entry.Parse(entry.ExampleRaw); err != nil {
				t.Errorf("ExampleRaw parse failed: %v", err)
			}
			t.Run("cli-string-shapes", func(t *testing.T) {
				t.Parallel()

				shaped := cloneExample(entry.ExampleRaw)
				converted := false
				for _, param := range entry.Params {
					if param.Type != operation.ParamBoolean {
						continue
					}
					value, ok := shaped[param.JSONName]
					if !ok {
						continue
					}
					if flag, ok := value.(bool); ok {
						if flag {
							shaped[param.JSONName] = "true"
						} else {
							shaped[param.JSONName] = "false"
						}
						converted = true
					}
				}
				if !converted {
					t.Skip("no boolean parameter in ExampleRaw")
				}
				if _, err := entry.Parse(shaped); err != nil {
					t.Errorf("CLI string boolean parse failed: %v", err)
				}
			})
			for _, param := range entry.Params {
				if !param.Required {
					continue
				}
				t.Run("missing-required-"+param.JSONName, func(t *testing.T) {
					t.Parallel()

					without := cloneExample(entry.ExampleRaw)
					delete(without, param.JSONName)
					delete(without, param.CLIName)
					if _, err := entry.Parse(without); err == nil {
						t.Errorf("parse without required param %q succeeded, want error", param.JSONName)
					} else if !errors.Is(err, operation.ErrInvalidParams) {
						t.Errorf("missing required param error = %v, want ErrInvalidParams", err)
					}
				})
			}
			for _, param := range entry.Params {
				if len(param.Enums) == 0 {
					continue
				}
				t.Run("bad-enum-"+param.JSONName, func(t *testing.T) {
					t.Parallel()

					invalid := cloneExample(entry.ExampleRaw)
					invalid[param.JSONName] = "___not_a_value___"
					if _, err := entry.Parse(invalid); err == nil {
						t.Errorf("parse with bad enum %q succeeded, want error", param.JSONName)
					} else if !errors.Is(err, operation.ErrInvalidParams) {
						t.Errorf("bad enum error = %v, want ErrInvalidParams", err)
					}
				})
			}
		})
	}
}

func TestRegistryLookupsCoverWiredOperations(t *testing.T) {
	t.Parallel()

	registry := operation.DefaultRegistry()
	if _, ok := registry.LookupKey("lookup"); !ok {
		t.Error("LookupKey(lookup) missed")
	}
	if _, ok := registry.LookupMCP("semantic_lookup"); !ok {
		t.Error("LookupMCP(semantic_lookup) missed")
	}
	if _, ok := registry.LookupMCP("semantic_rename"); !ok {
		t.Error("LookupMCP(semantic_rename) missed")
	}
	if _, ok := registry.LookupMCP("semantic_verify"); !ok {
		t.Error("LookupMCP(semantic_verify) missed")
	}
	if _, ok := registry.LookupMCP("semantic_inspect_symbol"); !ok {
		t.Error("LookupMCP(semantic_inspect_symbol) missed")
	}
	if _, ok := registry.LookupMCP("semantic_outline"); !ok {
		t.Error("LookupMCP(semantic_outline) missed")
	}
	if _, ok := registry.LookupCLI("lookup"); !ok {
		t.Error("LookupCLI(lookup) missed")
	}
	if _, ok := registry.LookupCLI("rename"); !ok {
		t.Error("LookupCLI(rename) missed")
	}
	if _, ok := registry.LookupCLI("verify"); !ok {
		t.Error("LookupCLI(verify) missed")
	}
	if _, ok := registry.LookupCLI("inspect-symbol"); !ok {
		t.Error("LookupCLI(inspect-symbol) missed")
	}
	if _, ok := registry.LookupCLI("outline"); !ok {
		t.Error("LookupCLI(outline) missed")
	}
}

func TestMatrixDerivesSupportFromHandlers(t *testing.T) {
	t.Parallel()
	registry := operation.DefaultRegistry()
	matrix, err := registry.Matrix(backend.LanguageGo)
	if err != nil {
		t.Errorf("Matrix(go) failed: %v", err)
	}
	for _, key := range []string{"lookup", "rename", "verify", "find_references"} {
		capability, ok := matrix.Operations[key]
		if !ok {
			t.Errorf("go matrix misses operation %q", key)
			continue
		}
		if !capability.Supported {
			t.Errorf("go matrix marks %q unsupported", key)
		}
	}
	scala, err := registry.Matrix(backend.LanguageScala)
	if err != nil {
		t.Errorf("Matrix(scala) failed: %v", err)
	}
	if !scala.Operations["lookup"].Supported {
		t.Error("scala matrix marks lookup unsupported")
	}
	if scala.Operations["rename"].Supported {
		t.Error("scala matrix marks rename supported")
	}
	if scala.Operations["verify"].Supported {
		t.Error("scala matrix marks verify supported")
	}
	if _, err := registry.Matrix("cobol"); !errors.Is(err, operation.ErrUnsupportedLanguage) {
		t.Errorf("Matrix(cobol) error = %v, want ErrUnsupportedLanguage", err)
	}
}

func TestReadOperationsAreOnlyReadOnlyOperations(t *testing.T) {
	t.Parallel()

	for _, entry := range operation.DefaultRegistry().All() {
		want := entry.Key == "lookup" || entry.Key == "inspect_symbol" || entry.Key == "outline" || entry.Key == "find_references"
		if entry.ReadOnly != want {
			t.Errorf("operation %q ReadOnly = %v, want %v", entry.Key, entry.ReadOnly, want)
		}
		if entry.Key == "find_references" && entry.Batchable {
			t.Error("find_references must not be batchable")
		}
	}
}

func TestLookupSharedProjectParserPreservesBackendConfiguration(t *testing.T) {
	entry, ok := operation.DefaultRegistry().LookupCLI("lookup")
	if !ok {
		t.Fatal("lookup registry entry missing")
	}
	request, err := entry.Parse(map[string]any{
		"symbol":             "Widget",
		"file":               "src/Widget.java",
		"language":           "java",
		"trust_workspace":    true,
		"jdtls_home":         "/jdtls",
		"java_bin":           "/java",
		"import_maven":       true,
		"metals_home":        "/metals",
		"metals_bin":         "/metals/bin",
		"java_version":       "21",
		"standalone_haskell": true,
		"ghc_bin":            "/ghc",
		"hls_bin":            "/hls",
		"ghc_version":        "9.10",
		"hls_version":        "2.11",
		"kotlin_bin":         "/kotlin",
		"bash_bin":           "/bash",
		"make_bin":           "/make",
	})
	if err != nil {
		t.Fatal(err)
	}
	lookup, ok := request.(operation.LookupReq)
	if !ok {
		t.Fatalf("parsed request type = %T, want LookupReq", request)
	}
	want := backend.ProjectContext{
		File:              "src/Widget.java",
		Language:          backend.LanguageJava,
		WorkspaceTrust:    backend.WorkspaceTrust{Trusted: true},
		Java:              backend.JavaConfig{JDTLSHome: "/jdtls", JavaBin: "/java", ImportMaven: true},
		Scala:             backend.ScalaConfig{MetalsHome: "/metals", MetalsBin: "/metals/bin", JavaBin: "/java", JavaVersion: "21"},
		Haskell:           backend.HaskellConfig{Standalone: true, GHCBin: "/ghc", HLSBin: "/hls", GHCVersion: "9.10", HLSVersion: "2.11"},
		HaskellStandalone: true,
		Kotlin:            backend.KotlinConfig{KotlinBin: "/kotlin"},
		Bash:              backend.BashConfig{BashBin: "/bash"},
		Make:              backend.MakeConfig{MakeBin: "/make"},
		KotlinBin:         "/kotlin",
	}
	if !reflect.DeepEqual(lookup.Project, want) {
		t.Errorf("lookup project = %#v, want %#v", lookup.Project, want)
	}
	if lookup.Symbol != "Widget" {
		t.Errorf("lookup symbol = %q, want Widget", lookup.Symbol)
	}
}

func TestOutlineDirectoryDispatchUsesEffectiveRootBeforeLanguageSelection(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/outline\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "generated.java")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "source.go"), []byte("package sample\nfunc Ready() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, explicitLanguage := range []bool{false, true} {
		name := "auto language"
		raw := map[string]any{"path": "generated.java"}
		if explicitLanguage {
			name = "explicit language"
			raw["language"] = "go"
		}
		t.Run(name, func(t *testing.T) {
			cc := operation.NewCallContext(root, backend.ProjectContext{RootDir: root})
			result, err := cc.Registry.Dispatch(cc, "outline", raw)
			if err != nil {
				t.Fatal(err)
			}
			outline, ok := result.(*backend.OutlineResult)
			if !ok {
				t.Fatalf("result type = %T, want *backend.OutlineResult", result)
			}
			if outline.Scope.Path != "generated.java" || len(outline.Files) != 1 || outline.Files[0].File != "generated.java/source.go" {
				t.Fatalf("outline = %#v, want selected directory contents", outline)
			}
		})
	}
}

func TestFindReferencesRegistryAliases(t *testing.T) {
	registry := operation.DefaultRegistry()
	if _, ok := registry.LookupKey("find_references"); !ok {
		t.Fatal("LookupKey(find_references) missed")
	}
	if _, ok := registry.LookupMCP("semantic_find_references"); !ok {
		t.Fatal("LookupMCP(semantic_find_references) missed")
	}
	if _, ok := registry.LookupCLI("find-references"); !ok {
		t.Fatal("LookupCLI(find-references) missed")
	}
}
