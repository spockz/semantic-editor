// Package verification tests feature execution against a backend retaining its diagnostic values.
// WHY: Go module result annotation must not mutate storage owned by a language backend.
package verification

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"semedit/internal/backend"
)

type retainingBackend struct {
	diagnostics []backend.Diagnostic
}

func (retainingBackend) Language() backend.LanguageID { return backend.LanguageGo }

func (retainingBackend) Capabilities() backend.Capabilities {
	return backend.NewCapabilities(backend.OperationVerify)
}

func (retainingBackend) Lookup(context.Context, backend.ProjectContext, string) (*backend.LookupResult, error) {
	return nil, nil
}

func (retainingBackend) Rename(context.Context, backend.RenameRequest) (*backend.RenameResult, error) {
	return nil, nil
}

func (backend retainingBackend) Verify(context.Context, backend.VerifyRequest) ([]backend.Diagnostic, error) {
	return backend.diagnostics, nil
}

func TestGoModuleCoverageLeavesBackendDiagnosticsUnchanged(t *testing.T) {
	root := t.TempDir()
	writeVerificationFile(t, filepath.Join(root, "go.mod"), "module example.com/verify\n\ngo 1.27.1\n")
	writeVerificationFile(t, filepath.Join(root, "verify.go"), "package verify\n")
	retained := []backend.Diagnostic{{Message: "backend diagnostic"}}
	service, err := backend.NewRegistry(retainingBackend{diagnostics: retained})
	if err != nil {
		t.Fatal(err)
	}
	call := Request{
		Project: backend.ProjectContext{
			RootDir:  root,
			Language: backend.LanguageGo,
		},
		Path:      ".",
		CheckOnly: true,
		Service:   backend.NewService(service),
	}
	for attempt := range 2 {
		result, runErr := Run(context.Background(), call)
		if runErr != nil {
			t.Fatalf("verification attempt %d: %v", attempt+1, runErr)
		}
		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Message != "[.] backend diagnostic" {
			t.Fatalf("verification attempt %d diagnostics = %#v", attempt+1, result.Diagnostics)
		}
		if retained[0].Message != "backend diagnostic" {
			t.Fatalf("backend-owned diagnostics changed on attempt %d: %#v", attempt+1, retained)
		}
	}
}

func writeVerificationFile(t *testing.T, path, source string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatalf("write %s: %v", filepath.Base(path), err)
	}
}
