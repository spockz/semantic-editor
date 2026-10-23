// Package astedit tests cross-package relocation against real typed module fixtures because reference identity and import changes need compiler-backed evidence.
package astedit

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMoveConstructAcrossPackagesRewritesTypedCallersAndImports(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeMoveFixture(t, root, "go.mod", "module example.com/cross\n\ngo 1.23\n")
	writeMoveFixture(t, root, "legacy/move.go", `package legacy
import "strings"
// MoveMe trims a value.
func MoveMe(value string) string { return strings.TrimSpace(value) }
func Local() string { return MoveMe(" local ") }
`)
	writeMoveFixture(t, root, "legacy/move_test.go", `package legacy
func testMoveMeReference() string { return MoveMe(" test ") }
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
	assertMoveFileContains(t, root, "legacy/move_test.go", "example.com/cross/api", "api.MoveMe(\" test \")")
	assertMoveFileContains(t, root, "caller/caller.go", "example.com/cross/api", "api.MoveMe(\" caller \")")
}

func TestMoveConstructAcrossPackagesMovesScalarConstantWithExactTypeAndValue(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeMoveFixture(t, root, "go.mod", "module example.com/constantmove\n\ngo 1.23\n")
	writeMoveFixture(t, root, "legacy/limits.go", `package legacy

const RetryWindow int64 = 180

func Local() int64 { return RetryWindow }
`)
	writeMoveFixture(t, root, "api/limits.go", "package api\n")
	writeMoveFixture(t, root, "caller/use.go", `package caller

import "example.com/constantmove/legacy"

func Use() int64 { return legacy.RetryWindow }
`)

	_, err := MoveConstruct(context.Background(), filepath.Join(root, "legacy/limits.go"), "RetryWindow", MoveConstructOptions{DestinationFile: filepath.Join(root, "api/limits.go"), Placement: PlacementFileEnd})
	if err != nil {
		t.Fatalf("MoveConstruct() error = %v", err)
	}
	assertMoveFileContains(t, root, "legacy/limits.go", "example.com/constantmove/api", "func Local() int64 { return api.RetryWindow }")
	assertMoveFileContains(t, root, "api/limits.go", "const RetryWindow int64 = 180")
	assertMoveFileContains(t, root, "caller/use.go", "example.com/constantmove/api", "func Use() int64 { return api.RetryWindow }")
}

func TestMoveConstructAcrossPackagesMovesScalarVariableAndPreservesReferences(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeMoveFixture(t, root, "go.mod", "module example.com/variablecross\n\ngo 1.23\n")
	writeMoveFixture(t, root, "legacy/state.go", `package legacy

var RetryWindow int = 180
func Local() int { return RetryWindow }
`)
	writeMoveFixture(t, root, "api/state.go", `package api

import "example.com/variablecross/legacy"

func Existing() int { return legacy.RetryWindow }
func Address() *int { return &legacy.RetryWindow }
`)
	writeMoveFixture(t, root, "caller/use.go", `package caller

import "example.com/variablecross/legacy"

func Read() int { return legacy.RetryWindow }
func Write(value int) { legacy.RetryWindow = value }
`)
	sourcePath := filepath.Join(root, "legacy/state.go")
	destinationPath := filepath.Join(root, "api/state.go")
	_, err := MoveConstruct(context.Background(), sourcePath, "RetryWindow", MoveConstructOptions{DestinationFile: destinationPath, Placement: PlacementFileEnd})
	if err != nil {
		t.Fatalf("MoveConstruct() error = %v", err)
	}
	assertMoveFileContains(t, root, "legacy/state.go", `"example.com/variablecross/api"`, `return api.RetryWindow`)
	assertMoveFileContains(t, root, "api/state.go", `func Existing() int { return RetryWindow }`, `func Address() *int { return &RetryWindow }`, `var RetryWindow int = 180`)
	assertMoveFileContains(t, root, "caller/use.go", `"example.com/variablecross/api"`, `api.RetryWindow = value`)
}

func TestMoveConstructAcrossPackagesMovesZeroInitializedScalarVariable(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeMoveFixture(t, root, "go.mod", "module example.com/zerovariable\n\ngo 1.23\n")
	writeMoveFixture(t, root, "legacy/state.go", `package legacy

var RetryWindow int
`)
	writeMoveFixture(t, root, "api/state.go", "package api\n")
	writeMoveFixture(t, root, "caller/use.go", `package caller

import "example.com/zerovariable/legacy"
func Use() int { return legacy.RetryWindow }
`)
	sourcePath := filepath.Join(root, "legacy/state.go")
	destinationPath := filepath.Join(root, "api/state.go")
	_, err := MoveConstruct(context.Background(), sourcePath, "RetryWindow", MoveConstructOptions{DestinationFile: destinationPath, Placement: PlacementFileEnd})
	if err != nil {
		t.Fatalf("MoveConstruct() error = %v", err)
	}
	assertMoveFileContains(t, root, "api/state.go", "var RetryWindow int")
	assertMoveFileContains(t, root, "caller/use.go", "example.com/zerovariable/api", "api.RetryWindow")
}

type crossPackageScalarVariableRefusal struct {
	name, source, destination, extraFile, extraSource, wantError string
}

func TestMoveConstructAcrossPackagesRefusesUnsafeScalarVariablesWithoutWriting(t *testing.T) {
	t.Parallel()
	tests := []crossPackageScalarVariableRefusal{
		{
			name: "dynamic initializer",
			source: `package legacy
func load() int { return 1 }
var RetryWindow = load()
`,
			wantError: "compile-time constant initializer",
		},
		{
			name: "named scalar type",
			source: `package legacy
type Duration int64
var RetryWindow Duration = 180
`,
			wantError: "predeclared basic scalar types",
		},
		{
			name:        "test reference",
			source:      "package legacy\nvar RetryWindow = 180\n",
			extraFile:   "legacy/state_test.go",
			extraSource: "package legacy\nfunc testState() int { return RetryWindow }\n",
			wantError:   "test reference",
		},
		{
			name:        "inactive platform source",
			source:      "package legacy\nvar RetryWindow = 180\n",
			extraFile:   "caller/use_windows.go",
			extraSource: "package caller\nvar _ = RetryWindow\n",
			wantError:   "build-constrained file",
		},
		{
			name:        "changed init graph",
			source:      "package legacy\nvar RetryWindow = 180\n",
			extraFile:   "caller/init.go",
			extraSource: "package caller\nfunc init() {}\n",
			wantError:   "initialization function",
		},
		{
			name:   "captured destination reference",
			source: "package legacy\nvar RetryWindow = 180\n",
			destination: `package api
import "example.com/variablecross/legacy"
func Existing() int {
	RetryWindow := 999
	_ = RetryWindow
	return legacy.RetryWindow
}
`,
			wantError: "different declaration",
		},
		{
			name:   "import cycle",
			source: "package legacy\nvar RetryWindow = 180\nfunc Other() int { return 1 }\nfunc Local() int { return RetryWindow }\n",
			destination: `package api
import "example.com/variablecross/legacy"
func Existing() int { return legacy.Other() }
`,
			wantError: "import cycle",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assertCrossPackageScalarVariableMoveRefusal(t, test)
		})
	}
}

func snapshotMoveFixture(t *testing.T, root string) map[string]string {
	t.Helper()
	files := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[relative] = string(content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func assertMoveFixtureUnchanged(t *testing.T, root string, before map[string]string) {
	t.Helper()
	after := snapshotMoveFixture(t, root)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("refused move changed fixture files:\nbefore: %#v\nafter: %#v", before, after)
	}
}

func TestMoveConstructAcrossPackagesRefusesUnresolvedCasesBeforeWriting(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, symbol, source, extraFilePath, extraFile, destination, wantError string
	}{
		{
			name:      "source package dependency",
			symbol:    "NeedsHelper",
			source:    "package legacy\nfunc helper() {}\nfunc NeedsHelper() { helper() }\n",
			wantError: "source-package dependency",
		},
		{
			name:          "inactive caller",
			symbol:        "HiddenTarget",
			source:        "package legacy\nfunc HiddenTarget() {}\n",
			extraFilePath: "hidden/hidden.go",
			extraFile:     "//go:build never\n\npackage hidden\nimport \"example.com/cross/legacy\"\nfunc Hidden() { legacy.HiddenTarget() }\n",
			wantError:     "possible reference to \"HiddenTarget\"",
		},
		{
			name:        "shadowed destination reference",
			symbol:      "ShadowTarget",
			source:      "package legacy\nfunc ShadowTarget() string { return \"target\" }\n",
			destination: "package api\nimport \"example.com/cross/legacy\"\nfunc Call() string { ShadowTarget := func() string { return \"local\" }; _ = ShadowTarget; return legacy.ShadowTarget() }\n",
			wantError:   "resolves to a different declaration",
		},
		{
			name:          "shadowed import alias reference",
			symbol:        "ShadowAliasTarget",
			source:        "package legacy\nfunc ShadowAliasTarget() string { return \"target\" }\n",
			destination:   "package api\nfunc LocalCall() string { return \"api\" }\n",
			extraFilePath: "caller/caller.go",
			extraFile:     "package caller\nimport (legacy \"example.com/cross/legacy\"; moved \"example.com/cross/api\")\nfunc Existing() string { return moved.LocalCall() }\nfunc Call() string { moved := struct{ ShadowAliasTarget func() string }{ShadowAliasTarget: func() string { return \"local\" }}; _ = moved.ShadowAliasTarget; return legacy.ShadowAliasTarget() }\n",
			wantError:     "resolves to a different declaration",
		},
		{
			name:          "reference in ignored Go directory",
			symbol:        "UnlistedTarget",
			source:        "package legacy\nfunc UnlistedTarget() {}\n",
			extraFilePath: "_private/reference.go",
			extraFile:     "package hidden\nimport \"example.com/cross/legacy\"\nfunc Hidden() { legacy.UnlistedTarget() }\n",
			wantError:     "untyped Go file",
		},
		{
			name:          "reference in testdata",
			symbol:        "TestdataTarget",
			source:        "package legacy\nfunc TestdataTarget() {}\n",
			extraFilePath: "testdata/reference.go",
			extraFile:     "package hidden\nimport \"example.com/cross/legacy\"\nfunc Hidden() { legacy.TestdataTarget() }\n",
			wantError:     "untyped Go file",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertUnresolvedMoveIsAtomic(t, test)
		})
	}
}

func assertUnresolvedMoveIsAtomic(t *testing.T, test struct {
	name, symbol, source, extraFilePath, extraFile, destination, wantError string
}) {
	t.Helper()
	root := t.TempDir()
	writeMoveFixture(t, root, "go.mod", "module example.com/cross\n\ngo 1.23\n")
	writeMoveFixture(t, root, "legacy/move.go", test.source)
	destination := test.destination
	if destination == "" {
		destination = "package api\n"
	}
	writeMoveFixture(t, root, "api/api.go", destination)
	if test.extraFile != "" {
		writeMoveFixture(t, root, test.extraFilePath, test.extraFile)
	}
	sourcePath := filepath.Join(root, "legacy/move.go")
	destinationPath := filepath.Join(root, "api/api.go")
	beforeSource, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	beforeDestination, err := os.ReadFile(destinationPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = MoveConstruct(context.Background(), sourcePath, test.symbol, MoveConstructOptions{DestinationFile: destinationPath, Placement: PlacementFileEnd})
	if err == nil || !strings.Contains(err.Error(), test.wantError) {
		t.Fatalf("MoveConstruct() error = %v, want substring %q", err, test.wantError)
	}
	assertMoveSourceFilesUnchanged(t, sourcePath, destinationPath, beforeSource, beforeDestination)
}

func assertMoveSourceFilesUnchanged(t *testing.T, sourcePath, destinationPath string, beforeSource, beforeDestination []byte) {
	t.Helper()
	afterSource, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	afterDestination, err := os.ReadFile(destinationPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterSource) != string(beforeSource) || string(afterDestination) != string(beforeDestination) {
		t.Fatal("refused move changed a source or destination file")
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

func TestMoveConstructAcrossPackagesMovesDefinedTypeAndDirectMethods(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeMoveFixture(t, root, "go.mod", "module example.com/typecross\n\ngo 1.23\n")
	writeMoveFixture(t, root, "legacy/widget.go", `package legacy

type Widget struct { Value int }
func (Widget) ValueMethod() int { return 1 }
func (_ Widget) Copy(value int) int { _ = value; local := value; return local }
func (Widget) AnonymousField() int {
	value := struct{ Outer struct{ X int } }{Outer: struct{ X int }{X: 3}}
	return value.Outer.X
}
func (w *Widget) PointerMethod() int { return w.Value }
func Local() int { return Widget{}.ValueMethod() }
func LocalMethodExpression() func(Widget) int { return Widget.ValueMethod }
`)
	writeMoveFixture(t, root, "api/api.go", "package api\n")
	writeMoveFixture(t, root, "caller/use.go", `package caller

import "example.com/typecross/legacy"

type valueAPI interface { ValueMethod() int }
type pointerAPI interface { PointerMethod() int }

func verifyInterfaces() {
	var _ valueAPI = legacy.Widget{}
	var _ pointerAPI = (*legacy.Widget)(nil)
}

func Call(w *legacy.Widget) int { return w.PointerMethod() }
func MethodValue(w *legacy.Widget) func() int { return w.PointerMethod }
func MethodExpression() func(*legacy.Widget) int { return (*legacy.Widget).PointerMethod }
`)
	sourcePath := filepath.Join(root, "legacy/widget.go")
	destinationPath := filepath.Join(root, "api/api.go")
	_, err := MoveConstruct(context.Background(), sourcePath, "Widget", MoveConstructOptions{DestinationFile: destinationPath, Placement: PlacementFileEnd})
	if err != nil {
		t.Fatalf("MoveConstruct() error = %v", err)
	}
	assertMoveFileContains(t, root, "api/api.go", "type Widget struct", "func (Widget) ValueMethod()", "func (_ Widget) Copy(value int)", "func (Widget) AnonymousField()", "func (w *Widget) PointerMethod()")
	assertMoveFileContains(t, root, "legacy/widget.go", "example.com/typecross/api", "api.Widget{}.ValueMethod()", "func LocalMethodExpression() func(api.Widget) int {\n\treturn api.Widget.ValueMethod\n}")
	assertMoveFileContains(t, root, "caller/use.go", "example.com/typecross/api", "api.Widget{}", "(*api.Widget).PointerMethod", "w.PointerMethod")
}

func assertCrossPackageScalarVariableMoveRefusal(t *testing.T, test crossPackageScalarVariableRefusal) {
	t.Helper()
	root := t.TempDir()
	writeMoveFixture(t, root, "go.mod", "module example.com/variablecross\n\ngo 1.23\n")
	writeMoveFixture(t, root, "legacy/state.go", test.source)
	destination := test.destination
	if destination == "" {
		destination = "package api\n"
	}
	writeMoveFixture(t, root, "api/state.go", destination)
	writeMoveFixture(t, root, "caller/use.go", `package caller

import "example.com/variablecross/legacy"
func Use() any { return legacy.RetryWindow }
`)
	if test.extraFile != "" {
		writeMoveFixture(t, root, test.extraFile, test.extraSource)
	}
	sourcePath := filepath.Join(root, "legacy/state.go")
	destinationPath := filepath.Join(root, "api/state.go")
	before := snapshotMoveFixture(t, root)
	_, err := MoveConstruct(context.Background(), sourcePath, "RetryWindow", MoveConstructOptions{DestinationFile: destinationPath, Placement: PlacementFileEnd})
	if err == nil || !strings.Contains(err.Error(), test.wantError) {
		t.Fatalf("MoveConstruct() error = %v, want substring %q", err, test.wantError)
	}
	assertMoveFixtureUnchanged(t, root, before)
}
