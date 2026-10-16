// Package astedit tests initialization-edge preservation because typechecking alone cannot detect reordered package initialization effects.
package astedit

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestMoveFileInitializationRefusesRemainingTestOrderChange(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		withCompanion bool
		testPackage   string
	}{
		{"source-without-companion", false, "aa"},
		{"destination-without-companion", false, "z"},
		{"source-with-companion", true, "aa"},
		{"destination-with-companion", true, "z"},
	} {
		withCompanion := tc.withCompanion
		name := tc.name
		t.Run(name, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			writeMoveFixture(t, root, "go.mod", "module example.com/initmove\n\ngo 1.24\n")
			writeMoveFixture(t, root, "aa/value.go", "package aa\nfunc Value() int { return 1 }\n")
			writeMoveFixture(t, root, "aa/keep.go", "package aa\n")
			writeMoveFixture(t, root, "z/value.go", "package z\n")
			writeMoveFixture(t, root, "ab/ab.go", "package ab\nimport \"example.com/initmove/aa\"\nvar X = aa.Value()\n")
			writeMoveFixture(t, root, "b/b.go", "package b\nvar X = 1\n")
			writeMoveFixture(t, root, tc.testPackage+"/keep_test.go", "package "+tc.testPackage+"_test\nimport (\n\t\"testing\"\n\t\"example.com/initmove/ab\"\n\t\"example.com/initmove/b\"\n)\nfunc TestValues(t *testing.T) { _, _ = ab.X, b.X }\n")
			if withCompanion {
				writeMoveFixture(t, root, "aa/value_test.go", "package aa_test\nimport \"testing\"\nfunc TestMoved(t *testing.T) {}\n")
			}
			source, destination := filepath.Join(root, "aa/value.go"), filepath.Join(root, "z/value.go")
			beforeSource, err := os.ReadFile(source)
			if err != nil {
				t.Fatal(err)
			}
			beforeDestination, err := os.ReadFile(destination)
			if err != nil {
				t.Fatal(err)
			}
			_, err = MoveFile(context.Background(), source, destination, MoveFileOptions{MoveMatchingTests: withCompanion})
			assertMoveFileRefusalUnchanged(t, source, destination, beforeSource, beforeDestination, err, "initialization-changing", "remaining-test initialization refusal")
		})
	}
}

func TestMoveFileInitializationRefusesCombinedTestVariantOrderChange(t *testing.T) {
	t.Parallel()
	for _, testPackage := range []string{"aa", "q"} {
		t.Run(testPackage, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			writeMoveFixture(t, root, "go.mod", "module example.com/initmove\n\ngo 1.24\n")
			writeMoveFixture(t, root, "aa/value.go", "package aa\nfunc Value() int { return 1 }\n")
			writeMoveFixture(t, root, "aa/keep.go", "package aa\n")
			writeMoveFixture(t, root, "z/value.go", "package z\n")
			writeMoveFixture(t, root, "ab/ab.go", "package ab\nimport \"example.com/initmove/aa\"\nvar X = aa.Value()\n")
			writeMoveFixture(t, root, "b/b.go", "package b\nvar X = 1\n")
			if testPackage == "q" {
				writeMoveFixture(t, root, "q/q.go", "package q\n")
			}
			writeMoveFixture(t, root, testPackage+"/internal_test.go", "package "+testPackage+"\nimport (\n\t\"testing\"\n\t\"example.com/initmove/b\"\n)\nfunc TestInternal(t *testing.T) { _ = b.X }\n")
			writeMoveFixture(t, root, testPackage+"/external_test.go", "package "+testPackage+"_test\nimport (\n\t\"testing\"\n\t\"example.com/initmove/ab\"\n)\nfunc TestExternal(t *testing.T) { _ = ab.X }\n")
			source, destination := filepath.Join(root, "aa/value.go"), filepath.Join(root, "z/value.go")
			beforeSource, err := os.ReadFile(source)
			if err != nil {
				t.Fatal(err)
			}
			beforeDestination, err := os.ReadFile(destination)
			if err != nil {
				t.Fatal(err)
			}
			_, err = MoveFile(context.Background(), source, destination, MoveFileOptions{})
			assertMoveFileRefusalUnchanged(t, source, destination, beforeSource, beforeDestination, err, "initialization-changing", "combined-test initialization refusal")
		})
	}
}

func TestMoveFileInitializationRefusesCompanionMergeOrderChange(t *testing.T) {
	t.Parallel()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeMoveFixture(t, root, "go.mod", "module example.com/initmove\n\ngo 1.24\n")
	writeMoveFixture(t, root, "aa/value.go", "package aa\nfunc Value() int { return 1 }\n")
	writeMoveFixture(t, root, "aa/keep.go", "package aa\n")
	writeMoveFixture(t, root, "z/value.go", "package z\n")
	writeMoveFixture(t, root, "ab/ab.go", "package ab\nimport \"example.com/initmove/aa\"\nvar X = aa.Value()\n")
	writeMoveFixture(t, root, "b/b.go", "package b\nvar X = 1\n")
	writeMoveFixture(t, root, "aa/value_test.go", "package aa_test\nimport (\n\t\"testing\"\n\t\"example.com/initmove/ab\"\n)\nfunc TestMoved(t *testing.T) { _ = ab.X }\n")
	writeMoveFixture(t, root, "z/keep_test.go", "package z_test\nimport (\n\t\"testing\"\n\t\"example.com/initmove/b\"\n)\nfunc TestRemaining(t *testing.T) { _ = b.X }\n")
	source, destination := filepath.Join(root, "aa/value.go"), filepath.Join(root, "z/value.go")
	beforeSource, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	beforeDestination, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	_, err = MoveFile(context.Background(), source, destination, MoveFileOptions{MoveMatchingTests: true})
	assertMoveFileRefusalUnchanged(t, source, destination, beforeSource, beforeDestination, err, "initialization-changing", "companion merge initialization refusal")
}

func TestMoveFileInitializationAllowsEquivalentCallerEffects(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeMoveFixture(t, root, "go.mod", "module example.com/initmove\n\ngo 1.24\n")
	writeMoveFixture(t, root, "old/value.go", "package old\nimport \"strings\"\nfunc Value(s string) string { return strings.TrimSpace(s) }\n")
	writeMoveFixture(t, root, "old/keep.go", "package old\nimport \"strings\"\nfunc Keep(s string) string { return strings.TrimSpace(s) }\n")
	writeMoveFixture(t, root, "api/value.go", "package api\n")
	writeMoveFixture(t, root, "api/keep.go", "package api\nimport \"strings\"\nfunc Keep(s string) string { return strings.TrimSpace(s) }\n")
	writeMoveFixture(t, root, "client/client.go", "package client\nimport \"example.com/initmove/old\"\nfunc Read(s string) string { return old.Value(s) }\n")
	before, after := loadMoveFileInitializationPackages(t, root, map[string][]byte{
		filepath.Join(root, "old/value.go"):     []byte("package old\n"),
		filepath.Join(root, "api/value.go"):     []byte("package api\nimport \"strings\"\nfunc Value(s string) string { return strings.TrimSpace(s) }\n"),
		filepath.Join(root, "client/client.go"): []byte("package client\nimport \"example.com/initmove/api\"\nfunc Read(s string) string { return api.Value(s) }\n"),
	})
	if err := validateMoveFileInitialization(before, after, filepath.Join(root, "old/value.go"), filepath.Join(root, "api/value.go"), "", ""); err != nil {
		t.Fatalf("validateMoveFileInitialization() error = %v", err)
	}
}

func TestMoveFileInitializationRefusesChangedEffectfulImportClosure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeMoveFixture(t, root, "go.mod", "module example.com/initmove\n\ngo 1.24\n")
	writeMoveFixture(t, root, "effects/effects.go", "package effects\nvar Value = record()\nfunc record() int { return 1 }\n")
	writeMoveFixture(t, root, "old/value.go", "package old\nimport \"example.com/initmove/effects\"\nfunc Value() int { return effects.Value }\n")
	writeMoveFixture(t, root, "api/value.go", "package api\n")
	before, after := loadMoveFileInitializationPackages(t, root, map[string][]byte{
		filepath.Join(root, "old/value.go"): []byte("package old\n"),
		filepath.Join(root, "api/value.go"): []byte("package api\nimport \"example.com/initmove/effects\"\nfunc Value() int { return effects.Value }\n"),
	})
	err := validateMoveFileInitialization(before, after, filepath.Join(root, "old/value.go"), filepath.Join(root, "api/value.go"), "", "")
	if err == nil || !strings.Contains(err.Error(), "initialization-changing") {
		t.Fatalf("validateMoveFileInitialization() error = %v, want initialization refusal", err)
	}
}

func TestMoveFileInitializationEventsUseGlobalReadyOrder(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeMoveFixture(t, root, "go.mod", "module example.com/initmove\n\ngo 1.24\n")
	writeMoveFixture(t, root, "z/z.go", "package z\nvar Value = 1\n")
	writeMoveFixture(t, root, "a/a.go", "package a\nimport \"example.com/initmove/z\"\nvar Value = z.Value\n")
	writeMoveFixture(t, root, "b/b.go", "package b\nvar Value = 1\n")
	writeMoveFixture(t, root, "root/root.go", "package root\nimport (\n\t\"example.com/initmove/a\"\n\t\"example.com/initmove/b\"\n)\nvar Value = a.Value + b.Value\n")
	pkgs, err := loadMovePackages(context.Background(), root, nil)
	if err != nil {
		t.Fatal(err)
	}
	index := moveFilePackageIndex(pkgs)
	var rootPackage *packages.Package
	for _, pkg := range pkgs {
		if pkg.PkgPath == "example.com/initmove/root" {
			rootPackage = pkg
			break
		}
	}
	if rootPackage == nil {
		t.Fatal("root package was not loaded")
	}
	events, err := moveFileInitializationEvents(rootPackage.Types, index)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"example.com/initmove/b",
		"example.com/initmove/z",
		"example.com/initmove/a",
		"example.com/initmove/root",
	}
	if !slices.Equal(events, want) {
		t.Fatalf("moveFileInitializationEvents() = %v, want %v", events, want)
	}
}

func loadMoveFileInitializationPackages(t *testing.T, root string, after map[string][]byte) ([]*packages.Package, []*packages.Package) {
	t.Helper()
	before, err := loadMovePackages(context.Background(), root, nil)
	if err != nil {
		t.Fatal(err)
	}
	loadedAfter, err := loadMovePackages(context.Background(), root, after)
	if err != nil {
		t.Fatal(err)
	}
	return before, loadedAfter
}
