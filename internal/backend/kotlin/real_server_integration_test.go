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
	server, serverPathErr := filepath.Abs(server)
	if serverPathErr != nil {
		t.Fatal(serverPathErr)
	}
	if _, statErr := os.Stat(server); statErr != nil {
		t.Fatalf("Kotlin language server: %v", statErr)
	}
	javaHome, javaHomeErr := filepath.Abs(javaHome)
	if javaHomeErr != nil {
		t.Fatal(javaHomeErr)
	}
	if _, runtimeErr := os.Stat(filepath.Join(javaHome, "bin", "java")); runtimeErr != nil {
		t.Fatalf("Java runtime: %v", runtimeErr)
	}
	t.Setenv("JAVA_HOME", javaHome)

	repoRoot, rootPathErr := filepath.Abs(filepath.Join("..", "..", ".."))
	if rootPathErr != nil {
		t.Fatal(rootPathErr)
	}
	scratch := filepath.Join(repoRoot, ".scratch")
	if err := os.MkdirAll(scratch, 0o700); err != nil {
		t.Fatal(err)
	}
	root, fixtureErr := os.MkdirTemp(scratch, "kotlin-real-integration-")
	if fixtureErr != nil {
		t.Fatal(fixtureErr)
	}
	t.Cleanup(func() {
		if cleanupErr := os.RemoveAll(root); cleanupErr != nil {
			t.Errorf("remove Kotlin integration fixture: %v", cleanupErr)
		}
	})

	file := filepath.Join(root, "Widget.kt")
	source := []byte("package sample\nclass Widget { fun launch() {} }\n")
	if writeErr := pipeline.WriteAtomic(file, source); writeErr != nil {
		t.Fatal(writeErr)
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
	assertKotlinIntegrationWorkspaceUnchanged(t, root, file, source)
}

func assertKotlinIntegrationWorkspaceUnchanged(t *testing.T, root, file string, source []byte) {
	t.Helper()
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
