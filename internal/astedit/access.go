// Package astedit provides AST-level code transformation routines including declaration insertion, body replacement, and visibility management.
package astedit

import (
	"fmt"
	"go/ast"
	"slices"
	"unicode"
)

// AccessModifier represents the visibility or accessibility scope of a declaration.
type AccessModifier string

// Canonical access modifier values.
const (
	AccessModifierInfer          AccessModifier = "infer"
	AccessModifierPublic         AccessModifier = "public"
	AccessModifierPrivate        AccessModifier = "private"
	AccessModifierProtected      AccessModifier = "protected"
	AccessModifierPackagePrivate AccessModifier = "package-private"
)

// LanguageBackend declares supported access modifiers and language-specific visibility rules.
type LanguageBackend interface {
	Name() string
	SupportedAccessModifiers() []AccessModifier
	ResolveEffectiveAccess(mod AccessModifier, identifier string) (AccessModifier, error)
	ValidateModifier(mod AccessModifier, identifier string) error
}

// GolangBackend implements LanguageBackend for the Go language.
type GolangBackend struct{}

// Name returns the language identifier.
func (g GolangBackend) Name() string {
	return "go"
}

// SupportedAccessModifiers returns access modifiers supported by Go.
func (g GolangBackend) SupportedAccessModifiers() []AccessModifier {
	return []AccessModifier{
		AccessModifierInfer,
		AccessModifierPublic,
		AccessModifierPrivate,
	}
}

// ResolveEffectiveAccess resolves an access modifier into either public or private according to Go rules.
func (g GolangBackend) ResolveEffectiveAccess(mod AccessModifier, identifier string) (AccessModifier, error) {
	if err := g.ValidateModifier(mod, identifier); err != nil {
		return "", err
	}

	if mod == "" || mod == AccessModifierInfer {
		if isIdentifierExported(identifier) {
			return AccessModifierPublic, nil
		}
		return AccessModifierPrivate, nil
	}

	return mod, nil
}

// ValidateModifier ensures the requested modifier is valid and matches the identifier's capitalization in Go.
func (g GolangBackend) ValidateModifier(mod AccessModifier, identifier string) error {
	if mod == "" || mod == AccessModifierInfer {
		return nil
	}

	supported := g.SupportedAccessModifiers()
	if !slices.Contains(supported, mod) {
		return fmt.Errorf("%w: %q is not supported by the Go language backend (supported: infer, public, private)", ErrUnsupportedModifier, mod)
	}

	exported := isIdentifierExported(identifier)
	if mod == AccessModifierPublic && !exported {
		return &VisibilityMismatchError{
			Identifier: identifier,
			Requested:  AccessModifierPublic,
			Effective:  AccessModifierPrivate,
			Err:        ErrVisibilityMismatch,
		}
	}
	if mod == AccessModifierPrivate && exported {
		return &VisibilityMismatchError{
			Identifier: identifier,
			Requested:  AccessModifierPrivate,
			Effective:  AccessModifierPublic,
			Err:        ErrVisibilityMismatch,
		}
	}

	return nil
}

// DefaultBackend returns the active language backend (defaults to Go).
var DefaultBackend LanguageBackend = GolangBackend{}

func isIdentifierExported(name string) bool {
	if name == "" {
		return false
	}
	runes := []rune(name)
	return unicode.IsUpper(runes[0]) && ast.IsExported(name)
}

// ValidateAccess ensures the modifier and identifier are valid for the active backend.
func ValidateAccess(backend LanguageBackend, mod AccessModifier, identifier string) error {
	if backend == nil {
		backend = DefaultBackend
	}
	return backend.ValidateModifier(mod, identifier)
}

// ResolveAccess resolves the effective modifier for the active backend.
func ResolveAccess(backend LanguageBackend, mod AccessModifier, identifier string) (AccessModifier, error) {
	if backend == nil {
		backend = DefaultBackend
	}
	if identifier == "" {
		return "", ErrMissingIdentifier
	}
	return backend.ResolveEffectiveAccess(mod, identifier)
}
