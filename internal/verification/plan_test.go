// Package verification tests feature execution against a backend retaining its diagnostic values.
// WHY: Go module result annotation must not mutate storage owned by a language backend.
package verification

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
)

type retainingBackend struct {
	diagnostics []backend.Diagnostic
}

type scopedBackend struct {
	calls []backend.VerifyRequest
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

func (*scopedBackend) Language() backend.LanguageID { return backend.LanguageGo }

func (*scopedBackend) Capabilities() backend.Capabilities {
	return backend.NewCapabilities(backend.OperationVerify)
}

func (*scopedBackend) Lookup(context.Context, backend.ProjectContext, string) (*backend.LookupResult, error) {
	return nil, nil
}

func (*scopedBackend) Rename(context.Context, backend.RenameRequest) (*backend.RenameResult, error) {
	return nil, nil
}

func (backend *scopedBackend) Verify(_ context.Context, request backend.VerifyRequest) ([]backend.Diagnostic, error) {
	backend.calls = append(backend.calls, request)
	return nil, nil
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

func TestGoPackagePatternsSkipDiscoveredModuleMissingFromScope(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "tools", "goctl")
	writeVerificationFile(t, filepath.Join(root, "go.mod"), "module example.com/verify\n\ngo 1.27.1\n")
	writeVerificationFile(t, filepath.Join(root, "main.go"), "package verify\n")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	writeVerificationFile(t, filepath.Join(nested, "go.mod"), "module example.com/goctl\n\ngo 1.27.1\n")
	writeVerificationFile(t, filepath.Join(nested, "main.go"), "package goctl\n")
	backendSpy := &scopedBackend{}
	registry, err := backend.NewRegistry(backendSpy)
	if err != nil {
		t.Fatal(err)
	}
	result, err := verifyGoLanguage(context.Background(), backend.NewService(registry), backend.ProjectContext{RootDir: root, Language: backend.LanguageGo}, Request{
		Project: backend.ProjectContext{RootDir: root, Language: backend.LanguageGo}, CheckOnly: true,
		GoPackagePatterns: map[string][]string{root: {"."}},
	}, "go", []string{"main.go", "tools/goctl/main.go"})
	if err != nil {
		t.Fatalf("verifyGoLanguage: %v", err)
	}
	if len(backendSpy.calls) != 1 || backendSpy.calls[0].Project.RootDir != root {
		t.Fatalf("verification calls = %#v, want only the selected root module", backendSpy.calls)
	}
	if len(result.Coverage) != 1 || result.Coverage[0].ModuleRoot != "." || result.Coverage[0].PackagePattern != "." {
		t.Fatalf("coverage = %#v, want only selected root coverage", result.Coverage)
	}
}

func writeVerificationFile(t *testing.T, path, source string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatalf("write %s: %v", filepath.Base(path), err)
	}
}
