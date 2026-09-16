// Package golang_test verifies gopls binary discovery and invocation.
package golang_test

import (
	"testing"

	"semedit/internal/adapters/golang"
)

func TestFindGopls(t *testing.T) {
	t.Parallel()

	path, err := golang.FindGopls()
	if err != nil {
		t.Skipf("gopls binary not found in environment: %v", err)
	}
	if path == "" {
		t.Errorf("expected non-empty gopls path")
	}
}
