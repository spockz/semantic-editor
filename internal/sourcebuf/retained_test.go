// This file verifies retained buffers isolate candidate revisions without weakening disk evidence.
package sourcebuf_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func TestRetainedForkKeepsParentRevision(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "value.go")
	if err := os.WriteFile(path, []byte("package sample\n\nfunc Value() int { return 1 }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	candidate := buffer.Fork()
	ctx := sourcebuf.WithBuffer(context.Background(), candidate)
	before, err := sourcebuf.ReadFile(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := sourcebuf.WriteFile(ctx, path, append(before[:len(before)-2], []byte("2 }\n")...)); err != nil {
		t.Fatal(err)
	}
	if buffer.DirtySourceBytes() != 0 {
		t.Fatalf("parent dirty bytes = %d", buffer.DirtySourceBytes())
	}
	if candidate.DirtySourceBytes() == 0 {
		t.Fatal("candidate did not retain dirty bytes")
	}
}

func TestSymbolRenameAliasesFollowForkAndRollback(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "value.go")
	if err := os.WriteFile(path, []byte("package sample\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: path, OldSelector: "Value", NewSelector: "Renamed", Kind: "function"})
	child := buffer.Fork()
	child.RecordSymbolRename(sourcebuf.SymbolAlias{File: path, OldSelector: "Renamed", NewSelector: "Final", Kind: "function"})
	if got := child.SymbolAliases(); len(got) != 2 || got[1].OldSelector != "Renamed" || got[1].NewSelector != "Final" {
		t.Fatalf("child aliases = %#v", got)
	}
	if got := buffer.SymbolAliases(); len(got) != 1 || got[0].NewSelector != "Renamed" {
		t.Fatalf("parent aliases = %#v", got)
	}
	checkpoint := child.Checkpoint()
	child.RecordSymbolRename(sourcebuf.SymbolAlias{File: path, OldSelector: "Final", NewSelector: "Published", Kind: "function"})
	child.Rollback(checkpoint)
	if got := child.SymbolAliases(); len(got) != 2 || got[1].NewSelector != "Final" {
		t.Fatalf("rolled-back aliases = %#v", got)
	}
	if got := child.ForkOriginal().SymbolAliases(); len(got) != 0 {
		t.Fatalf("original fork aliases = %#v", got)
	}
}

func TestMergeReadSetDoesNotAcceptCandidateRenameAliases(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "value.go")
	if err := os.WriteFile(path, []byte("package sample\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	baseline, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	candidate := baseline.Fork()
	candidate.RecordSymbolRename(sourcebuf.SymbolAlias{File: path, OldSelector: "Value", NewSelector: "Renamed", Kind: "function"})
	if err := baseline.MergeReadSetFrom(candidate); err != nil {
		t.Fatal(err)
	}
	if got := baseline.SymbolAliases(); len(got) != 0 {
		t.Fatalf("read-set merge accepted unretained rename aliases = %#v", got)
	}
}

func TestInvalidateSymbolAliasesRemovesDeletedTypeDescendants(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "value.go")
	if err := os.WriteFile(path, []byte("package sample\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: path, OldSelector: "Server.Start", NewSelector: "Server.Serve", Kind: "method"})
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: path, OldSelector: "Other", NewSelector: "Replacement", Kind: "type"})
	buffer.InvalidateSymbolAliases(path, "Server")
	if got := buffer.SymbolAliases(); len(got) != 1 || got[0].OldSelector != "Other" {
		t.Fatalf("aliases after type deletion = %#v", got)
	}
	buffer.InvalidateSymbolAliases(path, "Replacement")
	if got := buffer.SymbolAliases(); len(got) != 0 {
		t.Fatalf("aliases after replacement deletion = %#v", got)
	}
}

func TestInvalidateSymbolAliasesRemovesWholeChainAcrossReceiverFiles(t *testing.T) {
	root := t.TempDir()
	buffer, err := sourcebuf.New(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "source.go")
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: file, OldSelector: "ParseConfig", NewSelector: "LoadConfig", Kind: "function"})
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: file, OldSelector: "LoadConfig", NewSelector: "ReadConfig", Kind: "function"})
	buffer.InvalidateSymbolAliases(file, "ReadConfig")
	if aliases := buffer.SymbolAliases(); len(aliases) != 0 {
		t.Fatalf("deleted identity retained an earlier edge: %#v", aliases)
	}
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: filepath.Join(root, "methods.go"), OldSelector: "(*Server).Start", NewSelector: "(*Server).Serve", Kind: "method"})
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: filepath.Join(root, "methods.go"), OldSelector: "(Server).Run", NewSelector: "(Server).Execute", Kind: "method"})
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: filepath.Join(root, "other", "methods.go"), OldSelector: "Server.Run", NewSelector: "Server.Execute", Kind: "method"})
	buffer.InvalidateSymbolAliases(filepath.Join(root, "type.go"), "Server")
	if aliases := buffer.SymbolAliases(); len(aliases) != 1 || aliases[0].File != filepath.Join(root, "other", "methods.go") {
		t.Fatalf("receiver identity/scope invalidation: %#v", aliases)
	}
}

func TestInvalidateSymbolAliasesRetainsTypeReplacementTombstone(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "reader.go")
	buffer, err := sourcebuf.New(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: file, OldSelector: "Parser", NewSelector: "Reader", Kind: "type"})
	buffer.InvalidateSymbolAliases(file, "Reader.ParseConfig")
	if aliases := buffer.SymbolAliases(); len(aliases) != 0 {
		t.Fatalf("aliases after replacing a member of the renamed type = %#v", aliases)
	}
	invalidated := buffer.InvalidatedSymbolAliases()
	if len(invalidated) != 1 || invalidated[0].OldSelector != "Parser" || invalidated[0].NewSelector != "Reader" {
		t.Fatalf("type replacement tombstones = %#v", invalidated)
	}
	if copied := buffer.Fork().InvalidatedSymbolAliases(); len(copied) != 1 || copied[0] != invalidated[0] {
		t.Fatalf("forked tombstones = %#v", copied)
	}
	if original := buffer.ForkOriginal().InvalidatedSymbolAliases(); len(original) != 0 {
		t.Fatalf("original fork tombstones = %#v", original)
	}
}

func TestInvalidatedSymbolAliasesSurviveReuseAndRollback(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "config.go")
	buffer, err := sourcebuf.New(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: file, OldSelector: "ParseConfig", NewSelector: "ReadConfig", Kind: "function"})
	checkpoint := buffer.Checkpoint()
	buffer.InvalidateSymbolAliases(file, "ReadConfig")
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: file, OldSelector: "Other", NewSelector: "ParseConfig", Kind: "function"})
	invalidated := buffer.InvalidatedSymbolAliases()
	if len(invalidated) != 1 || invalidated[0].OldSelector != "ParseConfig" {
		t.Fatalf("tombstones after selector reuse = %#v", invalidated)
	}
	buffer.Rollback(checkpoint)
	if invalidated := buffer.InvalidatedSymbolAliases(); len(invalidated) != 0 {
		t.Fatalf("rollback retained child tombstones = %#v", invalidated)
	}
	if aliases := buffer.SymbolAliases(); len(aliases) != 1 || aliases[0].NewSelector != "ReadConfig" {
		t.Fatalf("rollback aliases = %#v", aliases)
	}
}

func TestInvalidateSymbolAliasesTombstonesUnrenamedDeclaration(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "config.go")
	buffer, err := sourcebuf.New(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	buffer.InvalidateSymbolAliases(file, "ParseConfig")
	invalidated := buffer.InvalidatedSymbolAliases()
	if len(invalidated) != 1 {
		t.Fatalf("unrenamed declaration tombstones = %#v", invalidated)
	}
	if got := invalidated[0]; got.File != file || got.NewFile != file || got.OldSelector != "ParseConfig" || got.NewSelector != "ParseConfig" || got.Language != "go" || got.Kind != "declaration" {
		t.Fatalf("unrenamed declaration tombstone = %#v", got)
	}
}

func TestSymbolHistoryRestoresBothRenameAndInvalidationEvidence(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "config.go")
	buffer, err := sourcebuf.New(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: file, OldSelector: "Live", NewSelector: "Current", Kind: "function"})
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: file, OldSelector: "Old", NewSelector: "Gone", Kind: "function"})
	buffer.InvalidateSymbolAliases(file, "Gone")
	history := buffer.SymbolHistory()
	fresh, err := sourcebuf.New(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := fresh.RestoreSymbolHistory(history); err != nil {
		t.Fatal(err)
	}
	if aliases := fresh.SymbolAliases(); len(aliases) != 1 || aliases[0].OldSelector != "Live" {
		t.Fatalf("restored live aliases = %#v", aliases)
	}
	if invalidated := fresh.InvalidatedSymbolAliases(); len(invalidated) != 1 || invalidated[0].OldSelector != "Old" {
		t.Fatalf("restored tombstones = %#v", invalidated)
	}
	if err := fresh.RestoreSymbolHistory(history); err == nil {
		t.Fatal("restoring history into non-fresh buffer succeeded")
	}
}

func TestFlushClearsSymbolHistoryAndAllowsBufferReuse(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "value.go")
	writeFiles(t, []string{file}, []byte("package sample\n"))
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: file, OldSelector: "Value", NewSelector: "Renamed", Kind: "function"})
	buffer.InvalidateSymbolAliases(file, "Removed")
	history := buffer.SymbolHistory()
	if len(history.Renames) != 1 || len(history.Invalidated) != 1 {
		t.Fatalf("initial history = %#v", history)
	}
	if err := sourcebuf.WriteFile(ctx, file, []byte("package sample\nfunc Renamed() {}\n")); err != nil {
		t.Fatal(err)
	}
	if err := buffer.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if got := buffer.SymbolHistory(); len(got.Renames) != 0 || len(got.Invalidated) != 0 {
		t.Fatalf("flushed history = %#v", got)
	}
	if err := buffer.RestoreSymbolHistory(history); err != nil {
		t.Fatalf("restore history into flushed buffer: %v", err)
	}
	if got := buffer.SymbolHistory(); len(got.Renames) != 1 || len(got.Invalidated) != 1 || got.Renames[0] != history.Renames[0] || got.Invalidated[0] != history.Invalidated[0] {
		t.Fatalf("restored history = %#v; want %#v", got, history)
	}
	final := "package sample\nfunc Renamed() int { return 2 }\n"
	if err := sourcebuf.WriteFile(ctx, file, []byte(final)); err != nil {
		t.Fatalf("reuse flushed buffer: %v", err)
	}
	if err := buffer.Flush(ctx); err != nil {
		t.Fatalf("publish reused buffer: %v", err)
	}
	disk, err := os.ReadFile(file)
	if err != nil || string(disk) != final {
		t.Fatalf("reused buffer disk = %q, err = %v; want %q", disk, err, final)
	}
	if got := buffer.SymbolHistory(); len(got.Renames) != 0 || len(got.Invalidated) != 0 {
		t.Fatalf("reused buffer history = %#v", got)
	}
}

func TestPublishRetainedReportsZeroPublicationWithoutClearing(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "value.go")
	if err := os.WriteFile(path, []byte("package sample\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(context.Background(), buffer)
	if writeErr := sourcebuf.WriteFile(ctx, path, []byte("package sample\n\nfunc Value() {}\n")); writeErr != nil {
		t.Fatal(writeErr)
	}
	if writeErr := os.WriteFile(path, []byte("package changed\n"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	published, err := buffer.PublishRetained(context.Background())
	if err == nil || published != 0 {
		t.Fatalf("PublishRetained = (%d, %v), want zero conflict", published, err)
	}
	if buffer.DirtySourceBytes() == 0 {
		t.Fatal("zero-publication failure cleared retained bytes")
	}
}

func TestFlushPublishesNewGoFileOnlyAtPublication(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/buffer\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "generated", "model.go")
	ctx := sourcebuf.WithBuffer(context.Background(), buffer)
	if writeErr := sourcebuf.WriteFile(ctx, path, []byte("package generated\n")); writeErr != nil {
		t.Fatal(writeErr)
	}
	if _, statErr := os.Stat(filepath.Dir(path)); !os.IsNotExist(statErr) {
		t.Fatalf("buffered output directory exists before Flush: %v", statErr)
	}
	if got, readErr := sourcebuf.ReadFile(ctx, path); readErr != nil || string(got) != "package generated\n" {
		t.Fatalf("ReadFile buffered new file = %q, %v", got, readErr)
	}
	if _, ok := sourcebuf.Overlay(ctx)[path]; !ok {
		t.Fatal("overlay omitted buffered new Go file")
	}
	if flushErr := buffer.Flush(ctx); flushErr != nil {
		t.Fatal(flushErr)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "package generated\n" {
		t.Fatalf("published new file = %q, %v", got, err)
	}
}

func TestWriteFileRejectsNewFileBelowSymlinkedParent(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if symlinkErr := os.Symlink(outside, filepath.Join(root, "outside")); symlinkErr != nil {
		t.Fatal(symlinkErr)
	}
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(context.Background(), buffer)
	path := filepath.Join(root, "outside", "model.go")
	if err := sourcebuf.WriteFile(ctx, path, []byte("package outside\n")); err == nil {
		t.Fatal("buffered write through symlinked parent succeeded")
	}
	if _, statErr := os.Stat(filepath.Join(outside, "model.go")); !os.IsNotExist(statErr) {
		t.Fatalf("write through symlink created outside file: %v", statErr)
	}
}

func TestPublishRetainedRejectsNewParentReplacedBySymlink(t *testing.T) {
	root := t.TempDir()
	sibling := filepath.Join(root, "sibling")
	if err := os.Mkdir(sibling, 0o700); err != nil {
		t.Fatal(err)
	}
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(context.Background(), buffer)
	path := filepath.Join(root, "generated", "model.go")
	if writeErr := sourcebuf.WriteFile(ctx, path, []byte("package generated\n")); writeErr != nil {
		t.Fatal(writeErr)
	}
	if symlinkErr := os.Symlink(sibling, filepath.Dir(path)); symlinkErr != nil {
		t.Fatal(symlinkErr)
	}
	published, err := buffer.PublishRetained(context.Background())
	if err == nil || published != 0 {
		t.Fatalf("PublishRetained = (%d, %v), want zero rejected publication", published, err)
	}
	if _, statErr := os.Stat(filepath.Join(sibling, "model.go")); !os.IsNotExist(statErr) {
		t.Fatalf("aliased publication created sibling source: %v", statErr)
	}
}
