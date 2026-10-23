// WHY: Verifies publication refuses stale source and module inputs before writing.

package astedit

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func TestApplySemanticEditRechecksCapturedInputsAtPublication(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	targetPath := filepath.Join(root, "main.go")
	manifestPath := filepath.Join(root, "go.mod")
	before := []byte("package sample\nfunc Value() int { return 1 }\n")
	after := []byte("package sample\nfunc Value() int { return 2 }\n")
	manifest := []byte("module example.com/sample\n\ngo 1.24\n")
	for path, content := range map[string][]byte{targetPath: before, manifestPath: manifest} {
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	plan := SemanticEditPlan{
		Change:  sourcebuf.Change{Path: targetPath, Before: bytes.Clone(before), After: after},
		ReadSet: map[string][]byte{targetPath: bytes.Clone(before), manifestPath: bytes.Clone(manifest)},
	}
	if err := os.WriteFile(manifestPath, []byte("module example.com/changed\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ApplySemanticEdit(context.Background(), plan); err == nil {
		t.Fatal("ApplySemanticEdit() error = nil, want captured manifest drift refusal")
	}
	actual, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, before) {
		t.Fatalf("target changed after manifest drift: %q", actual)
	}
}

func TestCompleteRefactorReadSetRefusesManifestDrift(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	manifestPath := filepath.Join(root, "go.mod")
	if err := os.WriteFile(manifestPath, []byte("module example.com/current\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	original := map[string][]byte{manifestPath: []byte("module example.com/captured\n\ngo 1.24\n")}
	if _, err := completeRefactorReadSet(context.Background(), root, original, nil); err == nil {
		t.Fatal("completeRefactorReadSet() error = nil, want manifest drift refusal")
	}
}

func TestRefactorPublicationsPreserveCallerOverlayUntilFlush(t *testing.T) {
	cases := []refactorPublicationCase{
		{
			name:   "extract function",
			source: "package sample\n\nfunc Render(value int) {\n\tconsume(value + 1)\n}\n\nfunc consume(int) {}\n",
			want:   []byte("func renderTail(value int) {"),
			run: func(ctx context.Context, file string) error {
				_, err := ExtractFunction(ctx, file, "Render", "renderTail", ExtractOptions{Kind: ExtractKindFunction, StartLine: 4, EndLine: 4})
				return err
			},
		},
		{
			name:   "extract variable",
			source: "package sample\n\nfunc consume(int) {}\n\nfunc Render(count int) {\n\tconsume(count + 1)\n}\n",
			want:   []byte("value := count + 1"),
			run: func(ctx context.Context, file string) error {
				_, err := ExtractVariable(ctx, file, "Render", "value", ExtractVariableOptions{Discriminator: "count + 1"})
				return err
			},
		},
		{
			name:   "extract constant",
			source: "package sample\n\nfunc Render() int {\n\treturn 3 * 60\n}\n",
			want:   []byte("const retryWindow int = 3 * 60"),
			run: func(ctx context.Context, file string) error {
				_, err := ExtractConstant(ctx, file, "Render", "retryWindow", ExtractConstantOptions{Discriminator: "3 * 60"})
				return err
			},
		},
		{
			name:   "inline variable",
			source: "package sample\n\nfunc Render() {\n\tvalue := 42\n\tprintln(value)\n}\n",
			want:   []byte("println(int(42))"),
			run: func(ctx context.Context, file string) error {
				_, err := InlineVariable(ctx, file, "Render", "value")
				return err
			},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) { assertRefactorPublicationCase(t, test) })
	}
}

type refactorPublicationCase struct {
	name   string
	source string
	want   []byte
	run    func(context.Context, string) error
}

func assertRefactorPublicationCase(t *testing.T, test refactorPublicationCase) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/publication\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "main.go")
	if err := os.WriteFile(file, []byte(test.source), 0o600); err != nil {
		t.Fatal(err)
	}
	sibling := filepath.Join(root, "sibling.go")
	beforeSibling := []byte("package sample\n\nconst Original = 1\n")
	stagedSibling := []byte("package sample\n\nconst Overlay = 2\n")
	if err := os.WriteFile(sibling, beforeSibling, 0o600); err != nil {
		t.Fatal(err)
	}
	buffer, err := sourcebuf.New(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sourcebuf.WithBuffer(t.Context(), buffer)
	if applyErr := buffer.Apply(ctx, []sourcebuf.Change{{Path: sibling, Before: beforeSibling, After: stagedSibling}}); applyErr != nil {
		t.Fatal(applyErr)
	}
	if runErr := test.run(ctx, file); runErr != nil {
		t.Fatalf("refactor failed: %v", runErr)
	}
	assertPublicationFile(t, file, []byte(test.source), "disk target before flush")
	stagedTarget, readErr := sourcebuf.ReadFile(ctx, file)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Contains(stagedTarget, test.want) {
		t.Fatalf("staged target %q does not contain %q", stagedTarget, test.want)
	}
	stagedSiblingRead, readErr := sourcebuf.ReadFile(ctx, sibling)
	if readErr != nil || !bytes.Equal(stagedSiblingRead, stagedSibling) {
		t.Fatalf("existing sibling overlay = %q, err %v", stagedSiblingRead, readErr)
	}
	if flushErr := buffer.Flush(ctx); flushErr != nil {
		t.Fatalf("flush caller-owned buffer: %v", flushErr)
	}
	assertPublicationFile(t, file, stagedTarget, "persisted target")
	assertPublicationFile(t, sibling, stagedSibling, "persisted sibling")
}

func assertPublicationFile(t *testing.T, path string, want []byte, label string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s = %q, want %q", label, got, want)
	}
}
