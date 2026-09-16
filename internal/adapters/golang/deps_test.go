// Package golang_test validates Go adapter dependency operations.
package golang_test

import (
	"context"
	"testing"

	"semedit/internal/adapters/golang"
)

func TestAddDependency_Validation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	err := golang.AddDependency(ctx, t.TempDir(), "")
	if err == nil {
		t.Fatal("expected error on empty package, got nil")
	}
}
