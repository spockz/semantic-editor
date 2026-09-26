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

func TestResolveStructFieldCoordinates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := `package config

type Config struct {
	Host string
	Port int
}
`
	filePath := filepath.Join(dir, "config.go")
	if err := os.WriteFile(filePath, []byte(source), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	res, err := symbol.Resolve(dir, "config.go", "Config.Port")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if res.Symbol != "Config.Port" {
		t.Errorf("expected Symbol Config.Port, got %q", res.Symbol)
	}
	if res.Line != 5 || res.Column != 2 {
		t.Errorf("expected Port at 5:2, got %d:%d", res.Line, res.Column)
	}
	if res.Kind != "field" {
		t.Errorf("expected Kind field, got %q", res.Kind)
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

func TestResolveInterfaceMethods(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	files := map[string]string{
		"outer.go": "package api\ntype Outer interface { Left; Right }\ntype Left interface { Base }\ntype Right interface { Base }\n",
		"base.go":  "package api\ntype Base interface { Write([]byte) error }\n",
		"other.go": "package other\ntype Base interface { Write([]byte) error }\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	result, err := symbol.Resolve(dir, "outer.go", "Outer.Write")
	if err != nil {
		t.Fatalf("Resolve interface method: %v", err)
	}
	if result.Definition == nil || result.Definition.File != "base.go" || result.Definition.Line != 2 {
		t.Fatalf("definition = %#v, want base.go:2", result.Definition)
	}
	if result.Definition.Receiver != "Outer" || result.Definition.Kind != "method" || result.Definition.Name != "Write" {
		t.Fatalf("definition = %#v, want Outer.Write method", result.Definition)
	}
	if result.Symbol != "Outer.Write" || result.File != result.Definition.File || result.Line != result.Definition.Line || result.Kind != result.Definition.Kind || result.Receiver != result.Definition.Receiver {
		t.Fatalf("legacy fields do not mirror definition: %#v", result)
	}
	if result.Ambiguous || len(result.Candidates) != 0 {
		t.Fatalf("diamond embedding produced ambiguity: %#v", result)
	}

	cycleDir := t.TempDir()
	cycleFiles := map[string]string{
		"cycle.go": "package api\ntype Cyclic interface { Loop }\ntype Loop interface { Cyclic; Ping() }\n",
	}
	for name, content := range cycleFiles {
		if err := os.WriteFile(filepath.Join(cycleDir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write cycle %s: %v", name, err)
		}
	}
	cycleResult, err := symbol.Resolve(cycleDir, "cycle.go", "Cyclic.Ping")
	if err != nil {
		t.Fatalf("Resolve method through embedding cycle: %v", err)
	}
	if cycleResult.Definition == nil || cycleResult.Definition.Name != "Ping" {
		t.Fatalf("cycle result = %#v, want Ping definition", cycleResult)
	}
}

func TestLookupDefinitionAndUsages(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "lookup.go")
	source := "package api\ntype Dispatcher struct { sink int }\nfunc NewDispatcher(sink int) {}\nfunc sink() {}\n"
	if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
		t.Fatalf("write lookup source: %v", err)
	}

	qualified, err := symbol.Resolve(dir, "lookup.go", "Dispatcher.sink")
	if err != nil {
		t.Fatalf("Resolve qualified field: %v", err)
	}
	if qualified.Definition == nil || qualified.Definition.Kind != "field" || qualified.Ambiguous {
		t.Fatalf("qualified result = %#v, want field definition", qualified)
	}
	if len(qualified.Usages) != 0 {
		t.Fatalf("qualified lookup included unrelated local bindings: %#v", qualified.Usages)
	}

	unqualified, err := symbol.Resolve(dir, "lookup.go", "sink")
	if err != nil {
		t.Fatalf("Resolve unqualified declaration: %v", err)
	}
	if unqualified.Definition == nil || unqualified.Definition.Kind != "function" || unqualified.Ambiguous {
		t.Fatalf("unqualified result = %#v, want function definition", unqualified)
	}
	if len(unqualified.Usages) != 1 || unqualified.Usages[0].Kind != "parameter" {
		t.Fatalf("unqualified usages = %#v, want the constructor parameter", unqualified.Usages)
	}
	if unqualified.Symbol != unqualified.Definition.BuildQualifiedName() || unqualified.File != unqualified.Definition.File || unqualified.Line != unqualified.Definition.Line {
		t.Fatalf("legacy fields do not mirror canonical definition: %#v", unqualified)
	}
}

func TestResolveLocalOnlyLookupPromotesDefinition(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "local.go")
	if err := os.WriteFile(file, []byte("package api\nfunc use(value int) {}\n"), 0o600); err != nil {
		t.Fatalf("write local source: %v", err)
	}
	result, err := symbol.Resolve(dir, "local.go", "value")
	if err != nil {
		t.Fatalf("Resolve local binding: %v", err)
	}
	if result.Definition == nil || result.Definition.Kind != "parameter" || result.Symbol != "value" || result.File != "local.go" {
		t.Fatalf("local result = %#v, want promoted parameter with legacy coordinates", result)
	}
	if len(result.Usages) != 1 || result.Usages[0] != result.Definition {
		t.Fatalf("local usages = %#v, want promoted binding", result.Usages)
	}
}

func TestResolveInterfaceMethodAmbiguity(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	files := map[string]string{
		"selected.go": "package api\ntype Selected interface { First; Second }\ntype Foreign interface { io.Writer }\n",
		"first.go":    "package api\ntype First interface { Read() error }\n",
		"second.go":   "package api\ntype Second interface { Read() error }\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	result, err := symbol.Resolve(dir, "selected.go", "Selected.Read")
	if err != nil {
		t.Fatalf("Resolve ambiguous interface method: %v", err)
	}
	if !result.Ambiguous || len(result.Candidates) != 2 {
		t.Fatalf("ambiguous result = %#v, want two physical method declarations", result)
	}
	for _, candidate := range result.Candidates {
		if candidate.Receiver != "Selected" || candidate.Kind != "method" {
			t.Fatalf("candidate = %#v, want Selected method", candidate)
		}
	}

	if _, err := symbol.Resolve(dir, "selected.go", "Foreign.Write"); !errors.Is(err, symbol.ErrNotFound) {
		t.Fatalf("Resolve imported embedding = %v, want ErrNotFound", err)
	}
}

func TestResolveInterfaceIndexScopeAndErrors(t *testing.T) {
	t.Parallel()
	selectedDir := t.TempDir()
	selectedFiles := map[string]string{
		"selected.go":  "package api\ntype Selected interface {}\n",
		"duplicate.go": "package api\ntype Selected interface { Hidden() }\n",
	}
	for name, content := range selectedFiles {
		if err := os.WriteFile(filepath.Join(selectedDir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if _, err := symbol.Resolve(selectedDir, "selected.go", "Selected.Hidden"); !errors.Is(err, symbol.ErrNotFound) {
		t.Fatalf("selected declaration lookup = %v, want ErrNotFound", err)
	}

	malformedDir := t.TempDir()
	malformedFiles := map[string]string{
		"selected.go": "package api\ntype Selected interface { Base }\n",
		"base.go":     "package api\ntype Base interface { Broken( }\n",
	}
	for name, content := range malformedFiles {
		if err := os.WriteFile(filepath.Join(malformedDir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write malformed fixture %s: %v", name, err)
		}
	}
	if _, err := symbol.Resolve(malformedDir, "selected.go", "Selected.Read"); err == nil || errors.Is(err, symbol.ErrNotFound) {
		t.Fatalf("malformed sibling lookup = %v, want an explicit parse error", err)
	}

	testDir := t.TempDir()
	testFiles := map[string]string{
		"selected.go":  "package api\ntype Selected interface { Base }\n",
		"base.go":      "package api\ntype Base interface {}\n",
		"base_test.go": "package api\ntype Base interface { Fake() }\n",
	}
	for name, content := range testFiles {
		if err := os.WriteFile(filepath.Join(testDir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write test sibling fixture %s: %v", name, err)
		}
	}
	if _, err := symbol.Resolve(testDir, "selected.go", "Selected.Fake"); !errors.Is(err, symbol.ErrNotFound) {
		t.Fatalf("test sibling lookup = %v, want ErrNotFound", err)
	}
}

func TestResolveHiddenWorkspaceRootAndDotRoot(t *testing.T) {
	t.Parallel()
	hiddenRoot := filepath.Join(t.TempDir(), ".private")
	if err := os.MkdirAll(hiddenRoot, 0o700); err != nil {
		t.Fatalf("create hidden root: %v", err)
	}
	file := filepath.Join(hiddenRoot, "target.go")
	if err := os.WriteFile(file, []byte("package private\ntype Visible int\n"), 0o600); err != nil {
		t.Fatalf("write hidden-root source: %v", err)
	}
	result, err := symbol.Resolve(hiddenRoot, "", "Visible")
	if err != nil {
		t.Fatalf("Resolve hidden explicit root: %v", err)
	}
	if result.Definition == nil || result.Definition.File != "target.go" {
		t.Fatalf("hidden-root result = %#v, want target.go definition", result)
	}

	if _, err := symbol.Resolve(".", "", "Resolve"); err != nil {
		t.Fatalf("Resolve dot search root: %v", err)
	}
}

func TestResolveInterfacePackageLocalAliasAndOffset(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	base := "package api\ntype Base interface { Write([]byte) error }\n"
	files := map[string]string{
		"outer.go": "package api\ntype Alias = Base\ntype Outer interface { Alias }\n",
		"base.go":  base,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	result, err := symbol.Resolve(dir, "outer.go", "Outer.Write")
	if err != nil {
		t.Fatalf("Resolve package-local alias: %v", err)
	}
	wantOffset := len("package api\ntype Base interface { ")
	if result.Definition == nil || result.Definition.File != "base.go" || result.Definition.Offset != wantOffset {
		t.Fatalf("alias result = %#v, want base.go offset %d", result.Definition, wantOffset)
	}
}

func TestResolveDefinedInterfaceIndirection(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "types.go")
	source := "package api\ntype Base interface { Write() }\ntype Derived Base\ntype Outer interface { Derived }\n"
	if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
		t.Fatalf("write interface source: %v", err)
	}
	for _, query := range []struct {
		name     string
		receiver string
	}{
		{name: "Derived.Write", receiver: "Derived"},
		{name: "Outer.Write", receiver: "Outer"},
	} {
		result, err := symbol.Resolve(dir, "types.go", query.name)
		if err != nil {
			t.Fatalf("Resolve %s: %v", query.name, err)
		}
		if result.Definition == nil || result.Definition.Name != "Write" || result.Definition.Receiver != query.receiver || result.Definition.Line != 2 {
			t.Fatalf("Resolve %s = %#v, want Write at line 2 with receiver %s", query.name, result.Definition, query.receiver)
		}
		if result.Ambiguous || len(result.Candidates) != 0 {
			t.Fatalf("Resolve %s = %#v, want one definition", query.name, result)
		}
	}
}

func TestResolveSnapshotCapturesSelectedAndInheritedSources(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	selectedPath := filepath.Join(dir, "outer.go")
	inheritedPath := filepath.Join(dir, "base.go")
	selectedSource := []byte("package api\ntype Outer interface { Base }\n")
	inheritedSource := []byte("package api\ntype Base interface { Write() }\n")
	if err := os.WriteFile(selectedPath, selectedSource, 0o600); err != nil {
		t.Fatalf("write selected source: %v", err)
	}
	if err := os.WriteFile(inheritedPath, inheritedSource, 0o600); err != nil {
		t.Fatalf("write inherited source: %v", err)
	}

	snapshot, err := symbol.ResolveSnapshot(dir, "outer.go", "Outer.Write")
	if err != nil {
		t.Fatalf("ResolveSnapshot: %v", err)
	}
	if snapshot.Result.Definition == nil || snapshot.Result.Definition.File != "base.go" {
		t.Fatalf("definition = %#v, want actual inherited declaration in base.go", snapshot.Result.Definition)
	}
	selectedKey, err := filepath.Abs(selectedPath)
	if err != nil {
		t.Fatalf("resolve selected path: %v", err)
	}
	inheritedKey, err := filepath.Abs(inheritedPath)
	if err != nil {
		t.Fatalf("resolve inherited path: %v", err)
	}
	if got := string(snapshot.Files[filepath.Clean(selectedKey)]); got != string(selectedSource) {
		t.Fatalf("selected snapshot = %q, want %q", got, selectedSource)
	}
	if got := string(snapshot.Files[filepath.Clean(inheritedKey)]); got != string(inheritedSource) {
		t.Fatalf("inherited snapshot = %q, want %q", got, inheritedSource)
	}
	if len(snapshot.Files) != 2 {
		t.Fatalf("snapshot files = %d, want selected and inherited declarations", len(snapshot.Files))
	}

	if err := os.WriteFile(selectedPath, []byte("package changed\n"), 0o600); err != nil {
		t.Fatalf("mutate selected source after resolution: %v", err)
	}
	if err := os.WriteFile(inheritedPath, []byte("package changed\n"), 0o600); err != nil {
		t.Fatalf("mutate inherited source after resolution: %v", err)
	}
	if got := string(snapshot.Files[filepath.Clean(selectedKey)]); got != string(selectedSource) {
		t.Errorf("selected snapshot changed after disk mutation: %q", got)
	}
	if got := string(snapshot.Files[filepath.Clean(inheritedKey)]); got != string(inheritedSource) {
		t.Errorf("inherited snapshot changed after disk mutation: %q", got)
	}
}
