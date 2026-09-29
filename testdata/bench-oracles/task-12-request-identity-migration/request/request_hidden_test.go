package request

import (
	"errors"
	"testing"
)

func TestRequestIdentityMigrationContract(t *testing.T) {
	if ErrInvalidRequest == nil {
		t.Fatal("ErrInvalidRequest is nil")
	}
	r := Request{RequestID: "req-hidden", Kind: "  AUDIT "}
	if err := r.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got := Normalize(r.Kind); got != "audit" {
		t.Fatalf("Normalize() = %q, want audit", got)
	}
	if got := normalizeKind("  AUDIT "); got != "audit" {
		t.Fatalf("normalizeKind() = %q, want audit", got)
	}
	if err := (Request{}).Validate(); err == nil || !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Validate() error = %v, want ErrInvalidRequest", err)
	}
}
