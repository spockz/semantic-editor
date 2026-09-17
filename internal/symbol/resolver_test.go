// Package symbol_test validates coordinate and identifier parsing logic for symbols.
package symbol_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"semedit/internal/symbol"
)

func TestParseIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		wantRecv string
		wantName string
		wantErr  bool
	}{
		{"Server.Start", "Server", "Start", false},
		{"(*Server).Start", "Server", "Start", false},
		{"*Server.Start", "Server", "Start", false},
		{"ValidateToken", "", "ValidateToken", false},
		{"'ValidateToken'", "", "ValidateToken", false},
		{"validate'", "", "validate'", false},
		{"", "", "", true},
		{"a.b.c", "", "", true},
	}

	for _, tt := range tests {
		recv, name, err := symbol.ParseIdentifier(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("ParseIdentifier(%q) expected error, got nil", tt.input)
			}
			if !errors.Is(err, symbol.ErrInvalidIdentifier) {
				t.Fatalf("ParseIdentifier(%q) expected ErrInvalidIdentifier, got %v", tt.input, err)
			}
		}
		if !tt.wantErr && err != nil {
			t.Fatalf("ParseIdentifier(%q) unexpected error: %v", tt.input, err)
		}
		if recv != tt.wantRecv || name != tt.wantName {
			t.Errorf("ParseIdentifier(%q) = (%q, %q), want (%q, %q)", tt.input, recv, name, tt.wantRecv, tt.wantName)
		}
	}
}

func TestResolveMethodCoordinates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := `package api

type Server struct{}

func (s *Server) Start() {}
`
	filePath := filepath.Join(dir, "server.go")
	if err := os.WriteFile(filePath, []byte(source), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	res, err := symbol.Resolve(dir, "server.go", "Server.Start")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if res.Line != 5 {
		t.Errorf("expected Line 5, got %d", res.Line)
	}
	if res.Column != 18 {
		t.Errorf("expected Column 18, got %d", res.Column)
	}
	if res.Kind != "method" {
		t.Errorf("expected Kind 'method', got %q", res.Kind)
	}
}

func TestResolveNotFound_StructuredError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := `package api

type Server struct{}

func (s *Server) Start() {}
`
	filePath := filepath.Join(dir, "server.go")
	if err := os.WriteFile(filePath, []byte(source), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	res, err := symbol.Resolve(dir, "server.go", "NonExistent")
	if err == nil {
		t.Fatalf("expected error, got result: %v", res)
	}

	if !errors.Is(err, symbol.ErrNotFound) {
		t.Fatalf("expected ErrNotFound via errors.Is, got %v", err)
	}

	var symErr *symbol.SymbolError
	if !errors.As(err, &symErr) {
		t.Fatalf("expected *symbol.SymbolError via errors.As, got %T", err)
	}

	if symErr.Symbol != "NonExistent" {
		t.Errorf("expected Symbol 'NonExistent', got %q", symErr.Symbol)
	}
	if symErr.File != "server.go" {
		t.Errorf("expected File 'server.go', got %q", symErr.File)
	}
	if symErr.Op != "resolve" {
		t.Errorf("expected Op 'resolve', got %q", symErr.Op)
	}
}
