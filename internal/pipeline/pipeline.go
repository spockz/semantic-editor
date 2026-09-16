// Package pipeline orchestrates atomic writes, code formatting, and compiler diagnostic reporting.
package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/tools/imports"
)

var (
	reMissingPackage = regexp.MustCompile(`cannot find package "([^"]+)"`)
	reNoModule       = regexp.MustCompile(`no required module provides package ([^;:\s]+)`)
)

// WriteAtomic writes data to a file atomically via sibling temp file, fsync, and rename.
func WriteAtomic(targetPath string, data []byte) error {
	dir := filepath.Dir(targetPath)
	base := filepath.Base(targetPath)
	tmpFile, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Ensure advancing mtime before renaming (ADR-0010)
	if fi, err := os.Stat(targetPath); err == nil {
		oldTime := fi.ModTime()
		now := time.Now()
		if !now.After(oldTime) {
			newTime := oldTime.Add(2 * time.Millisecond)
			_ = os.Chtimes(tmpName, newTime, newTime)
		}
	}

	if err := os.Rename(tmpName, targetPath); err != nil {
		return fmt.Errorf("rename to target: %w", err)
	}

	return nil
}

// Format runs gofmt on the specified paths or directories.
func Format(ctx context.Context, workDir string, paths ...string) error {
	args := append([]string{"-w"}, paths...)
	// #nosec G204 -- canonical formatter invocation
	cmd := exec.CommandContext(ctx, "gofmt", args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Env = os.Environ()

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gofmt failed: %w: %s", err, errBuf.String())
	}
	return nil
}

// DiagnosticDelta records compiler diagnostic shifts across an edit (ADR-0004, RQ-0006).
type DiagnosticDelta struct {
	Before      []string `json:"before"`
	After       []string `json:"after"`
	NetDelta    int      `json:"net_delta"`
	Introduced  []string `json:"introduced"`
	Resolved    []string `json:"resolved"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// ComputeDelta calculates introduced and resolved diagnostics between two states.
func ComputeDelta(before []string, after []string) DiagnosticDelta {
	beforeMap := make(map[string]bool, len(before))
	for _, b := range before {
		beforeMap[b] = true
	}

	afterMap := make(map[string]bool, len(after))
	for _, a := range after {
		afterMap[a] = true
	}

	var introduced []string
	for _, a := range after {
		if !beforeMap[a] {
			introduced = append(introduced, a)
		}
	}

	var resolved []string
	for _, b := range before {
		if !afterMap[b] {
			resolved = append(resolved, b)
		}
	}

	var suggestions []string
	seenSuggestion := make(map[string]bool)
	for _, intro := range introduced {
		if m := reMissingPackage.FindStringSubmatch(intro); len(m) > 1 {
			pkg := m[1]
			sug := fmt.Sprintf("Run 'go get %s' or use semantic_add_dependency to install the missing dependency.", pkg)
			if !seenSuggestion[sug] {
				suggestions = append(suggestions, sug)
				seenSuggestion[sug] = true
			}
		} else if m := reNoModule.FindStringSubmatch(intro); len(m) > 1 {
			pkg := m[1]
			sug := fmt.Sprintf("Run 'go get %s' or use semantic_add_dependency to install the missing dependency.", pkg)
			if !seenSuggestion[sug] {
				suggestions = append(suggestions, sug)
				seenSuggestion[sug] = true
			}
		}
	}

	return DiagnosticDelta{
		Before:      before,
		After:       after,
		NetDelta:    len(after) - len(before),
		Introduced:  introduced,
		Resolved:    resolved,
		Suggestions: suggestions,
	}
}

// OrganizeImports adjusts imports and formats the given paths using golang.org/x/tools/imports.
func OrganizeImports(_ context.Context, workDir string, paths ...string) error {
	if len(paths) == 0 {
		paths = []string{"."}
	}

	var goFiles []string
	for _, p := range paths {
		target := p
		if workDir != "" && !filepath.IsAbs(target) {
			target = filepath.Join(workDir, target)
		}
		fi, err := os.Stat(target)
		if err != nil {
			return fmt.Errorf("stat path %s: %w", p, err)
		}
		if fi.IsDir() {
			err := filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					name := info.Name()
					if name == ".git" || name == ".scratch" || name == "vendor" || name == "node_modules" {
						return filepath.SkipDir
					}
					return nil
				}
				if strings.HasSuffix(info.Name(), ".go") {
					goFiles = append(goFiles, path)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("walk dir %s: %w", target, err)
			}
		} else if strings.HasSuffix(target, ".go") {
			goFiles = append(goFiles, target)
		}
	}

	for _, file := range goFiles {
		data, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			return fmt.Errorf("read file %s: %w", file, err)
		}

		res, err := imports.Process(file, data, nil)
		if err != nil {
			return fmt.Errorf("organize imports for %s: %w", file, err)
		}

		if !bytes.Equal(data, res) {
			if err := WriteAtomic(file, res); err != nil {
				return fmt.Errorf("write organized file %s: %w", file, err)
			}
		}
	}
	return nil
}

// FindModuleRoot locates the nearest enclosing directory containing go.mod.
func FindModuleRoot(dir string) string {
	cur := filepath.Clean(dir)
	for {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return dir
}

// CheckDiagnostics collects compiler/linter diagnostics without rolling back intermediate states (ADR-0004).
func CheckDiagnostics(ctx context.Context, workDir string) ([]string, error) {
	effectiveDir := FindModuleRoot(workDir)

	cmd := exec.CommandContext(ctx, "go", "vet", "./...")
	if effectiveDir != "" {
		cmd.Dir = effectiveDir
	}
	cmd.Env = os.Environ()

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	_ = cmd.Run() // Exit status may be non-zero on compiler errors, which is normal for diagnostic reporting.

	output := out.String()
	if output == "" {
		return nil, nil
	}

	var diagnostics []string
	for line := range bytes.SplitSeq([]byte(output), []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 {
			diagnostics = append(diagnostics, string(trimmed))
		}
	}
	return diagnostics, nil
}
