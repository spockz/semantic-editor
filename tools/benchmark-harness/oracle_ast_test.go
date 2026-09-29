// This test protects the AST oracle call assertion used to verify extracted helper delegation.
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEvaluateASTRequiresConfiguredFunctionCall(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		want bool
	}{
		{name: "required call exists", src: "package sample\nfunc Normalize() { normalizeKind() }\nfunc normalizeKind() {}\n", want: true},
		{name: "required call absent", src: "package sample\nfunc Normalize() {}\nfunc normalizeKind() {}\n", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "sample.go")
			if err := os.WriteFile(path, []byte(tc.src), 0o600); err != nil {
				t.Fatal(err)
			}
			err := evaluateAST(ASTConfig{File: "sample.go", MustCallFunctions: []string{"Normalize->normalizeKind"}}, dir)
			if (err == nil) != tc.want {
				t.Fatalf("evaluateAST() error = %v, want success %t", err, tc.want)
			}
		})
	}
}
