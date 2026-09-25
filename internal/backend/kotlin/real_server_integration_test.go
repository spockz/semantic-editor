// Package kotlin tests real server startup and selected-file behavior that fake LSP sessions cannot prove.
package kotlin

import (
	"context"
	"os"
	"path/filepath"
	neutralbackend "semedit/internal/backend"
	"semedit/internal/backend/pathutil"
	"semedit/internal/pipeline"
	"strings"
	"testing"
)

func TestRealKotlinLanguageServerIntegration(t *testing.T) {
	server := os.Getenv("SEMEDIT_REAL_KOTLIN_LS")
	javaHome := os.Getenv("SEMEDIT_REAL_KOTLIN_JAVA_HOME")
	if server == "" || javaHome == "" {
		t.Skip("set SEMEDIT_REAL_KOTLIN_LS and SEMEDIT_REAL_KOTLIN_JAVA_HOME to run the real server integration test")
	}
	server, err := filepath.Abs(server)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(server); err != nil {
		t.Fatalf("Kotlin language server: %v", err)
	}
	javaHome, err = filepath.Abs(javaHome)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(javaHome, "bin", "java")); err != nil {
		t.Fatalf("Java runtime: %v", err)
	}
	t.Setenv("JAVA_HOME", javaHome)

	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	scratch := filepath.Join(repoRoot, ".scratch")
	if err := os.MkdirAll(scratch, 0o700); err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(scratch, "kotlin-real-integration-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(root); err != nil {
			t.Errorf("remove Kotlin integration fixture: %v", err)
		}
	})

	file := filepath.Join(root, "Widget.kt")
	source := []byte("package sample\nclass Widget { fun launch() {} }\n")
	if err := pipeline.WriteAtomic(file, source); err != nil {
		t.Fatal(err)
	}
	project := neutralbackend.ProjectContext{
		RootDir:        root,
		File:           file,
		WorkspaceTrust: neutralbackend.NewWorkspaceTrust(root, true),
		Kotlin:         neutralbackend.KotlinConfig{KotlinBin: server},
	}
	adapter := NewKotlinBackend()
	result, err := adapter.Lookup(context.Background(), project, "Widget.launch")
	if err != nil {
		t.Fatalf("real Kotlin lookup: %v", err)
	}
	if result == nil || result.Symbol != "Widget.launch" || result.Kind != "function" ||
		result.Offset != strings.Index(string(source), "launch") ||
		result.Location.URI != pathutil.FileURI(file) {
		t.Fatalf("real Kotlin lookup result: %+v", result)
	}
	diagnostics, err := adapter.Verify(context.Background(), neutralbackend.VerifyRequest{Project: project})
	if err != nil {
		t.Fatalf("real Kotlin diagnostics: %v", err)
	}
	if diagnostics == nil || len(diagnostics) != 0 {
		t.Fatalf("real Kotlin diagnostics = %#v, want an explicit empty report", diagnostics)
	}
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(source) {
		t.Fatal("real Kotlin operations changed the selected source")
	}
	entries, err := os.ReadDir(filepath.Join(root, ".scratch"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("real Kotlin operations left scratch sessions: %v", entries)
	}
}
