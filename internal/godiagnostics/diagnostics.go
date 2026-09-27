// Package godiagnostics requests Go diagnostics from a fresh gopls session for selected project files.
package godiagnostics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/build"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"semedit/internal/lsp"

	"golang.org/x/tools/go/packages"
)

// Diagnostic carries gopls severity and source data for callers that need policy decisions.
type Diagnostic struct {
	Message  string
	Severity int
	Path     string
	Line     int
	Column   int
	Source   string
}

func (d Diagnostic) String() string {
	if d.Source == "package" {
		return fmt.Sprintf("%s: [package] %s", d.Path, d.Message)
	}
	severity := map[int]string{1: "error", 2: "warning", 3: "information", 4: "hint"}[d.Severity]
	if severity == "" {
		severity = "unknown"
	}
	return fmt.Sprintf("%s:%d:%d: [%s] %s", d.Path, d.Line, d.Column, severity, d.Message)
}

// Messages projects typed diagnostics to their stable string representation.
func Messages(diagnostics []Diagnostic) []string {
	messages := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		messages = append(messages, diagnostic.String())
	}
	slices.Sort(messages)
	return messages
}

// Check requests diagnostics for all Go files selected by the project package pattern.
func Check(ctx context.Context, root, executable string, environment []string) ([]string, error) {
	diagnostics, err := CheckDetailed(ctx, root, executable, environment)
	return Messages(diagnostics), err
}

// ErrGoplsNotFound indicates the gopls executable could not be located.
var ErrGoplsNotFound = errors.New("gopls executable not found")

// FindGopls locates the gopls executable on PATH or in standard Go binary directories.
func FindGopls() (string, error) {
	if executable, err := exec.LookPath("gopls"); err == nil {
		return executable, nil
	}
	candidates := []string{
		filepath.Join(os.Getenv("GOBIN"), "gopls"),
		filepath.Join(os.Getenv("GOPATH"), "bin", "gopls"),
		filepath.Join(build.Default.GOPATH, "bin", "gopls"),
		filepath.Join(os.Getenv("HOME"), "go", "bin", "gopls"),
		"/opt/homebrew/bin/gopls",
		"/usr/local/bin/gopls",
	}
	for _, candidate := range candidates {
		if candidate == "gopls" || candidate == "" {
			continue
		}
		// #nosec G703 -- candidates are fixed Go executable search paths.
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return filepath.Clean(candidate), nil
		}
	}
	return "", ErrGoplsNotFound
}

// CheckDetailed requests typed diagnostics for all Go files selected by the project package pattern.
func CheckDetailed(ctx context.Context, root, executable string, environment []string) (diagnostics []Diagnostic, resultErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	sessionCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve Go diagnostics root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat Go diagnostics root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("go diagnostics root %q is not a directory", root)
	}
	if executable == "" {
		return nil, errors.New("gopls executable is required")
	}
	files, packageFindings, err := loadGoFiles(sessionCtx, root, environment)
	if err != nil {
		return nil, err
	}
	diagnostics = append(diagnostics, packageFindings...)

	cmd := exec.CommandContext(sessionCtx, executable, "serve")
	cmd.Dir = root
	cmd.Env = environment
	client, err := lsp.NewProcessClient(cmd)
	if err != nil {
		return nil, fmt.Errorf("start gopls diagnostics session: %w", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("close gopls diagnostics session: %w", closeErr))
		}
	}()

	rootURI := (&url.URL{Scheme: "file", Path: filepath.ToSlash(root)}).String()
	initResult, err := client.Request(sessionCtx, "initialize", map[string]any{
		"processId":        nil,
		"rootUri":          rootURI,
		"workspaceFolders": []map[string]any{{"uri": rootURI, "name": filepath.Base(root)}},
		"capabilities": map[string]any{
			"textDocument": map[string]any{"diagnostic": map[string]any{"dynamicRegistration": false}},
		},
		"initializationOptions": map[string]any{"pullDiagnostics": true},
	})
	if err != nil {
		return nil, fmt.Errorf("initialize gopls diagnostics session: %w", err)
	}
	var initialized struct {
		Capabilities struct {
			DiagnosticProvider json.RawMessage `json:"diagnosticProvider"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(initResult, &initialized); err != nil {
		return nil, fmt.Errorf("decode gopls initialization result: %w", err)
	}
	provider := initialized.Capabilities.DiagnosticProvider
	if len(provider) == 0 || string(provider) == "null" || string(provider) == "false" {
		return nil, errors.New("gopls does not advertise a diagnostic provider")
	}
	if err := client.Notify(sessionCtx, "initialized", map[string]any{}); err != nil {
		return nil, fmt.Errorf("complete gopls initialization: %w", err)
	}

	for _, path := range files {
		found, err := pullDiagnostics(sessionCtx, client, root, path)
		if err != nil {
			return nil, err
		}
		diagnostics = append(diagnostics, found...)
	}
	if _, err := client.Request(sessionCtx, "shutdown", map[string]any{}); err != nil {
		return nil, fmt.Errorf("shutdown gopls diagnostics session: %w", err)
	}
	if err := client.Notify(sessionCtx, "exit", nil); err != nil {
		return nil, fmt.Errorf("exit gopls diagnostics session: %w", err)
	}
	if err := sessionCtx.Err(); err != nil {
		return nil, fmt.Errorf("go diagnostics session: %w", err)
	}
	slices.SortFunc(diagnostics, func(a, b Diagnostic) int { return strings.Compare(a.String(), b.String()) })
	return diagnostics, nil
}

func loadGoFiles(ctx context.Context, root string, environment []string) ([]string, []Diagnostic, error) {
	loaded, err := packages.Load(&packages.Config{
		Context:    ctx,
		Dir:        root,
		Env:        environment,
		Mode:       packages.NeedName | packages.NeedFiles,
		Tests:      true,
		BuildFlags: []string{"-mod=readonly"},
	}, "./...")
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, nil, fmt.Errorf("load Go packages for diagnostics: %w", ctxErr)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("load Go packages for diagnostics: %w", err)
	}
	return selectPackageFiles(root, loaded)
}

func pullDiagnostics(ctx context.Context, client *lsp.Client, root, path string) ([]Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("request Go diagnostics: %w", err)
	}
	// #nosec G304 -- packages.Load selected this source from the project root.
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read Go source %q: %w", path, err)
	}
	uri := (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
	if err := client.Notify(ctx, "textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "go", "version": 1, "text": string(content)},
	}); err != nil {
		return nil, fmt.Errorf("open Go source %q in gopls: %w", path, err)
	}
	raw, err := client.Request(ctx, "textDocument/diagnostic", map[string]any{
		"textDocument": map[string]any{"uri": uri},
	})
	if err != nil {
		return nil, fmt.Errorf("request diagnostics for %q: %w", path, err)
	}
	var result struct {
		Kind  *string         `json:"kind"`
		Items json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decode diagnostics for %q: %w", path, err)
	}
	if result.Kind == nil || (*result.Kind != "full" && *result.Kind != "") || len(result.Items) == 0 || bytes.Equal(bytes.TrimSpace(result.Items), []byte("null")) {
		return nil, fmt.Errorf("gopls returned incomplete diagnostics for %q (kind %v, result %s)", path, result.Kind, raw)
	}
	// gopls v0.23 omits the kind marker but returns a full items array for this fresh pull.
	var items []struct {
		Range struct {
			Start struct {
				Line      int `json:"line"`
				Character int `json:"character"`
			} `json:"start"`
		} `json:"range"`
		Message  string `json:"message"`
		Severity int    `json:"severity"`
	}
	if err := json.Unmarshal(result.Items, &items); err != nil {
		return nil, fmt.Errorf("decode diagnostic items for %q: %w", path, err)
	}
	if items == nil {
		return nil, fmt.Errorf("gopls returned null diagnostics for %q", path)
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return nil, fmt.Errorf("make diagnostic path relative: %w", err)
	}
	diagnostics := make([]Diagnostic, 0, len(items))
	for _, item := range items {
		diagnostics = append(diagnostics, Diagnostic{Message: item.Message, Severity: item.Severity, Path: filepath.ToSlash(relative), Line: item.Range.Start.Line + 1, Column: item.Range.Start.Character + 1, Source: "gopls"})
	}
	if err := client.Notify(ctx, "textDocument/didClose", map[string]any{"textDocument": map[string]any{"uri": uri}}); err != nil {
		return nil, fmt.Errorf("close Go source %q in gopls: %w", path, err)
	}
	return diagnostics, nil
}

func selectPackageFiles(root string, loaded []*packages.Package) ([]string, []Diagnostic, error) {
	excludedPath := func(path string) bool {
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return false
		}
		for part := range strings.SplitSeq(filepath.ToSlash(relative), "/") {
			if part == "." {
				continue
			}
			if part == "testdata" || part == "build" || part == "dist" || part == "target" || part == "vendor" || part == "node_modules" || part == ".scratch" || part == ".git" || strings.HasPrefix(part, "bazel-") || strings.HasPrefix(part, "_") || strings.HasPrefix(part, ".") {
				return true
			}
		}
		return false
	}
	fileSet := make(map[string]struct{})
	findingsSet := make(map[Diagnostic]struct{})
	for _, pkg := range loaded {
		includedFiles := 0
		for _, path := range pkg.GoFiles {
			if excludedPath(path) {
				continue
			}
			fileSet[path] = struct{}{}
			includedFiles++
		}
		if includedFiles == 0 {
			if len(pkg.GoFiles) == 0 && len(pkg.Errors) > 0 {
				return nil, nil, fmt.Errorf("incomplete Go package coverage for %q: package has errors but no selected Go files", pkg.ID)
			}
			continue
		}
		for _, packageErr := range pkg.Errors {
			findingsSet[Diagnostic{Message: packageErr.Msg, Severity: 1, Path: pkg.ID, Source: "package"}] = struct{}{}
		}
	}
	if len(fileSet) == 0 {
		return nil, nil, fmt.Errorf("no Go source files found in project root %q", root)
	}
	files := make([]string, 0, len(fileSet))
	for path := range fileSet {
		files = append(files, path)
	}
	slices.Sort(files)
	findings := make([]Diagnostic, 0, len(findingsSet))
	for finding := range findingsSet {
		findings = append(findings, finding)
	}
	slices.SortFunc(findings, func(a, b Diagnostic) int { return strings.Compare(a.String(), b.String()) })
	return files, findings, nil
}
