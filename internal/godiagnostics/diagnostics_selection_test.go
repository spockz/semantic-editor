// Package godiagnostics verifies package selection rejects incomplete coverage.
package godiagnostics

import (
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestSelectPackageFilesRejectsErrorsWithoutGoFiles(t *testing.T) {
	root := t.TempDir()
	healthy := filepath.Join(root, "healthy.go")
	loaded := []*packages.Package{
		{ID: "healthy", GoFiles: []string{healthy}},
		{ID: "orphan", Errors: []packages.Error{{Msg: "build constraints exclude all Go files"}}},
	}
	_, _, err := selectPackageFiles(root, loaded)
	if err == nil || !strings.Contains(err.Error(), "incomplete Go package coverage") {
		t.Fatalf("selectPackageFiles() error = %v, want incomplete package coverage", err)
	}
}

func TestSelectPackageFilesKeepsFindingsForUsableFiles(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "broken.go")
	loaded := []*packages.Package{{
		ID:      "broken",
		GoFiles: []string{file},
		Errors:  []packages.Error{{Msg: "missing import"}, {Msg: "missing import"}},
	}}
	files, findings, err := selectPackageFiles(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != file {
		t.Fatalf("files = %v, want %q", files, file)
	}
	if len(findings) != 1 || findings[0].Severity != 1 || findings[0].Source != "package" || !strings.Contains(findings[0].String(), "[package] missing import") {
		t.Fatalf("findings = %+v, want deduplicated package severity 1", findings)
	}
}
