// Package backend_test verifies the language-neutral registry and service contract.
package backend_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"semedit/internal/backend"
)

type testBackend struct {
	language     backend.LanguageID
	capabilities backend.Capabilities
}

func (b testBackend) Language() backend.LanguageID       { return b.language }
func (b testBackend) Capabilities() backend.Capabilities { return b.capabilities }
func (b testBackend) Lookup(context.Context, backend.ProjectContext, string) (*backend.LookupResult, error) {
	return &backend.LookupResult{}, nil
}
func (b testBackend) Rename(context.Context, backend.ProjectContext, *backend.LookupResult, string) error {
	return nil
}
func (b testBackend) Verify(context.Context, backend.ProjectContext, string) ([]backend.Diagnostic, error) {
	return nil, nil
}

func TestRegistrySelectsExplicitAndAutoGoBackend(t *testing.T) {
	registry, err := backend.NewRegistry(backend.NewGoBackend())
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}

	selected, err := registry.Select(backend.ProjectContext{Language: backend.LanguageGo})
	if err != nil {
		t.Fatalf("explicit selection failed: %v", err)
	}
	if selected.Language() != backend.LanguageGo {
		t.Fatalf("selected language = %q, want %q", selected.Language(), backend.LanguageGo)
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	selected, err = registry.Select(backend.ProjectContext{RootDir: root})
	if err != nil || selected.Language() != backend.LanguageGo {
		t.Fatalf("auto selection = %v, backend = %v", err, selected)
	}

	_, err = registry.Select(backend.ProjectContext{RootDir: t.TempDir()})
	if err == nil || !errors.Is(err, backend.ErrLanguageUndetected) {
		t.Fatalf("auto selection without project marker error = %v", err)
	}
}

func TestServiceRejectsUnsupportedCapability(t *testing.T) {
	registry, err := backend.NewRegistry(testBackend{
		language:     "test",
		capabilities: backend.NewCapabilities(backend.OperationLookup),
	})
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}
	service := backend.NewService(registry)
	_, err = service.Verify(context.Background(), backend.VerifyRequest{
		Project: backend.ProjectContext{Language: "test"},
	})
	if !errors.Is(err, backend.ErrUnsupportedOperation) {
		t.Fatalf("Verify error = %v, want ErrUnsupportedOperation", err)
	}
}

func TestWorkspaceTrustRequiresExplicitCanonicalConsent(t *testing.T) {
	root := t.TempDir()
	registry, err := backend.NewRegistry(testBackend{
		language:     "external",
		capabilities: backend.NewCapabilitiesRequiringWorkspaceTrust(backend.OperationLookup),
	})
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}
	service := backend.NewService(registry)
	project := backend.ProjectContext{RootDir: root, Language: "external"}

	_, err = service.Lookup(context.Background(), project, "Thing")
	if err == nil || !errors.Is(err, backend.ErrWorkspaceTrustRequired) {
		t.Fatalf("untrusted lookup error = %v, want workspace trust error", err)
	}
	var trustErr *backend.WorkspaceTrustError
	if !errors.As(err, &trustErr) {
		t.Fatalf("lookup error type = %T, want *WorkspaceTrustError", err)
	}
	if trustErr.Operation != backend.OperationLookup || trustErr.Language != "external" || trustErr.Workspace != backend.CanonicalWorkspaceRoot(root) {
		t.Fatalf("trust error = %+v, want operation, language, and canonical workspace", trustErr)
	}

	trusted := project
	trusted.WorkspaceTrust = backend.NewWorkspaceTrust(root, true)
	if _, err := service.Lookup(context.Background(), trusted, "Thing"); err != nil {
		t.Fatalf("trusted lookup failed: %v", err)
	}

	// Consent is carried by the request only; a later request remains untrusted.
	if _, err := service.Lookup(context.Background(), project, "Thing"); !errors.Is(err, backend.ErrWorkspaceTrustRequired) {
		t.Fatalf("trust persisted across requests: %v", err)
	}
}

func TestWorkspaceTrustUsesCanonicalRootScope(t *testing.T) {
	root := t.TempDir()
	alias := filepath.Join(t.TempDir(), "workspace-link")
	if err := os.Symlink(root, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	trust := backend.NewWorkspaceTrust(alias, true)
	if !trust.Allows(root) {
		t.Fatalf("canonical trust does not apply to symlinked workspace: %+v", trust)
	}
	if trust.Allows(t.TempDir()) {
		t.Fatal("trust unexpectedly applies to another workspace")
	}
	if (backend.WorkspaceTrust{}).Allows(root) {
		t.Fatal("zero-value workspace trust is trusted")
	}
}

func TestPositionFromByteOffsetUsesUTF16Units(t *testing.T) {
	source := []byte("🙂name\nnext")
	position := backend.PositionFromByteOffset(source, 6)
	if position != (backend.Position{Line: 0, Character: 4}) {
		t.Fatalf("position = %+v, want line 0 character 4", position)
	}

	position = backend.PositionFromByteOffset(source, len("🙂name\nnext"))
	if position != (backend.Position{Line: 1, Character: 4}) {
		t.Fatalf("position = %+v, want line 1 character 4", position)
	}
}
