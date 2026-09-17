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
