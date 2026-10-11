// Package sourcebuf keeps batch source revisions private so AST and LSP operations share one working revision.
package sourcebuf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"semedit/internal/atomicfile"
	"slices"
	"strings"
	"sync"
)

// Change describes one source revision and its expected working preimage.
type Change struct {
	Path          string
	Before, After []byte
}

// Buffer retains source preimages and working bytes for one buffered batch segment.
type Buffer struct {
	mu         sync.Mutex
	root       string
	writeRoots map[string]bool
	files      map[string]*fileState
	trees      map[string]map[string]bool
	aliases    map[string]string
}

// ChangedPaths reports outputs whose working bytes differ from their disk preimages.
func (b *Buffer) ChangedPaths() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	paths := make([]string, 0)
	for path, state := range b.files {
		if !bytes.Equal(state.original, state.content) {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	return paths
}

// AssertDisk rejects changed source sets and disk preimages used by semantic analysis.
func (b *Buffer) AssertDisk() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.assertDisk()
}

// Flush stages every output before publishing; a late failure reports the published count.
func (b *Buffer) Flush(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	defer func() { clear(b.files); clear(b.trees); clear(b.aliases) }()
	if err := b.assertDisk(); err != nil {
		return err
	}
	paths := make([]string, 0)
	for path, state := range b.files {
		if !bytes.Equal(state.original, state.content) {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	staged := make([]*atomicfile.StagedFile, 0, len(paths))
	defer func() {
		for _, file := range staged {
			file.Discard()
		}
	}()
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("stage buffered batch: %w", err)
		}
		file, err := atomicfile.Stage(path, b.files[path].content)
		if err != nil {
			return fmt.Errorf("stage buffered batch: %w", err)
		}
		staged = append(staged, file)
	}
	if err := b.assertDisk(); err != nil {
		return err
	}
	for index, file := range staged {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("batch published %d of %d files: %w", index, len(staged), err)
		}
		if err := b.assertDisk(); err != nil {
			return fmt.Errorf("batch published %d of %d files: %w", index, len(staged), err)
		}
		if err := file.Publish(); err != nil {
			return fmt.Errorf("batch published %d of %d files: %w", index, len(staged), err)
		}
		state := b.files[paths[index]]
		state.original = bytes.Clone(state.content)
		info, err := os.Stat(paths[index])
		if err != nil {
			return fmt.Errorf("batch published %d of %d files: stat published output: %w", index+1, len(staged), err)
		}
		state.info = info
	}
	return nil
}

// AuthorizeWrites binds the main module before semantic analysis and permits its outputs.
func (b *Buffer) AuthorizeWrites(ctx context.Context, root string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("authorize buffered module: %w", err)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve writable module: %w", err)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return fmt.Errorf("canonicalize writable module: %w", err)
	}
	if err := ObserveTree(WithBuffer(ctx, b), canonical); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.writeRoots == nil {
		b.writeRoots = make(map[string]bool)
	}
	b.writeRoots[canonical] = true
	return nil
}

// Apply changes all outputs of one semantic operation after validating every preimage.
func (b *Buffer) Apply(ctx context.Context, changes []Change) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("apply buffered operation: %w", err)
	}
	pending := make(map[string][]byte, len(changes))
	for _, change := range changes {
		absolute, err := filepath.Abs(change.Path)
		if err != nil {
			return fmt.Errorf("resolve buffered output: %w", err)
		}
		canonical, state, err := b.load(absolute)
		if err != nil {
			return err
		}
		if canonical != absolute || !b.writable(canonical) || filepath.Ext(canonical) != ".go" || state.info == nil {
			return fmt.Errorf("invalid buffered output %q", change.Path)
		}
		if _, exists := pending[canonical]; exists {
			return fmt.Errorf("duplicate buffered output %q", change.Path)
		}
		if !bytes.Equal(change.Before, state.content) {
			return fmt.Errorf("%w: working preimage of %s changed", ErrConflict, canonical)
		}
		pending[canonical] = bytes.Clone(change.After)
	}
	if err := b.assertDisk(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("apply buffered operation: %w", err)
	}
	for path, content := range pending {
		b.files[path].content = content
	}
	return nil
}

// Changes reports original and working bytes for unpublished outputs.
func (b *Buffer) Changes() []Change {
	b.mu.Lock()
	defer b.mu.Unlock()
	changes := make([]Change, 0)
	for _, path := range slices.Sorted(maps.Keys(b.files)) {
		state := b.files[path]
		if !bytes.Equal(state.original, state.content) {
			changes = append(changes, Change{Path: path, Before: bytes.Clone(state.original), After: bytes.Clone(state.content)})
		}
	}
	return changes
}

// ChangesSince reports this edit's changes relative to a checkpoint's working revision.
func (b *Buffer) ChangesSince(checkpoint Checkpoint) []Change {
	b.mu.Lock()
	defer b.mu.Unlock()
	changes := make([]Change, 0)
	for _, path := range slices.Sorted(maps.Keys(b.files)) {
		state := b.files[path]
		before, exists := checkpoint.content[path]
		if !exists {
			before = state.original
		}
		if bytes.Equal(before, state.content) {
			continue
		}
		changes = append(changes, Change{Path: path, Before: bytes.Clone(before), After: bytes.Clone(state.content)})
	}
	return changes
}

// AuthorizedRoots reports main module roots explicitly permitted for publication.
func (b *Buffer) AuthorizedRoots() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return slices.Sorted(maps.Keys(b.writeRoots))
}

// Checkpoint captures the current working bytes so a failed child edit can be rolled back.
func (b *Buffer) Checkpoint() Checkpoint {
	b.mu.Lock()
	defer b.mu.Unlock()
	checkpoint := Checkpoint{
		content:    make(map[string][]byte, len(b.files)),
		writeRoots: maps.Clone(b.writeRoots),
	}
	for path, state := range b.files {
		checkpoint.content[path] = bytes.Clone(state.content)
	}
	return checkpoint
}

// Rollback restores the captured bytes while retaining inputs observed by the failed edit.
func (b *Buffer) Rollback(checkpoint Checkpoint) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for path, state := range b.files {
		content, ok := checkpoint.content[path]
		if !ok {
			content = state.original
		}
		state.content = bytes.Clone(content)
	}
	b.writeRoots = maps.Clone(checkpoint.writeRoots)
}

// ErrConflict identifies stale disk inputs to a buffered transformation.
var ErrConflict = errors.New("buffered source changed on disk")

// New binds the workspace source set before a batch segment begins.
func New(root string) (*Buffer, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve buffer root: %w", err)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fmt.Errorf("canonicalize buffer root: %w", err)
	}
	sources, err := sourceSet(canonical)
	if err != nil {
		return nil, err
	}
	buffer := &Buffer{root: canonical, files: make(map[string]*fileState), trees: map[string]map[string]bool{canonical: sources}, aliases: make(map[string]string)}
	for _, path := range slices.Sorted(maps.Keys(sources)) {
		if _, _, err := buffer.load(path); err != nil {
			return nil, err
		}
	}
	return buffer, nil
}

// WithBuffer attaches the source view shared by AST and LSP operations.
func WithBuffer(ctx context.Context, buffer *Buffer) context.Context {
	return context.WithValue(ctx, contextKey{}, buffer)
}

// WithoutBuffer selects disk operations at a publication barrier.
func WithoutBuffer(ctx context.Context) context.Context { return WithBuffer(ctx, nil) }

// FromContext returns the current batch source view, or nil for direct execution.
func FromContext(ctx context.Context) *Buffer {
	if ctx == nil {
		return nil
	}
	buffer, _ := ctx.Value(contextKey{}).(*Buffer)
	return buffer
}

// ReadFile captures a disk preimage once and returns the current working bytes.
func ReadFile(ctx context.Context, path string) ([]byte, error) {
	buffer := FromContext(ctx)
	if buffer == nil {
		return os.ReadFile(filepath.Clean(path))
	}
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	path, state, err := buffer.load(path)
	if err != nil {
		return nil, err
	}
	if state.info == nil {
		return nil, &os.PathError{Op: "read", Path: path, Err: os.ErrNotExist}
	}
	return bytes.Clone(state.content), nil
}

// WriteFile buffers existing Go sources during a batch and publishes directly otherwise.
func WriteFile(ctx context.Context, path string, data []byte) error {
	return WriteFileChecked(ctx, path, data, nil)
}

// WriteFileChecked validates semantic preimages before changing the working revision.
func WriteFileChecked(ctx context.Context, path string, data []byte, check func() error) error {
	buffer := FromContext(ctx)
	if buffer == nil {
		return atomicfile.WriteAtomicChecked(path, data, check)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("buffer source: %w", err)
	}
	if check != nil {
		if err := check(); err != nil {
			return fmt.Errorf("validate buffered source: %w", err)
		}
	}
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	originalPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve buffered output: %w", err)
	}
	canonical, state, err := buffer.load(originalPath)
	if err != nil {
		return err
	}
	if !buffer.writable(canonical) || filepath.Ext(canonical) != ".go" || state.info == nil || canonical != originalPath {
		return fmt.Errorf("buffered output %q must be an existing regular Go file beneath the workspace without symlink aliases", path)
	}
	state.content = bytes.Clone(data)
	return nil
}

// Overlay returns independent copies of source bytes for package loading.
func Overlay(ctx context.Context) map[string][]byte {
	buffer := FromContext(ctx)
	if buffer == nil {
		return nil
	}
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	overlay := make(map[string][]byte)
	for path, state := range buffer.files {
		if state.info != nil && filepath.Ext(path) == ".go" {
			overlay[path] = bytes.Clone(state.content)
		}
	}
	return overlay
}

// Dirty reports whether a source has unpublished bytes.
func Dirty(ctx context.Context, path string) bool {
	buffer := FromContext(ctx)
	if buffer == nil {
		return false
	}
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	state, ok := buffer.files[path]
	return ok && !bytes.Equal(state.original, state.content)
}

// ObserveTree binds source membership in local replacement modules.
func ObserveTree(ctx context.Context, root string) error {
	buffer := FromContext(ctx)
	if buffer == nil {
		return nil
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("canonicalize source tree: %w", err)
	}
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	if _, ok := buffer.trees[root]; ok {
		return nil
	}
	files, err := sourceSet(root)
	if err != nil {
		return err
	}
	for _, path := range slices.Sorted(maps.Keys(files)) {
		if _, _, err := buffer.load(path); err != nil {
			return err
		}
	}
	buffer.trees[root] = files
	return nil
}

// ParseFile reads through the batch source view when no explicit source is supplied.
func ParseFile(ctx context.Context, fset *token.FileSet, path string, source any, mode parser.Mode) (*ast.File, error) {
	if source == nil {
		data, err := ReadFile(ctx, path)
		if err != nil {
			return nil, err
		}
		source = data
	}
	return parser.ParseFile(fset, path, source, mode)
}

// Run binds an owned buffer for a standalone edit or rolls back a failed shared-buffer edit.
func Run(ctx context.Context, root string, edit func(context.Context) error) error {
	if edit == nil {
		return fmt.Errorf("run buffered edit: nil edit")
	}
	buffer := FromContext(ctx)
	var checkpoint Checkpoint
	if buffer != nil {
		checkpoint = buffer.Checkpoint()
	}
	editCtx, buffer, owned, err := Prepare(ctx, root)
	if err != nil {
		return err
	}
	if owned {
		checkpoint = buffer.Checkpoint()
	}
	if err := edit(editCtx); err != nil {
		buffer.Rollback(checkpoint)
		return err
	}
	if owned {
		if err := buffer.Flush(editCtx); err != nil {
			return fmt.Errorf("publish buffered edit: %w", err)
		}
	}
	return nil
}

// Prepare binds and authorizes an edit buffer, preserving any batch buffer already in context.
func Prepare(ctx context.Context, root string) (context.Context, *Buffer, bool, error) {
	buffer := FromContext(ctx)
	owned := buffer == nil
	if owned {
		var err error
		buffer, err = New(root)
		if err != nil {
			return nil, nil, false, fmt.Errorf("capture buffered edit workspace: %w", err)
		}
		ctx = WithBuffer(ctx, buffer)
	}
	if err := buffer.AuthorizeWrites(ctx, root); err != nil {
		return nil, nil, false, fmt.Errorf("authorize buffered edit: %w", err)
	}
	return ctx, buffer, owned, nil
}

func (b *Buffer) load(path string) (string, *fileState, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", nil, fmt.Errorf("resolve buffered path: %w", err)
	}
	if canonical, canonicalErr := filepath.EvalSymlinks(path); canonicalErr == nil {
		if path != canonical {
			b.aliases[path] = canonical
		}
		path = canonical
	} else if !errors.Is(canonicalErr, os.ErrNotExist) {
		return "", nil, fmt.Errorf("canonicalize buffered path: %w", canonicalErr)
	}
	if state, ok := b.files[path]; ok {
		return path, state, nil
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		state := &fileState{}
		b.files[path] = state
		return path, state, nil
	}
	if err != nil {
		return "", nil, fmt.Errorf("stat buffered input: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("buffered input %q is not regular", path)
	}
	// #nosec G304 -- path is a canonical regular input captured for this buffer.
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("read buffered input: %w", err)
	}
	after, err := os.Stat(path)
	if err != nil {
		return "", nil, fmt.Errorf("restat buffered input: %w", err)
	}
	if !os.SameFile(info, after) || info.Size() != after.Size() || !info.ModTime().Equal(after.ModTime()) {
		return "", nil, fmt.Errorf("%w: %s changed while reading", ErrConflict, path)
	}
	state := &fileState{original: bytes.Clone(data), content: data, info: info}
	b.files[path] = state
	return path, state, nil
}

func (b *Buffer) assertDisk() error {
	for path, expected := range b.aliases {
		canonical, err := filepath.EvalSymlinks(path)
		if err != nil || canonical != expected {
			return fmt.Errorf("%w: source alias %s changed", ErrConflict, path)
		}
	}

	for root, expected := range b.trees {
		actual, err := sourceSet(root)
		if err != nil {
			return err
		}
		if !maps.Equal(actual, expected) {
			return fmt.Errorf("%w: source set beneath %s changed", ErrConflict, root)
		}
	}
	for _, path := range slices.Sorted(maps.Keys(b.files)) {
		state := b.files[path]
		info, err := os.Lstat(path)
		if state.info == nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("%w: missing input %s appeared", ErrConflict, path)
		}
		if err != nil {
			return fmt.Errorf("%w: stat %s: %w", ErrConflict, path, err)
		}
		if !info.Mode().IsRegular() || !os.SameFile(state.info, info) || info.Mode() != state.info.Mode() {
			return fmt.Errorf("%w: identity or permissions of %s changed", ErrConflict, path)
		}
		// #nosec G304 -- path is a canonical regular input retained in the buffer read set.
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%w: read %s: %w", ErrConflict, path, err)
		}
		if !bytes.Equal(data, state.original) {
			return fmt.Errorf("%w: %s", ErrConflict, path)
		}
	}
	return nil
}

func (b *Buffer) writable(path string) bool {
	for root := range b.writeRoots {
		relative, err := filepath.Rel(root, path)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return true
		}
	}
	relative, err := filepath.Rel(b.root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

type contextKey struct{}

type fileState struct {
	original, content []byte
	info              os.FileInfo
}

func sourceSet(root string) (map[string]bool, error) {
	files := make(map[string]bool)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("scan buffered source set: %w", err)
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == ".git" || entry.Name() == ".scratch" || entry.Name() == "vendor" || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".go" {
			files[path] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// Checkpoint captures buffer state so a failed child edit can restore its parent revision.
type Checkpoint struct {
	content    map[string][]byte
	writeRoots map[string]bool
}

// Content returns a copy of one source revision captured by the checkpoint.
func (checkpoint Checkpoint) Content(path string) ([]byte, bool) {
	content, ok := checkpoint.content[path]
	return bytes.Clone(content), ok
}
