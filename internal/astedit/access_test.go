// Tests for cross-language access modifier abstraction and Go validation rules.
package astedit

import (
	"errors"
	"testing"
)

func TestAccessModifier_Inference(t *testing.T) {
	t.Parallel()
	backend := GolangBackend{}
	for _, testCase := range accessInferenceCases() {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := backend.ResolveEffectiveAccess(testCase.mod, testCase.identifier)
			if testCase.wantErr {
				assertAccessError(t, err, testCase.identifier, testCase.mod, testCase.wantTargetErr)
				return
			}
			assertAccessResult(t, got, err, testCase.want)
		})
	}
}

type accessInferenceCase struct {
	name          string
	identifier    string
	mod           AccessModifier
	want          AccessModifier
	wantErr       bool
	wantTargetErr error
}

func accessInferenceCases() []accessInferenceCase {
	return []accessInferenceCase{
		{
			name:       "infer uppercase as public",
			identifier: "HandleRequest",
			mod:        AccessModifierInfer,
			want:       AccessModifierPublic,
		},
		{
			name:       "infer lowercase as private",
			identifier: "handleRequest",
			mod:        AccessModifierInfer,
			want:       AccessModifierPrivate,
		},
		{
			name:       "empty mod defaults to infer uppercase",
			identifier: "Server",
			mod:        "",
			want:       AccessModifierPublic,
		},
		{
			name:       "empty mod defaults to infer lowercase",
			identifier: "config",
			mod:        "",
			want:       AccessModifierPrivate,
		},
		{
			name:       "explicit public matching uppercase",
			identifier: "ServeHTTP",
			mod:        AccessModifierPublic,
			want:       AccessModifierPublic,
		},
		{
			name:       "explicit private matching lowercase",
			identifier: "parseHeader",
			mod:        AccessModifierPrivate,
			want:       AccessModifierPrivate,
		},
		{
			name:          "unsupported protected in Go",
			identifier:    "Helper",
			mod:           AccessModifierProtected,
			wantErr:       true,
			wantTargetErr: ErrUnsupportedModifier,
		},
		{
			name:          "unsupported package-private in Go",
			identifier:    "helper",
			mod:           AccessModifierPackagePrivate,
			wantErr:       true,
			wantTargetErr: ErrUnsupportedModifier,
		},
		{
			name:          "mismatch public requested for lowercase identifier",
			identifier:    "helper",
			mod:           AccessModifierPublic,
			wantErr:       true,
			wantTargetErr: ErrVisibilityMismatch,
		},
		{
			name:          "mismatch private requested for uppercase identifier",
			identifier:    "Helper",
			mod:           AccessModifierPrivate,
			wantErr:       true,
			wantTargetErr: ErrVisibilityMismatch,
		},
	}
}

func assertAccessError(t *testing.T, err error, identifier string, requested AccessModifier, targetErr error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected access error for %q", identifier)
	}
	if targetErr != nil && !errors.Is(err, targetErr) {
		t.Fatalf("expected error wrapping %v, got %v", targetErr, err)
	}
	if !errors.Is(err, ErrVisibilityMismatch) {
		return
	}
	var visibilityErr *VisibilityMismatchError
	if !errors.As(err, &visibilityErr) {
		t.Fatalf("expected *VisibilityMismatchError via errors.As, got %T", err)
	}
	if visibilityErr.Identifier != identifier {
		t.Errorf("expected Identifier %q, got %q", identifier, visibilityErr.Identifier)
	}
	if visibilityErr.Requested != requested {
		t.Errorf("expected Requested %q, got %q", requested, visibilityErr.Requested)
	}
}

func assertAccessResult(t *testing.T, got AccessModifier, err error, expected AccessModifier) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Fatalf("ResolveEffectiveAccess() = %q, want %q", got, expected)
	}
}
