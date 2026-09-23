// path_test.go verifies canonical path and file-URI behavior shared across language backends.
package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalWorkspaceRootResolvesSymlinks(t *testing.T) {
	root := t.TempDir()
	alias := filepath.Join(t.TempDir(), "workspace")
	if err := os.Symlink(root, alias); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if got, want := CanonicalWorkspaceRoot(alias), CanonicalWorkspaceRoot(root); got != want {
		t.Fatalf("canonical alias = %q, want %q", got, want)
	}
}

func TestFileURIAndFilePathRoundTrip(t *testing.T) {
	file := filepath.Join(t.TempDir(), "a file.go")
	uri := FileURI(file)
	got, err := FilePathFromURI(uri)
	if err != nil {
		t.Fatal(err)
	}
	if want := CanonicalWorkspaceRoot(file); got != want {
		t.Fatalf("round trip path = %q, want %q", got, want)
	}
}

func TestFilePathFromURIRejectsRemoteAndEmptyPaths(t *testing.T) {
	for _, raw := range []string{"https://example.test/file.go", "file://remote-host/file.go", "file://"} {
		t.Run(raw, func(t *testing.T) {
			if _, err := FilePathFromURI(raw); err == nil {
				t.Fatalf("FilePathFromURI(%q) succeeded", raw)
			}
		})
	}
}

func TestPathWithinUsesPathBoundaries(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	if !PathWithin(root, filepath.Join(root, "src", "main.go")) {
		t.Fatal("PathWithin rejected a nested file")
	}
	if PathWithin(root, root+"-neighbor") {
		t.Fatal("PathWithin accepted a sibling path with a shared prefix")
	}
}
