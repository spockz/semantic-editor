// Package operation dispatches ingress-neutral calls to per-language operation handlers.
package operation

import (
	"context"
	"errors"
	"fmt"

	"semedit/internal/backend"
	"semedit/internal/backends"
	"semedit/internal/sourcebuf"
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
	ctx := cc.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	buffer := sourcebuf.FromContext(ctx)
	var checkpoint sourcebuf.Checkpoint
	if buffer != nil {
		checkpoint = buffer.Checkpoint()
	}
	result, err := entry.Invoke(cc, raw)
	if err != nil {
		return nil, rollbackDispatchError(ctx, buffer, checkpoint, !entry.ReadOnly, err)
	}
	if entry.FinalizeOutput != nil {
		result, err = entry.FinalizeOutput(cc, raw, result)
		if err != nil {
			return nil, rollbackDispatchError(ctx, buffer, checkpoint, !entry.ReadOnly, err)
		}
	}
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
