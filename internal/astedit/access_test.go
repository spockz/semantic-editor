// Tests for cross-language access modifier abstraction and Go validation rules.
package astedit

import (
	"errors"
	"testing"
)

func TestAccessModifier_Inference(t *testing.T) {
	backend := GolangBackend{}

	tests := []struct {
		name          string
		identifier    string
		mod           AccessModifier
		want          AccessModifier
		wantErr       bool
		wantTargetErr error
	}{
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := backend.ResolveEffectiveAccess(tt.mod, tt.identifier)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for mod %q and identifier %q, got nil", tt.mod, tt.identifier)
				}
				if tt.wantTargetErr != nil && !errors.Is(err, tt.wantTargetErr) {
					t.Fatalf("expected error wrapping %v, got %v", tt.wantTargetErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("ResolveEffectiveAccess() = %q, want %q", got, tt.want)
			}
		})
	}
}
