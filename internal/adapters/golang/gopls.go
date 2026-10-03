// Package golang provides the language adapter driving gopls for Go refactorings.
package golang

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"semedit/internal/backend"
	"semedit/internal/gocache"
	"semedit/internal/godiagnostics"
	"semedit/internal/sourcebuf"
)

var (
	// ErrGoplsNotFound indicates the gopls executable could not be located.
	ErrGoplsNotFound = errors.New("gopls executable not found")
	// ErrRenameFailed indicates gopls rename failed to execute.
	ErrRenameFailed = errors.New("gopls rename failed")
)

// FindGopls locates the gopls binary across system PATH and Go environment directories.
func FindGopls() (string, error) {
	executable, err := godiagnostics.FindGopls()
	if errors.Is(err, godiagnostics.ErrGoplsNotFound) {
		return "", fmt.Errorf("%w: %w", ErrGoplsNotFound, err)
	}
	return executable, err
}

// Rename executes gopls rename against an exact file coordinate.
func Rename(ctx context.Context, workDir string, file string, line int, col int, newName string) error {
	if !filepath.IsAbs(file) {
		file = filepath.Join(workDir, file)
	}
	executable, err := FindGopls()
	if err != nil {
		return err
	}
	environment, err := gocache.Environment(ctx, workDir)
	if err != nil {
		return fmt.Errorf("prepare gopls environment: %w", err)
	}
	if err := godiagnostics.UseWithContext(ctx, workDir, executable, environment, func(ctx context.Context, session *godiagnostics.Session) error {
		return renameInSession(ctx, session, file, line, col, newName)
	}); err != nil {
		return fmt.Errorf("%w: %w", ErrRenameFailed, err)
	}
	return nil
}

// RenameSelected binds the complete backend lookup snapshot to a shared-session rename.
func RenameSelected(ctx context.Context, workDir string, selected backend.SymbolCandidate, expected map[string][]byte, newName string) error {
	return renameSelected(ctx, workDir, selected, expected, newName, nil)
}

// RenameSelectedFiles reports only sources changed by the validated shared-session rename.
func RenameSelectedFiles(ctx context.Context, workDir string, selected backend.SymbolCandidate, expected map[string][]byte, newName string) ([]string, error) {
	paths := make([]string, 0)
	err := renameSelected(ctx, workDir, selected, expected, newName, func(changed []string) { paths = changed })
	if err != nil {
		return nil, err
	}
	return paths, nil
}

func renameSelected(ctx context.Context, workDir string, selected backend.SymbolCandidate, expected map[string][]byte, newName string, record func([]string)) error {
	file := selected.File
	if !filepath.IsAbs(file) {
		file = filepath.Join(workDir, file)
	}
	executable, err := FindGopls()
	if err != nil {
		return err
	}
	environment, err := gocache.Environment(ctx, workDir)
	if err != nil {
		return fmt.Errorf("prepare gopls environment: %w", err)
	}
	if err := godiagnostics.UseWithContext(ctx, workDir, executable, environment, func(ctx context.Context, session *godiagnostics.Session) error {
		document, err := session.Snapshot(file)
		if err != nil {
			return err
		}
		line, column, err := validateRenameLookupSnapshot(ctx, file, document.Content, selected, expected)
		if err != nil {
			return err
		}
		return renameInSessionFiles(ctx, session, file, line, column, newName, record)
	}); err != nil {
		return fmt.Errorf("%w: %w", ErrRenameFailed, err)
	}
	return nil
}

func validateRenameLookupSnapshot(ctx context.Context, file string, current []byte, selected backend.SymbolCandidate, expected map[string][]byte) (int, int, error) {
	if err := validateRenameExpectedFiles(ctx, expected); err != nil {
		return 0, 0, err
	}
	selectedPath, err := filepath.Abs(file)
	if err != nil {
		return 0, 0, fmt.Errorf("resolve rename lookup file: %w", err)
	}
	source, ok := expected[filepath.Clean(selectedPath)]
	if !ok {
		return 0, 0, errors.New("rename lookup lacks its selected source")
	}
	if !bytes.Equal(current, source) {
		return 0, 0, errors.New("rename source changed since symbol lookup")
	}
	if selected.Offset < 0 || selected.Offset+len(selected.Name) > len(source) || string(source[selected.Offset:selected.Offset+len(selected.Name)]) != selected.Name {
		return 0, 0, errors.New("rename identity does not match the lookup source")
	}
	prefix := source[:selected.Offset]
	line := bytes.Count(prefix, []byte{'\n'}) + 1
	column := selected.Offset - bytes.LastIndexByte(prefix, '\n')
	return line, column, nil
}

func validateRenameExpectedFiles(ctx context.Context, expected map[string][]byte) error {
	for path, source := range expected {
		canonical, err := filepath.EvalSymlinks(path)
		if err != nil {
			return fmt.Errorf("resolve rename lookup input: %w", err)
		}
		info, err := os.Stat(canonical)
		if err != nil {
			return fmt.Errorf("stat rename lookup input: %w", err)
		}
		if !info.Mode().IsRegular() {
			return errors.New("rename lookup input is not a regular file")
		}
		// #nosec G304 -- inputs come from the backend's exact resolver snapshot.
		input, err := sourcebuf.ReadFile(ctx, canonical)
		if err != nil {
			return fmt.Errorf("read rename lookup input: %w", err)
		}
		if !bytes.Equal(input, source) {
			return errors.New("rename analysis input changed since symbol lookup")
		}
	}
	return nil
}
