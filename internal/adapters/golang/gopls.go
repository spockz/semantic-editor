// Package golang provides the language adapter driving gopls for Go refactorings.
package golang

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backend/pathutil"
	"github.com/spockz/semantic-editor/internal/backend/readlsp"
	"github.com/spockz/semantic-editor/internal/gocache"
	"github.com/spockz/semantic-editor/internal/godiagnostics"
)

var (
	// ErrGoplsNotFound indicates the gopls executable could not be located.
	ErrGoplsNotFound = errors.New("gopls executable not found")
	// ErrRenameFailed indicates gopls rename failed to execute.
	ErrRenameFailed = errors.New("gopls rename failed")
	// ErrRenamePublicationConflict indicates a rename output changed before atomic publication.
	ErrRenamePublicationConflict = errors.New("rename publication conflict")
)

// RenamePublicationError reports outputs that were actually published before a rename failed.
type RenamePublicationError struct {
	PublishedPaths []string
	TotalFiles     int
	Cause          error
}

func (err *RenamePublicationError) Error() string {
	if len(err.PublishedPaths) > 0 {
		return fmt.Sprintf("rename published %d of %d files (%s); inspect the disk diff before continuing; remaining stale edits were not replayed: %v", len(err.PublishedPaths), err.TotalFiles, strings.Join(err.PublishedPaths, ", "), err.Cause)
	}
	return fmt.Sprintf("rename published 0 of %d files: %v", err.TotalFiles, err.Cause)
}

func (err *RenamePublicationError) Unwrap() error { return err.Cause }

// Retryable marks conflicts that happened before any output was published.
func (err *RenamePublicationError) Retryable() bool {
	return len(err.PublishedPaths) == 0 && errors.Is(err.Cause, ErrRenamePublicationConflict)
}

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
func RenameSelected(ctx context.Context, workDir string, selected backend.NamedTarget, expected map[string][]byte, newName string) error {
	return renameSelected(ctx, workDir, selected, expected, newName, nil)
}

// ValidateRenameSelected proves that gopls accepts the selected identifier for
// a rename without requesting edits or publishing source changes.
func ValidateRenameSelected(ctx context.Context, workDir string, selected backend.NamedTarget, expected map[string][]byte, newName string) error {
	return withSelectedRenameSession(ctx, workDir, selected, expected, func(ctx context.Context, session *godiagnostics.Session, document godiagnostics.Document, coordinate renameCoordinate) error {
		offset, err := selectedRenameOffset(document.Content, coordinate.line, coordinate.column, newName)
		if err != nil {
			return err
		}
		if _, prepareErr := prepareSelectedRename(ctx, session, document, offset); prepareErr != nil {
			return prepareErr
		}
		_, err = session.Snapshot(document.Path)
		return err
	})
}

// RenameSelectedFiles reports only sources changed by the validated shared-session rename.
func RenameSelectedFiles(ctx context.Context, workDir string, selected backend.NamedTarget, expected map[string][]byte, newName string) ([]string, error) {
	paths := make([]string, 0)
	err := renameSelected(ctx, workDir, selected, expected, newName, func(changed []string) { paths = changed })
	if err != nil {
		return paths, err
	}
	return paths, nil
}

func renameSelected(ctx context.Context, workDir string, selected backend.NamedTarget, expected map[string][]byte, newName string, record func([]string)) error {
	return withSelectedRenameSession(ctx, workDir, selected, expected, func(ctx context.Context, session *godiagnostics.Session, document godiagnostics.Document, coordinate renameCoordinate) error {
		return renameInSessionFiles(ctx, session, document.Path, coordinate.line, coordinate.column, newName, record)
	})
}

func withSelectedRenameSession(ctx context.Context, workDir string, selected backend.NamedTarget, expected map[string][]byte, action func(context.Context, *godiagnostics.Session, godiagnostics.Document, renameCoordinate) error) error {
	file, err := selectedFilePath(workDir, selected)
	if err != nil {
		return err
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
			return renameSourceConflict(err, 1)
		}
		coordinate, err := validateRenameLookupSnapshot(file, document.Content, selected, expected)
		if err != nil {
			return renameSourceConflict(err, 1)
		}
		return action(ctx, session, document, coordinate)
	}); err != nil {
		return fmt.Errorf("%w: %w", ErrRenameFailed, err)
	}
	return nil
}

func selectedFilePath(workDir string, selected backend.NamedTarget) (string, error) {
	file, err := pathutil.FilePathFromURI(selected.Location.URI)
	if err != nil {
		return "", fmt.Errorf("resolve rename target location: %w", err)
	}
	if !filepath.IsAbs(file) {
		file = filepath.Join(workDir, file)
	}
	return file, nil
}

type renameCoordinate struct {
	line   int
	column int
}

func validateRenameLookupSnapshot(file string, current []byte, selected backend.NamedTarget, expected map[string][]byte) (renameCoordinate, error) {
	selectedPath, err := filepath.Abs(file)
	if err != nil {
		return renameCoordinate{}, fmt.Errorf("resolve rename lookup file: %w", err)
	}
	source, ok := expected[filepath.Clean(selectedPath)]
	if !ok {
		return renameCoordinate{}, errors.New("rename lookup lacks its selected source")
	}
	if !bytes.Equal(current, source) {
		return renameCoordinate{}, fmt.Errorf("%w: rename source changed since symbol lookup", godiagnostics.ErrSourceChanged)
	}
	offset, err := readlsp.ByteOffset(source, selected.Location.Range.Start)
	if err != nil {
		return renameCoordinate{}, fmt.Errorf("map rename target position: %w", err)
	}
	if selected.Name == "" || offset+len(selected.Name) > len(source) || string(source[offset:offset+len(selected.Name)]) != selected.Name {
		return renameCoordinate{}, errors.New("rename identity does not match the lookup source")
	}
	prefix := source[:offset]
	line := bytes.Count(prefix, []byte{'\n'}) + 1
	column := offset - bytes.LastIndexByte(prefix, '\n')
	return renameCoordinate{line: line, column: column}, nil
}

func renameSourceConflict(err error, totalFiles int) error {
	if !errors.Is(err, godiagnostics.ErrSourceChanged) {
		return err
	}
	return &RenamePublicationError{
		TotalFiles: totalFiles,
		Cause:      errors.Join(ErrRenamePublicationConflict, godiagnostics.ErrSourceChanged, err),
	}
}
