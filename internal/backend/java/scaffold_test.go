// WHY: Verifies Java scaffold package inference and fail-closed behavior without starting JDT LS.
package java

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScaffoldFileInfersSiblingPackageAndWritesAtomically(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "src", "main", "java", "com", "acme", "billing")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Invoice.java"), []byte("/* header */\npackage com.acme.billing;\nclass Invoice {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "InvoiceService.java")
	got, err := ScaffoldFile(context.Background(), root, target, "infer", false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "com.acme.billing" {
		t.Fatalf("inferred package = %q", got)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	want := "// WHY: Starts this Java source file in its package so declarations can be added with semantic operations.\npackage com.acme.billing;\n\n"
	if string(content) != want {
		t.Fatalf("scaffold content = %q, want %q", content, want)
	}
}

func TestScaffoldFileInfersPackageFromMavenSourcePath(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "module", "src", "test", "java", "com", "acme", "billing", "InvoiceTest.java")
	got, err := ScaffoldFile(context.Background(), root, target, "infer", false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "com.acme.billing" {
		t.Fatalf("inferred package = %q", got)
	}
}

func TestScaffoldFileRejectsConflictingSiblingsAndInvalidPackages(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "src", "main", "java", "com", "acme")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"One.java", "Two.java"} {
		pkg := "com.acme.one"
		if file == "Two.java" {
			pkg = "com.acme.two"
		}
		if err := os.WriteFile(filepath.Join(dir, file), []byte("package "+pkg+";\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(dir, "Three.java")
	if _, err := ScaffoldFile(context.Background(), root, target, "infer", false); err == nil {
		t.Fatal("conflicting sibling packages were accepted")
	}
	if _, err := ScaffoldFile(context.Background(), root, target, "com.acme.class", false); err == nil {
		t.Fatal("keyword package segment was accepted")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("failed scaffold created target: %v", err)
	}
}
