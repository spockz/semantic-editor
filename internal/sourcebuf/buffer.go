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
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/spockz/semantic-editor/internal/gocache"

	"github.com/spockz/semantic-editor/internal/atomicfile"
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
	exclusions []string
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

// DirtySourceBytes reports the retained original and working bytes for changed paths.
func (b *Buffer) DirtySourceBytes() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	var total int64
	for _, state := range b.files {
		if !bytes.Equal(state.original, state.content) {
			total += int64(len(state.original) + len(state.content))
		}
	}
	return total
}

func (b *Buffer) forkLocked(original bool) *Buffer {
	fork := &Buffer{
		root:       b.root,
		exclusions: slices.Clone(b.exclusions),
		writeRoots: maps.Clone(b.writeRoots),
		files:      make(map[string]*fileState, len(b.files)),
		trees:      make(map[string]map[string]bool, len(b.trees)),
		aliases:    maps.Clone(b.aliases),
	}
	for path, state := range b.files {
		preimage := bytes.Clone(state.original)
		content := bytes.Clone(state.content)
		if original {
			content = bytes.Clone(preimage)
		}
		fork.files[path] = &fileState{original: preimage, content: content, info: state.info, parents: maps.Clone(state.parents)}
	}
	for root, files := range b.trees {
		fork.trees[root] = maps.Clone(files)
	}
	return fork
}

// Fork returns an independent retained working revision.
func (b *Buffer) Fork() *Buffer {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.forkLocked(false)
}

// ForkOriginal returns an independent retained baseline with working bytes reset to preimages.
func (b *Buffer) ForkOriginal() *Buffer {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.forkLocked(true)
}

// MergeReadSetFrom retains source inputs observed by candidate without accepting its edits.
func (b *Buffer) MergeReadSetFrom(candidate *Buffer) error {
	candidate.mu.Lock()
	snapshot := candidate.forkLocked(false)
	candidate.mu.Unlock()
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.root != snapshot.root {
		return errors.New("merge buffered read set: roots differ")
	}
	for alias, resolved := range snapshot.aliases {
		if prior, ok := b.aliases[alias]; ok && prior != resolved {
			return fmt.Errorf("%w: source alias %s changed", ErrConflict, alias)
		}
	}
	for root, files := range snapshot.trees {
		if prior, ok := b.trees[root]; ok && !maps.Equal(prior, files) {
			return fmt.Errorf("%w: source set beneath %s changed", ErrConflict, root)
		}
	}
	for path, state := range snapshot.files {
		if prior, ok := b.files[path]; ok && !bytes.Equal(prior.original, state.original) {
			return fmt.Errorf("%w: source preimage differs for %s", ErrConflict, path)
		}
	}
	for path, state := range snapshot.files {
		if _, ok := b.files[path]; !ok {
			b.files[path] = &fileState{original: bytes.Clone(state.original), content: bytes.Clone(state.original), info: state.info, parents: maps.Clone(state.parents)}
		}
	}
	for root, files := range snapshot.trees {
		b.trees[root] = maps.Clone(files)
	}
	maps.Copy(b.aliases, snapshot.aliases)
	if b.writeRoots == nil {
		b.writeRoots = make(map[string]bool)
	}
	maps.Copy(b.writeRoots, snapshot.writeRoots)
	return nil
}

// AssertDisk rejects changed source sets and disk preimages used by semantic analysis.
func (b *Buffer) AssertDisk() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.assertDisk()
}

// Flush stages every output before publishing and then discards the working revision.
func (b *Buffer) Flush(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	defer func() { clear(b.files); clear(b.trees); clear(b.aliases) }()
	_, err := b.publishLocked(ctx)
	return err
}

// PublishRetained publishes changed files while retaining the buffer for explicit lifecycle handling.
func (b *Buffer) PublishRetained(ctx context.Context) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.publishLocked(ctx)
}

func (b *Buffer) publishLocked(ctx context.Context) (published int, err error) {
	diskErr := b.assertDisk()
	if diskErr != nil {
		return 0, diskErr
	}
	paths := b.changedPathsLocked()
	parentErr := b.assertNewFileParents()
	if parentErr != nil {
		return 0, parentErr
	}
	publication, stageErr := b.stagePublication(ctx, paths)
	defer func() {
		for _, file := range publication.files {
			file.Discard()
		}
		if published == 0 {
			cleanupErr := removeUnpublishedDirectories(publication.createdDirs)
			if cleanupErr != nil {
				err = errors.Join(err, cleanupErr)
			}
		}
	}()
	if stageErr != nil {
		return 0, stageErr
	}
	return b.publishStaged(ctx, publication)
}

type stagedPublication struct {
	paths       []string
	files       []*atomicfile.StagedFile
	createdDirs []string
}

func (b *Buffer) changedPathsLocked() []string {
	paths := make([]string, 0, len(b.files))
	for path, state := range b.files {
		if !bytes.Equal(state.original, state.content) {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	return paths
}

func (b *Buffer) stagePublication(ctx context.Context, paths []string) (stagedPublication, error) {
	publication := stagedPublication{paths: paths, files: make([]*atomicfile.StagedFile, 0, len(paths))}
	created := make(map[string]bool)
	for _, path := range paths {
		if contextErr := ctx.Err(); contextErr != nil {
			return publication, fmt.Errorf("stage retained buffer: %w", contextErr)
		}
		if parentErr := b.createPublicationParents(path, created, &publication.createdDirs); parentErr != nil {
			return publication, parentErr
		}
		if parentErr := b.assertNewFileParents(); parentErr != nil {
			return publication, parentErr
		}
		file, stageErr := atomicfile.Stage(path, b.files[path].content)
		if stageErr != nil {
			return publication, fmt.Errorf("stage retained buffer: %w", stageErr)
		}
		publication.files = append(publication.files, file)
	}
	return publication, nil
}

func (b *Buffer) createPublicationParents(path string, created map[string]bool, createdDirs *[]string) error {
	missingDirs, err := missingPublicationParents(path)
	if err != nil {
		return err
	}
	for _, directory := range slices.Backward(missingDirs) {
		mkdirErr := os.Mkdir(directory, 0o750)
		switch {
		case mkdirErr == nil:
			if ownedErr := b.recordOwnedNewFileParent(directory); ownedErr != nil {
				return ownedErr
			}
			if !created[directory] {
				*createdDirs = append(*createdDirs, directory)
				created[directory] = true
			}
		case errors.Is(mkdirErr, os.ErrExist):
			if err := regularOutputDirectory(directory); err != nil {
				return err
			}
		default:
			return fmt.Errorf("create retained output directory: %w", mkdirErr)
		}
	}
	return nil
}

func missingPublicationParents(path string) ([]string, error) {
	missing := make([]string, 0)
	for directory := filepath.Dir(path); ; directory = filepath.Dir(directory) {
		statErr := regularOutputDirectory(directory)
		if statErr == nil {
			return missing, nil
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			return nil, statErr
		}
		missing = append(missing, directory)
		if parent := filepath.Dir(directory); parent == directory {
			return nil, fmt.Errorf("find existing output parent for %s", path)
		}
	}
}

func regularOutputDirectory(directory string) error {
	info, err := os.Stat(directory)
	if err != nil {
		return fmt.Errorf("inspect retained output directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("output parent %s is not a directory", directory)
	}
	return nil
}

func (b *Buffer) publishStaged(ctx context.Context, publication stagedPublication) (int, error) {
	if diskErr := b.assertDisk(); diskErr != nil {
		return 0, diskErr
	}
	for index, file := range publication.files {
		if contextErr := ctx.Err(); contextErr != nil {
			return index, fmt.Errorf("retained buffer published %d of %d files: %w", index, len(publication.files), contextErr)
		}
		if parentErr := b.assertNewFileParents(); parentErr != nil {
			return index, fmt.Errorf("retained buffer published %d of %d files: %w", index, len(publication.files), parentErr)
		}
		if diskErr := b.assertDisk(); diskErr != nil {
			return index, fmt.Errorf("retained buffer published %d of %d files: %w", index, len(publication.files), diskErr)
		}
		if publishErr := file.Publish(); publishErr != nil {
			return index, fmt.Errorf("retained buffer published %d of %d files: %w", index, len(publication.files), publishErr)
		}
		if stateErr := b.recordPublishedFile(publication.paths[index]); stateErr != nil {
			return index + 1, fmt.Errorf("retained buffer published %d of %d files: %w", index+1, len(publication.files), stateErr)
		}
	}
	return len(publication.files), nil
}

func (b *Buffer) recordPublishedFile(path string) error {
	state := b.files[path]
	state.original = bytes.Clone(state.content)
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat output: %w", err)
	}
	state.info = info
	for root, files := range b.trees {
		rel, relErr := filepath.Rel(root, path)
		if relErr == nil && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			files[path] = true
		}
	}
	return nil
}

func removeUnpublishedDirectories(directories []string) error {
	slices.SortFunc(directories, func(left, right string) int {
		leftDepth := strings.Count(left, string(filepath.Separator))
		rightDepth := strings.Count(right, string(filepath.Separator))
		if leftDepth != rightDepth {
			return rightDepth - leftDepth
		}
		return strings.Compare(left, right)
	})
	var cleanupErr error
	for _, directory := range directories {
		removeErr := os.Remove(directory)
		if removeErr == nil || errors.Is(removeErr, os.ErrNotExist) {
			continue
		}
		cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove unpublished output directory %s: %w", directory, removeErr))
	}
	return cleanupErr
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
		if canonical != absolute || !b.writable(canonical) || (filepath.Ext(canonical) != ".go" && filepath.Ext(canonical) != ".java") || (state.info == nil && state.content == nil) {
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
func New(ctx context.Context, root string) (*Buffer, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve buffer root: %w", err)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fmt.Errorf("canonicalize buffer root: %w", err)
	}
	exclusions, err := sourceExclusions(ctx, canonical)
	if err != nil {
		return nil, err
	}
	sources, err := sourceSet(canonical, exclusions)
	if err != nil {
		return nil, err
	}
	buffer := &Buffer{root: canonical, exclusions: exclusions, files: make(map[string]*fileState), trees: map[string]map[string]bool{canonical: sources}, aliases: make(map[string]string)}
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
		if state.content != nil {
			return bytes.Clone(state.content), nil
		}
		return nil, &os.PathError{Op: "read", Path: path, Err: os.ErrNotExist}
	}
	return bytes.Clone(state.content), nil
}

// CanonicalPath resolves an existing source path or an accepted buffered new Go or Java source without creating parent directories.
func CanonicalPath(ctx context.Context, path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve buffered source path %q: %w", path, err)
	}
	canonical, canonicalErr := filepath.EvalSymlinks(absolute)
	if canonicalErr == nil {
		return canonical, nil
	}
	if !errors.Is(canonicalErr, os.ErrNotExist) {
		return "", fmt.Errorf("canonicalize buffered source path %q: %w", path, canonicalErr)
	}
	buffer := FromContext(ctx)
	if buffer == nil {
		return "", fmt.Errorf("canonicalize buffered source path %q: %w", path, canonicalErr)
	}
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	canonical, state, err := buffer.load(absolute)
	if err != nil {
		return "", err
	}
	if state.info == nil && state.content != nil {
		if parentErr := buffer.assertNewFileParents(); parentErr != nil {
			return "", parentErr
		}
		return canonical, nil
	}
	return "", fmt.Errorf("canonicalize buffered source path %q: %w", path, canonicalErr)
}

// WriteFile buffers supported Go or Java sources during a batch and publishes directly otherwise.
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
	if !buffer.writable(canonical) || (filepath.Ext(canonical) != ".go" && filepath.Ext(canonical) != ".java") || canonical != originalPath {
		return fmt.Errorf("buffered output %q must be supported source beneath the workspace without symlink aliases", path)
	}
	if state.info == nil && state.content == nil {
		parents, err := buffer.captureNewFileParents(canonical)
		if err != nil {
			return err
		}
		state.parents = parents
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
		if (state.info != nil || state.content != nil) && filepath.Ext(path) == ".go" {
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
	files, err := sourceSet(root, buffer.exclusions)
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
		buffer, err = New(ctx, root)
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
	} else if errors.Is(canonicalErr, os.ErrNotExist) {
		missingCanonical, missingErr := canonicalMissingPath(path)
		if missingErr != nil {
			return "", nil, missingErr
		}
		path = missingCanonical
	} else {
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

func canonicalMissingPath(path string) (string, error) {
	missing := make([]string, 0)
	for current := path; ; current = filepath.Dir(current) {
		canonical, err := filepath.EvalSymlinks(current)
		if err == nil {
			for _, m := range slices.Backward(missing) {
				canonical = filepath.Join(canonical, m)
			}
			return canonical, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("canonicalize missing buffered path: %w", err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("find existing parent of buffered path %s", path)
		}
		missing = append(missing, filepath.Base(current))
	}
}

func (b *Buffer) captureNewFileParents(path string) (map[string]os.FileInfo, error) {
	root := b.root
	for writeRoot := range b.writeRoots {
		relative, err := filepath.Rel(writeRoot, path)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && len(writeRoot) > len(root) {
			root = writeRoot
		}
	}
	parents := make(map[string]os.FileInfo)
	for directory := filepath.Dir(path); ; directory = filepath.Dir(directory) {
		info, err := os.Lstat(directory)
		switch {
		case err == nil:
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("new buffered output parent %s is not a regular directory", directory)
			}
			parents[directory] = info
		case errors.Is(err, os.ErrNotExist):
			parents[directory] = nil
		default:
			return nil, fmt.Errorf("inspect new buffered output parent %s: %w", directory, err)
		}
		if directory == root {
			return parents, nil
		}
		if parent := filepath.Dir(directory); parent == directory {
			return nil, fmt.Errorf("new buffered output %s is outside an authorized root", path)
		}
	}
}

func (b *Buffer) assertNewFileParents() error {
	for path, state := range b.files {
		if state.info != nil || state.content == nil || state.parents == nil {
			continue
		}
		for directory, expected := range state.parents {
			actual, err := os.Lstat(directory)
			if expected == nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				if err != nil {
					return fmt.Errorf("inspect new buffered output parent %s: %w", directory, err)
				}
				return fmt.Errorf("%w: new output parent %s appeared", ErrConflict, directory)
			}
			if err != nil {
				return fmt.Errorf("%w: stat new output parent %s: %w", ErrConflict, directory, err)
			}
			if !actual.IsDir() || actual.Mode()&os.ModeSymlink != 0 || !os.SameFile(expected, actual) || actual.Mode() != expected.Mode() {
				return fmt.Errorf("%w: new output parent %s changed", ErrConflict, directory)
			}
		}
		if filepath.Ext(path) != ".go" && filepath.Ext(path) != ".java" {
			return fmt.Errorf("new buffered output %s is not supported source", path)
		}
	}
	return nil
}

func (b *Buffer) recordOwnedNewFileParent(directory string) error {
	info, err := os.Lstat(directory)
	if err != nil {
		return fmt.Errorf("inspect created buffered output parent %s: %w", directory, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("created buffered output parent %s is not a regular directory", directory)
	}
	for _, state := range b.files {
		if state.parents != nil {
			if expected, ok := state.parents[directory]; ok && expected == nil {
				state.parents[directory] = info
			}
		}
	}
	return nil
}

func (b *Buffer) assertDisk() error {
	for path, expected := range b.aliases {
		canonical, err := filepath.EvalSymlinks(path)
		if err != nil || canonical != expected {
			return fmt.Errorf("%w: source alias %s changed", ErrConflict, path)
		}
	}

	for root, expected := range b.trees {
		actual, err := sourceSet(root, b.exclusions)
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
	parents           map[string]os.FileInfo
}

func sourceSet(root string, exclusions []string) (map[string]bool, error) {
	files := make(map[string]bool)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("scan buffered source set: %w", err)
		}
		if entry.IsDir() {
			if (SourceDirectoryFilter{exclusions: exclusions}).Skip(root, path) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".go" || filepath.Ext(path) == ".java" {
			files[path] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// SourceDirectoryFilter excludes generated runtime trees from source snapshots.
type SourceDirectoryFilter struct {
	exclusions []string
}

// GeneratedRoots returns a copy of the runtime roots excluded by this filter.
func (filter SourceDirectoryFilter) GeneratedRoots() []string {
	return slices.Clone(filter.exclusions)
}

// DirectoryFilterFor reuses a buffer's frozen exclusions or resolves a new snapshot policy.
func DirectoryFilterFor(ctx context.Context, root string) (SourceDirectoryFilter, error) {
	if buffer := FromContext(ctx); buffer != nil {
		buffer.mu.Lock()
		defer buffer.mu.Unlock()
		return SourceDirectoryFilter{exclusions: slices.Clone(buffer.exclusions)}, nil
	}
	exclusions, err := sourceExclusions(ctx, root)
	if err != nil {
		return SourceDirectoryFilter{}, err
	}
	return SourceDirectoryFilter{exclusions: exclusions}, nil
}

// Skip reports whether path is a generated or ignored directory strictly below root.
func (filter SourceDirectoryFilter) Skip(root, path string) bool {
	if path == root {
		return false
	}
	if slices.Contains(filter.exclusions, path) {
		return true
	}
	switch filepath.Base(path) {
	case ".git", ".scratch", "vendor", "node_modules":
		return true
	default:
		return false
	}
}

func sourceExclusions(ctx context.Context, root string) ([]string, error) {
	roots := append([]string{os.TempDir()}, environmentSourceRoots(os.Environ())...)
	exclusions, err := normalizeSourceExclusions(root, roots)
	if err != nil {
		return nil, err
	}
	hasGo, err := containsGoSources(root, exclusions)
	if err != nil {
		return nil, err
	}
	if !hasGo && !gocache.HasBaseDir(ctx) {
		return exclusions, nil
	}
	effective, err := gocache.Environment(ctx, root)
	if err != nil {
		if !gocache.HasBaseDir(ctx) && errors.Is(err, exec.ErrNotFound) {
			return exclusions, nil
		}
		return nil, fmt.Errorf("resolve effective Go source exclusions: %w", err)
	}
	roots = append(roots, environmentSourceRoots(effective)...)
	return normalizeSourceExclusions(root, roots)
}

func environmentSourceRoots(environment []string) []string {
	roots := make([]string, 0, 8)
	for _, entry := range environment {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || value == "" {
			continue
		}
		switch key {
		case "GOCACHE", "GOMODCACHE", "GOTMPDIR", "GOPLSCACHE", "TMPDIR", "TEMP", "TMP":
			roots = append(roots, value)
		case "GOPATH":
			for _, path := range filepath.SplitList(value) {
				if path != "" {
					roots = append(roots, filepath.Join(path, "pkg", "mod"))
				}
			}
		}
	}
	return roots
}

func normalizeSourceExclusions(root string, paths []string) ([]string, error) {
	exclusions := make([]string, 0, len(paths)*2)
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve generated source root %q: %w", path, err)
		}
		absolute = filepath.Clean(absolute)
		if !slices.Contains(exclusions, absolute) {
			exclusions = append(exclusions, absolute)
		}
		canonical, err := filepath.EvalSymlinks(absolute)
		if err == nil {
			if !slices.Contains(exclusions, canonical) {
				exclusions = append(exclusions, canonical)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("canonicalize generated source root %q: %w", absolute, err)
		}
	}
	return exclusions, nil
}

func containsGoSources(root string, exclusions []string) (bool, error) {
	found := false
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("probe buffered Go sources: %w", err)
		}
		if entry.IsDir() {
			if (SourceDirectoryFilter{exclusions: exclusions}).Skip(root, path) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() == "go.mod" || entry.Name() == "go.work" || filepath.Ext(path) == ".go" {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return found, nil
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
