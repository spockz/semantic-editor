// Package astedit tests cross-package relocation against real typed module fixtures because reference identity and import changes need compiler-backed evidence.
package astedit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMoveConstructAcrossPackagesRewritesTypedCallersAndImports(t *testing.T) {
	root := t.TempDir()
	writeMoveFixture(t, root, "go.mod", "module example.com/cross\n\ngo 1.23\n")
	writeMoveFixture(t, root, "legacy/move.go", `package legacy
import "strings"
// MoveMe trims a value.
func MoveMe(value string) string { return strings.TrimSpace(value) }
func Local() string { return MoveMe(" local ") }
`)
	writeMoveFixture(t, root, "api/api.go", `package api
import "example.com/cross/legacy"
func LocalCall() string { return legacy.MoveMe(" destination ") }
`)
	writeMoveFixture(t, root, "caller/caller.go", `package caller
import "example.com/cross/legacy"
func Call() string { return legacy.MoveMe(" caller ") }
`)
	writeMoveFixture(t, root, "legacy/unrelated_linux.go", "//go:build linux\n\npackage legacy\nfunc Unrelated() {}\n")

	diff, err := MoveConstruct(context.Background(), filepath.Join(root, "legacy/move.go"), "MoveMe", MoveConstructOptions{DestinationFile: filepath.Join(root, "api/api.go"), Placement: PlacementFileEnd})
	if err != nil {
		t.Fatalf("MoveConstruct() error = %v", err)
	}
	if !strings.Contains(diff, "caller/caller.go") {
		t.Fatalf("diff does not include third-package caller: %s", diff)
	}
	assertMoveFileContains(t, root, "api/api.go", "import (\n\t\"strings\"\n)", "func MoveMe(value string) string { return strings.TrimSpace(value) }", "func LocalCall() string { return MoveMe(\" destination \") }")
	assertMoveFileContains(t, root, "legacy/move.go", "example.com/cross/api", "api.MoveMe(\" local \")")
	assertMoveFileContains(t, root, "caller/caller.go", "example.com/cross/api", "api.MoveMe(\" caller \")")
}

func TestMoveConstructAcrossPackagesRefusesUnresolvedCasesBeforeWriting(t *testing.T) {
	tests := []struct {
		name, symbol, source, extraFile, wantError string
	}{
		{
			name:      "source package dependency",
			symbol:    "NeedsHelper",
			source:    "package legacy\nfunc helper() {}\nfunc NeedsHelper() { helper() }\n",
			wantError: "source-package dependency",
		},
		{
			name:      "inactive caller",
			symbol:    "HiddenTarget",
			source:    "package legacy\nfunc HiddenTarget() {}\n",
			extraFile: "//go:build never\n\npackage hidden\nimport \"example.com/cross/legacy\"\nfunc Hidden() { legacy.HiddenTarget() }\n",
			wantError: "possible reference to \"HiddenTarget\"",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeMoveFixture(t, root, "go.mod", "module example.com/cross\n\ngo 1.23\n")
			writeMoveFixture(t, root, "legacy/move.go", test.source)
			writeMoveFixture(t, root, "api/api.go", "package api\n")
			if test.extraFile != "" {
				writeMoveFixture(t, root, "hidden/hidden.go", test.extraFile)
			}
			beforeSource, err := os.ReadFile(filepath.Join(root, "legacy/move.go"))
			if err != nil {
				t.Fatal(err)
			}
			beforeDestination, err := os.ReadFile(filepath.Join(root, "api/api.go"))
			if err != nil {
				t.Fatal(err)
			}

			_, err = MoveConstruct(context.Background(), filepath.Join(root, "legacy/move.go"), test.symbol, MoveConstructOptions{DestinationFile: filepath.Join(root, "api/api.go"), Placement: PlacementFileEnd})
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("MoveConstruct() error = %v, want substring %q", err, test.wantError)
			}
			afterSource, err := os.ReadFile(filepath.Join(root, "legacy/move.go"))
			if err != nil {
				t.Fatal(err)
			}
			afterDestination, err := os.ReadFile(filepath.Join(root, "api/api.go"))
			if err != nil {
				t.Fatal(err)
			}
			if string(afterSource) != string(beforeSource) || string(afterDestination) != string(beforeDestination) {
				t.Fatal("refused move changed a source or destination file")
			}
		})
	}
}

func writeMoveFixture(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertMoveFileContains(t *testing.T, root, relative string, fragments ...string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range fragments {
		if !strings.Contains(string(content), fragment) {
			t.Errorf("%s does not contain %q:\n%s", relative, fragment, content)
		}
	}
}
