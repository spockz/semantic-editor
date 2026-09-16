// Package golang_test validates Go adapter dependency operations.
package golang_test

import (
	"context"
	"errors"
	"testing"

	"semedit/internal/adapters/golang"
)

func TestAddDependency_Validation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	err := golang.AddDependency(ctx, t.TempDir(), "")
	if !errors.Is(err, golang.ErrEmptyPackage) {
		t.Fatalf("expected ErrEmptyPackage, got %v", err)
	}
}
