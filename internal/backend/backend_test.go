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
func (b testBackend) Rename(context.Context, backend.RenameRequest) (*backend.RenameResult, error) {
	return &backend.RenameResult{}, nil
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

// TestCapabilityMatrixConformance asserts that every backend implementing
// MatrixProvider declares operations that exactly match its runtime Capabilities.
// This is the CI drift gate between documentation and behaviour.
func TestCapabilityMatrixConformance(t *testing.T) {
	svc := backend.NewDefaultService()
	langs := []backend.LanguageID{
		backend.LanguageGo,
		backend.LanguageRust,
		backend.LanguageJava,
		backend.LanguageScala,
		backend.LanguageHaskell,
	}
	for _, lang := range langs {
		b, ok := svc.Registry.Backend(lang)
		if !ok {
			t.Errorf("language %q: not found in default registry", lang)
			continue
		}
		mp, ok := b.(backend.MatrixProvider)
		if !ok {
			// Not every backend must implement MatrixProvider, but all current ones do.
			t.Errorf("language %q: Backend does not implement MatrixProvider", lang)
			continue
		}
		matrix := mp.CapabilityMatrix()
		caps := b.Capabilities()

		// Map operation names used in the matrix to the backend.Operation constants.
		opMapping := map[string]backend.Operation{
			"rename": backend.OperationRename,
			"lookup": backend.OperationLookup,
			"verify": backend.OperationVerify,
		}

		for opName, opCap := range matrix.Operations {
			if !opCap.Supported {
				continue
			}
			runtimeOp, known := opMapping[opName]
			if !known {
				// Operations like insert_func are Go-AST-layer operations not in the
				// backend.Operation set; they are not gated by Capabilities().Supports().
				continue
			}
			if !caps.Supports(runtimeOp) {
				t.Errorf("language %q operation %q: declared Supported=true in matrix but Capabilities().Supports(%q)=false",
					lang, opName, runtimeOp)
			}
		}

		// Verify that runtime operations not in the matrix are not silently missing.
		for _, runtimeOp := range []backend.Operation{backend.OperationLookup, backend.OperationRename, backend.OperationVerify} {
			if !caps.Supports(runtimeOp) {
				continue
			}
			// Find the matrix key that maps to this runtime operation.
			matrixKey := ""
			for key, mapped := range opMapping {
				if mapped == runtimeOp {
					matrixKey = key
					break
				}
			}
			if matrixKey == "" {
				continue
			}
			opCap, declared := matrix.Operations[matrixKey]
			if !declared || !opCap.Supported {
				t.Errorf("language %q: runtime Capabilities().Supports(%q)=true but matrix operation %q is absent or Supported=false",
					lang, runtimeOp, matrixKey)
			}
		}

		if matrix.Language == "" {
			t.Errorf("language %q: LanguageMatrix.Language is empty", lang)
		}
		if matrix.DisplayName == "" {
			t.Errorf("language %q: LanguageMatrix.DisplayName is empty", lang)
		}
		if matrix.Maturity == "" {
			t.Errorf("language %q: LanguageMatrix.Maturity is empty", lang)
		}
	}
}
