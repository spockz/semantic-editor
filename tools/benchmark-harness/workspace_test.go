// Runtime preparation tests protect fixture isolation and atomic replacement across all harnesses.

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRuntimeLayoutConfinesFilesAndDirectories(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	for _, directory := range []string{"../outside", "escape/nested"} {
		if err := prepareRuntimeLayout(root, []string{directory}, nil); err == nil {
			t.Fatalf("created escaping directory %q", directory)
		}
	}
	for _, path := range []string{"../outside", "escape/config.json"} {
		if err := prepareRuntimeLayout(root, nil, map[string][]byte{path: []byte("private")}); err == nil {
			t.Fatalf("published escaping file %q", path)
		}
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("runtime setup wrote outside fixture: %v", entries)
	}
}

func TestRuntimeLayoutReplacesFilesWithAdvancingMtime(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(".agents", "config.json")
	if err := prepareRuntimeLayout(root, nil, map[string][]byte{path: []byte("first")}); err != nil {
		t.Fatal(err)
	}
	fullPath := filepath.Join(root, path)
	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(fullPath, future, future); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(fullPath)
	if err != nil {
		t.Fatal(err)
	}
	if writeErr := prepareRuntimeLayout(root, nil, map[string][]byte{path: []byte("second")}); writeErr != nil {
		t.Fatal(writeErr)
	}
	after, err := os.Stat(fullPath)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().After(before.ModTime()) {
		t.Fatal("runtime replacement did not advance mtime")
	}
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "second" {
		t.Fatalf("published content = %q", content)
	}
	entries, err := os.ReadDir(filepath.Dir(fullPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("temporary runtime files remain: %v", entries)
	}
}
