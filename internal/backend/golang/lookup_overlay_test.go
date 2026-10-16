// Package gobackend verifies the public lookup contract against retained source bytes.
package gobackend

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func TestLookupUsesOverlayBytesAndUTF16Location(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/lookupoverlay\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	diskSource := "package sample\n\nfunc OldName() {}\n"
	file := filepath.Join(root, "main.go")
	if err := os.WriteFile(file, []byte(diskSource), 0o600); err != nil {
		t.Fatal(err)
	}
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	err = buffer.AuthorizeWrites(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	ctx = sourcebuf.WithBuffer(ctx, buffer)
	overlay := "// 😀 shifted prefix\npackage sample\n\nvar prefix = \"😀\"; func NewName() {}\n"
	err = sourcebuf.WriteFile(ctx, file, []byte(overlay))
	if err != nil {
		t.Fatal(err)
	}
	project := backend.ProjectContext{RootDir: root, File: "main.go", Language: backend.LanguageGo}
	result, err := (GoBackend{}).Lookup(ctx, project, "NewName")
	if err != nil {
		t.Fatalf("Lookup(NewName) from overlay: %v", err)
	}
	if result.Definition == nil {
		t.Fatal("Lookup(NewName) omitted its definition")
	}
	location := result.Definition.Location
	if location.Range.Start.Line != 3 || location.Range.Start.Character != 24 || location.Range.End.Line != 3 || location.Range.End.Character != 31 {
		t.Fatalf("overlay UTF-16 location = %#v, want line 3 characters 24..31", location.Range)
	}
	if result.Definition.DeclarationRange == nil || result.Definition.DeclarationRange.Start.Line != 3 || result.Definition.DeclarationRange.End.Line != 3 {
		t.Fatalf("overlay declaration range = %#v, want line 3 declaration", result.Definition.DeclarationRange)
	}
	if result.Definition.Signature == nil || *result.Definition.Signature != "func NewName()" {
		t.Fatalf("overlay declaration signature = %#v, want func NewName()", result.Definition.Signature)
	}
	if result.SourceView != "working" {
		t.Fatalf("source view = %q, want working", result.SourceView)
	}
	if _, lookupErr := (GoBackend{}).Lookup(ctx, project, "OldName"); lookupErr == nil {
		t.Fatal("Lookup(OldName) resolved a name that exists only on disk")
	}
	if _, lookupErr := (GoBackend{}).Lookup(ctx, backend.ProjectContext{RootDir: root, File: "missing.go", Language: backend.LanguageGo}, "NewName"); lookupErr == nil {
		t.Fatal("Lookup(NewName) with a missing target unexpectedly succeeded")
	}
	contents, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != diskSource {
		t.Fatalf("Lookup changed disk source to %q", contents)
	}
}
