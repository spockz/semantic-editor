// Package operation_test verifies backend-wired operation contracts without invoking language backends.
package operation_test

import (
	"errors"
	"maps"
	"testing"

	"semedit/internal/backend"
	"semedit/internal/operation"
)

func cloneExample(raw map[string]any) map[string]any {
	cloned := make(map[string]any, len(raw))
	maps.Copy(cloned, raw)
	return cloned
}

func TestRegisteredDefsHonorContracts(t *testing.T) {
	t.Parallel()

	registry := operation.DefaultRegistry()
	entries := registry.All()
	if len(entries) != 15 {
		t.Errorf("registered operations = %d, want 15", len(entries))
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
	if _, ok := registry.LookupCLI("lookup"); !ok {
		t.Error("LookupCLI(lookup) missed")
	}
	if _, ok := registry.LookupCLI("rename"); !ok {
		t.Error("LookupCLI(rename) missed")
	}
	if _, ok := registry.LookupCLI("verify"); !ok {
		t.Error("LookupCLI(verify) missed")
	}
}

func TestMatrixDerivesSupportFromHandlers(t *testing.T) {
	t.Parallel()

	registry := operation.DefaultRegistry()
	matrix, err := registry.Matrix(backend.LanguageGo)
	if err != nil {
		t.Errorf("Matrix(go) failed: %v", err)
	}
	for _, key := range []string{"lookup", "rename", "verify"} {
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

func TestExactlyLookupIsReadOnly(t *testing.T) {
	t.Parallel()

	for _, entry := range operation.DefaultRegistry().All() {
		want := entry.Key == "lookup"
		if entry.ReadOnly != want {
			t.Errorf("operation %q ReadOnly = %v, want %v", entry.Key, entry.ReadOnly, want)
		}
	}
}
