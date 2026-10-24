// Package java_test checks overload behavior that protocol stand-ins cannot establish.
// WHY: The CLI fixture's primitive and generic postimages must retain their original Java overload targets.
package java_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/rogpeppe/go-internal/txtar"
)

func TestJavaExplicitTypePreservesOverloadBehavior(t *testing.T) {
	javaHome := os.Getenv("SEMEDIT_REAL_JAVA_HOME")
	if javaHome == "" {
		t.Skip("set SEMEDIT_REAL_JAVA_HOME to compile and run Java explicit-type CLI fixture postimages")
	}
	archive, err := txtar.ParseFile(filepath.Join("..", "..", "..", "testdata", "scripts", "refactor_make_type_explicit_java.txtar"))
	if err != nil {
		t.Fatal(err)
	}
	var before, after []byte
	for _, file := range archive.Files {
		switch file.Name {
		case "src/Service.java":
			before = file.Data
		case "want/Service.java":
			after = file.Data
		}
	}
	if len(before) == 0 || len(after) == 0 {
		t.Fatal("CLI fixture must contain original and expected Java sources")
	}
	for _, scenario := range []struct {
		name   string
		source []byte
		want   string
	}{
		{"inferred", before, "int:3/array:1\n"},
		{"explicit", after, "int:3/array:1\n"},
		{"primitive oracle sensitivity", bytes.Replace(after, []byte("int total"), []byte("long total"), 1), "long:3/array:1\n"},
		{"generic oracle sensitivity", bytes.Replace(after, []byte("ArrayList<String> items"), []byte("List<String> items"), 1), "int:3/list:1\n"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			root := t.TempDir()
			file := filepath.Join(root, "Service.java")
			if err := os.WriteFile(file, scenario.source, 0o600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			output, err := exec.CommandContext(ctx, filepath.Join(javaHome, "bin", "javac"), "--release", "21", "-d", root, file).CombinedOutput()
			if err != nil {
				t.Fatalf("compile: %v\n%s", err, output)
			}
			output, err = exec.CommandContext(ctx, filepath.Join(javaHome, "bin", "java"), "-cp", root, "Service").CombinedOutput()
			if err != nil {
				t.Fatalf("run: %v\n%s", err, output)
			}
			if string(output) != scenario.want {
				t.Fatalf("output = %q, want %q", output, scenario.want)
			}
		})
	}
}
