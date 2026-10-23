// Package astedit publishes prepared Go source edits through the buffer that owns their read set.
// WHY: Refactoring operations must preserve caller overlays and flush only buffers they created.
package astedit

import (
	"context"
	"fmt"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

type sourcePublication struct {
	path        string
	diffPath    string
	before      []byte
	after       []byte
	ownedBuffer *sourcebuf.Buffer
	standalone  bool
	operation   string
}

func publishPreparedSource(ctx context.Context, publication sourcePublication) (string, error) {
	if publication.path == "" || publication.diffPath == "" || publication.operation == "" {
		return "", fmt.Errorf("publish prepared source: path, diff path, and operation are required")
	}
	if publication.standalone && publication.ownedBuffer == nil {
		return "", fmt.Errorf("%s: standalone source publication has no owned buffer", publication.operation)
	}
	buffer := sourcebuf.FromContext(ctx)
	if buffer == nil {
		return "", fmt.Errorf("%s: source buffer is unavailable", publication.operation)
	}
	change := sourcebuf.Change{Path: publication.path, Before: publication.before, After: publication.after}
	if err := buffer.Apply(ctx, []sourcebuf.Change{change}); err != nil {
		return "", fmt.Errorf("%s: apply staged source: %w", publication.operation, err)
	}
	finalSource, err := sourcebuf.ReadFile(ctx, publication.path)
	if err != nil {
		return "", fmt.Errorf("%s: read staged source: %w", publication.operation, err)
	}
	if publication.standalone {
		if err := publication.ownedBuffer.Flush(ctx); err != nil {
			return "", fmt.Errorf("%s: flush owned source buffer: %w", publication.operation, err)
		}
	}
	return UnifiedDiff(publication.diffPath, string(publication.before), string(finalSource)), nil
}
