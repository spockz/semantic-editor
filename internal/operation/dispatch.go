// Package operation dispatches ingress-neutral calls to per-language operation handlers.
package operation

import (
	"context"
	"errors"
	"fmt"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/backends"
	"github.com/spockz/semantic-editor/internal/gocache"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

// ErrUnknownOperation indicates no operation is registered for a key.
var ErrUnknownOperation = errors.New("unknown operation")

// ErrUnsupportedLanguage indicates an operation has no handler for the resolved language.
var ErrUnsupportedLanguage = errors.New("operation is not supported for language")

// Dispatch looks up key and invokes the registered entry with Parse, language resolution,
// and per-language handler selection performed inside the entry closure.
func (r *Registry) Dispatch(cc CallContext, key string, raw map[string]any) (any, error) {
	if r == nil {
		return nil, fmt.Errorf("dispatch %q: %w", key, ErrUnknownOperation)
	}
	entry, ok := r.byKey[key]
	if !ok {
		return nil, fmt.Errorf("dispatch %q: %w", key, ErrUnknownOperation)
	}
	return r.DispatchEntry(cc, entry, raw)
}

// DispatchEntry invokes one resolved operation entry through the shared validation path.
func (r *Registry) DispatchEntry(cc CallContext, entry Entry, raw map[string]any) (any, error) {
	if r == nil {
		return nil, fmt.Errorf("dispatch %q: %w", entry.Key, ErrUnknownOperation)
	}
	ctx := cc.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = gocache.WithEnvironmentCache(ctx)
	cc.Ctx = ctx
	buffer := sourcebuf.FromContext(ctx)
	var checkpoint sourcebuf.Checkpoint
	if buffer != nil {
		checkpoint = buffer.Checkpoint()
	}
	result, err := entry.Invoke(cc, raw)
	if err != nil {
		cause := rollbackDispatchError(ctx, buffer, checkpoint, !entry.ReadOnly, err)
		return nil, suggestFailedTarget(cc, entry, raw, cause)
	}
	if entry.FinalizeOutput != nil {
		result, err = entry.FinalizeOutput(cc, raw, result)
		if err != nil {
			return nil, rollbackDispatchError(ctx, buffer, checkpoint, !entry.ReadOnly, err)
		}
	}
	recordSuccessfulTargetChange(ctx, entry.Key, raw, result)
	return result, nil
}

func rollbackDispatchError(ctx context.Context, buffer *sourcebuf.Buffer, checkpoint sourcebuf.Checkpoint, resyncObserver bool, cause error) error {
	if buffer == nil {
		return cause
	}
	buffer.Rollback(checkpoint)
	if observer, ok := ctx.Value(mutationObserverContextKey{}).(func(context.Context) error); ok && observer != nil && resyncObserver {
		return errors.Join(cause, observer(ctx))
	}
	return cause
}

// resolveLanguage selects the backend for project, mirroring ingress language detection,
// so handler selection follows the same explicit-or-auto rules as the backend service.
func resolveLanguage(cc CallContext, project backend.ProjectContext) (backend.LanguageID, error) {
	service := cc.Service
	if service == nil {
		service = backends.NewDefaultService()
	}
	selected, err := service.Registry.Select(project)
	if err != nil {
		return "", fmt.Errorf("resolve operation language: %w", errors.Join(err, ErrUnsupportedLanguage))
	}
	return selected.Language(), nil
}
