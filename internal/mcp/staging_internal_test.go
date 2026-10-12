// This file verifies retained-stage workspace ownership and expiry at the manager boundary.
package mcp

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStageRejectsOverlappingWorkspaceRootsInOneModule(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{"go.mod": "module example.test/staging\n\ngo 1.27\n", "value.go": "package staging\n", "child/value.go": "package staging\n"} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	server := NewServer("full", root, io.Discard)
	if _, err := server.stages.Start(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	if _, err := server.stages.Start(context.Background(), filepath.Join(root, "child")); err == nil {
		t.Fatal("overlapping module workspace stage succeeded")
	}
}

func TestStageRejectsAncestorAfterChildStarts(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{"go.mod": "module example.test/staging\n\ngo 1.27\n", "child/value.go": "package child\n"} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	server := NewServer("full", root, io.Discard)
	if _, err := server.stages.Start(context.Background(), filepath.Join(root, "child")); err != nil {
		t.Fatal(err)
	}
	if _, err := server.stages.Start(context.Background(), root); err == nil {
		t.Fatal("ancestor stage succeeded after child stage")
	}
}

func TestStageRejectsSiblingWorkspaceInOwnedModule(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{"go.mod": "module example.test/staging\n\ngo 1.27\n", "first/value.go": "package first\n", "second/value.go": "package second\n"} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	server := NewServer("full", root, io.Discard)
	if _, err := server.stages.Start(context.Background(), filepath.Join(root, "first")); err != nil {
		t.Fatal(err)
	}
	if _, err := server.stages.Start(context.Background(), filepath.Join(root, "second")); err == nil {
		t.Fatal("sibling stage succeeded in an owned module")
	}
}

func TestStageBlocksNestedModuleRequestAfterStart(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{"go.mod": "module example.test/staging\n\ngo 1.27\n", "value.go": "package staging\n", "child/value.go": "package child\n"} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	server := NewServer("full", root, io.Discard)
	if _, err := server.stages.Start(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "child", "go.mod"), []byte("module example.test/child\n\ngo 1.27\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := server.stages.BlockBatch(context.Background(), filepath.Join(root, "child")); err == nil {
		t.Fatal("batch request in a newly nested module bypassed active stage ownership")
	}
}

func TestStageExpiryDropsRetainedBytesWithoutPublication(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/staging\n\ngo 1.27\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "value.go"), []byte("package staging\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := NewServer("full", root, io.Discard, WithStageLimits(50, 10<<20, 10<<20, time.Nanosecond))
	if _, err := server.stages.Start(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)
	if _, err := server.stages.Abort(context.Background(), root); err == nil {
		t.Fatal("expired stage abort succeeded")
	}
	content, err := os.ReadFile(filepath.Join(root, "value.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "package staging\n" {
		t.Fatalf("expiry published source: %q", content)
	}
}

func TestExpiredStageNoLongerBlocksBatchOrSiblingStart(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{"go.mod": "module example.test/staging\n\ngo 1.27\n", "value.go": "package staging\n", "sibling/value.go": "package sibling\n"} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	server := NewServer("full", root, io.Discard, WithStageLimits(50, 10<<20, 10<<20, time.Nanosecond))
	if _, err := server.stages.Start(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)
	if err := server.stages.BlockBatch(context.Background(), root); err != nil {
		t.Fatalf("expired stage still blocked batch: %v", err)
	}
	if _, err := server.stages.Start(context.Background(), filepath.Join(root, "sibling")); err != nil {
		t.Fatalf("expired stage still blocked sibling start: %v", err)
	}
}
