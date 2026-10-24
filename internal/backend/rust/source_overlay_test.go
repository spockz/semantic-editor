// Package rust tests that language-server input reflects the active semantic source overlay.
package rust

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	neutralbackend "github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func TestResolveRustProjectReadsBufferedSource(t *testing.T) {
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if writeErr := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte("[package]\nname = \"overlay\"\nversion = \"0.1.0\"\n"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	file := filepath.Join(root, "lib.rs")
	disk := []byte("pub fn value() -> i32 { 1 }\n")
	if writeErr := os.WriteFile(file, disk, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	buffer, err := sourcebuf.New(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if authorizeErr := buffer.AuthorizeWrites(context.Background(), root); authorizeErr != nil {
		t.Fatal(authorizeErr)
	}
	updated := []byte("pub fn value() -> i32 { 2 }\n")
	if applyErr := buffer.Apply(context.Background(), []sourcebuf.Change{{Path: file, Before: disk, After: updated}}); applyErr != nil {
		t.Fatal(applyErr)
	}
	_, _, got, err := resolveRustProject(sourcebuf.WithBuffer(context.Background(), buffer), neutralbackend.ProjectContext{RootDir: root, File: file})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(updated) {
		t.Fatalf("resolveRustProject() source = %q, want overlay %q", got, updated)
	}
}

func TestLookupAndOutlineUseBufferedSourceAndReportItsView(t *testing.T) {
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if writeErr := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte("[package]\nname = \"overlay\"\nversion = \"0.1.0\"\n"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	file := filepath.Join(root, "lib.rs")
	disk := []byte("fn run() {\n    consume(1 + 2);\n}\n")
	if writeErr := os.WriteFile(file, disk, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	buffer, err := sourcebuf.New(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if authorizeErr := buffer.AuthorizeWrites(context.Background(), root); authorizeErr != nil {
		t.Fatal(authorizeErr)
	}
	updated := []byte("fn run() {\n    consume(3 + 4);\n}\n")
	if applyErr := buffer.Apply(context.Background(), []sourcebuf.Change{{Path: file, Before: disk, After: updated}}); applyErr != nil {
		t.Fatal(applyErr)
	}
	ctx := sourcebuf.WithBuffer(context.Background(), buffer)
	trust := neutralbackend.NewWorkspaceTrust(root, true)
	project := neutralbackend.ProjectContext{RootDir: root, File: file, WorkspaceTrust: trust}
	session := &extractRustSession{}
	b := NewRustBackendWithFactory(func(context.Context, string) (RustSession, error) { return session, nil })
	lookup, err := b.Lookup(ctx, project, "run")
	if err != nil || lookup == nil || lookup.SourceView != "working" {
		t.Fatalf("Lookup() result = %#v, err %v; want working source", lookup, err)
	}
	outline, err := b.Outline(ctx, neutralbackend.OutlineRequest{Project: project, Path: file, IncludeUnexported: true})
	if err != nil || outline == nil || len(outline.Files) != 1 || outline.Files[0].SourceView != "working" {
		t.Fatalf("Outline() result = %#v, err %v; want one working-source file", outline, err)
	}
	if len(session.opened) != 2 || session.opened[0] != string(updated) || session.opened[1] != string(updated) {
		t.Fatalf("LSP didOpen text = %#v, want overlay %q twice", session.opened, updated)
	}
	afterDisk, err := os.ReadFile(file) // #nosec G304 -- test fixture path is created in t.TempDir.
	if err != nil || string(afterDisk) != string(disk) {
		t.Fatalf("disk source = %q, err %v; want unchanged %q", afterDisk, err, disk)
	}
}
