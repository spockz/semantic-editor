// WHY: Proves constant extraction captures type-analysis inputs before buffered publication.

package astedit

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"semedit/internal/sourcebuf"
	"testing"
)

func TestLoadConstantPackageBindsReadSetBeforeApply(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(t *testing.T, dependency, manifest string)
		want   string
	}{
		{
			name: "imported package source",
			mutate: func(t *testing.T, dependency, manifest string) {
				t.Helper()
				if err := os.WriteFile(dependency, []byte("package constants\nconst Window = 4\n"), 0o600); err != nil {
					t.Fatalf("change imported source: %v", err)
				}
			},
			want: "imported source change",
		},
		{
			name: "module manifest",
			mutate: func(t *testing.T, dependency, manifest string) {
				t.Helper()
				content, err := os.ReadFile(manifest)
				if err != nil {
					t.Fatalf("read go.mod: %v", err)
				}
				if err := os.WriteFile(manifest, append(content, []byte("\n// concurrent change\n")...), 0o600); err != nil {
					t.Fatalf("change go.mod: %v", err)
				}
			},
			want: "module manifest change",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, buffer, selected, dependency, manifest, original := constantInputFixture(t)
			if _, err := loadConstantPackage(ctx, filepath.Dir(selected), sourcebuf.Overlay(ctx)); err != nil {
				t.Fatalf("load constant package: %v", err)
			}
			tc.mutate(t, dependency, manifest)
			candidate := append(append([]byte(nil), original...), []byte("\n// planned edit\n")...)
			err := buffer.Apply(ctx, []sourcebuf.Change{{Path: selected, Before: original, After: candidate}})
			if !errors.Is(err, sourcebuf.ErrConflict) {
				t.Fatalf("Buffer.Apply() error = %v, want conflict after %s", err, tc.want)
			}
			if changes := buffer.Changes(); len(changes) != 0 {
				t.Fatalf("failed Buffer.Apply() retained %d edits", len(changes))
			}
			disk, err := os.ReadFile(selected)
			if err != nil {
				t.Fatalf("read selected source: %v", err)
			}
			if !bytes.Equal(disk, original) {
				t.Fatalf("%s changed selected source", tc.want)
			}
		})
	}
}

func constantInputFixture(t *testing.T) (context.Context, *sourcebuf.Buffer, string, string, string, []byte) {
	t.Helper()
	root := t.TempDir()
	manifest := filepath.Join(root, "go.mod")
	if err := os.WriteFile(manifest, []byte("module example.com/constantinputs\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatalf("write module manifest: %v", err)
	}
	packageDir := filepath.Join(root, "service")
	dependencyDir := filepath.Join(root, "constants")
	if err := os.MkdirAll(packageDir, 0o750); err != nil {
		t.Fatalf("create selected package: %v", err)
	}
	if err := os.MkdirAll(dependencyDir, 0o750); err != nil {
		t.Fatalf("create dependency package: %v", err)
	}
	selected := filepath.Join(packageDir, "retry.go")
	original := []byte("package service\nimport \"example.com/constantinputs/constants\"\nfunc Retry() int { return constants.Window * 60 }\n")
	if err := os.WriteFile(selected, original, 0o600); err != nil {
		t.Fatalf("write selected source: %v", err)
	}
	dependency := filepath.Join(dependencyDir, "constants.go")
	if err := os.WriteFile(dependency, []byte("package constants\nconst Window = 3\n"), 0o600); err != nil {
		t.Fatalf("write imported source: %v", err)
	}
	buffer, err := sourcebuf.New(root)
	if err != nil {
		t.Fatalf("create source buffer: %v", err)
	}
	return sourcebuf.WithBuffer(context.Background(), buffer), buffer, selected, dependency, manifest, original
}
