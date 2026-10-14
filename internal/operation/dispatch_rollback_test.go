// Package operation tests dispatch rollback across handler and finalization failures.
package operation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func TestDispatchFailureRestoresOnlyCurrentSharedBufferEdit(t *testing.T) {
	for _, failurePoint := range []string{"handler", "finalizer"} {
		t.Run(failurePoint, func(t *testing.T) {
			assertDispatchFailureRollback(t, failurePoint)
		})
	}
}

func assertDispatchFailureRollback(t *testing.T, failurePoint string) {
	t.Helper()
	root, ctx, path, prior := dispatchRollbackBuffer(t)
	var observed [][]byte
	ctx = WithMutationObserver(ctx, func(observerCtx context.Context) error {
		content, readErr := sourcebuf.ReadFile(observerCtx, path)
		if readErr == nil {
			observed = append(observed, content)
		}
		return readErr
	})
	def := dispatchRollbackDef(path, failurePoint)
	registry := NewRegistry()
	if registerErr := Register(registry, def); registerErr != nil {
		t.Fatal(registerErr)
	}
	cc := CallContext{Ctx: ctx, WorkDir: root, Project: backend.ProjectContext{Language: backend.LanguageGo}}
	if _, dispatchErr := registry.Dispatch(cc, def.Key, map[string]any{}); dispatchErr == nil {
		t.Fatal("Dispatch succeeded after staged edit failure")
	}
	assertRestoredDispatchBuffer(ctx, t, path, prior, observed)
}

func dispatchRollbackBuffer(t *testing.T) (string, context.Context, string, []byte) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "value.go")
	if writeErr := os.WriteFile(path, []byte("package sample\nfunc Value() int { return 1 }\n"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	buffer, bufferErr := sourcebuf.New(root)
	if bufferErr != nil {
		t.Fatal(bufferErr)
	}
	ctx := sourcebuf.WithBuffer(context.Background(), buffer)
	if authorizeErr := buffer.AuthorizeWrites(ctx, root); authorizeErr != nil {
		t.Fatal(authorizeErr)
	}
	prior := []byte("package sample\nfunc Value() int { return 2 }\n")
	if stageErr := sourcebuf.WriteFile(ctx, path, prior); stageErr != nil {
		t.Fatal(stageErr)
	}
	return root, ctx, path, prior
}

func dispatchRollbackDef(path, failurePoint string) Def[struct{}, string] {
	def := Def[struct{}, string]{
		Key:   "dispatch_rollback_" + failurePoint,
		Parse: func(map[string]any) (struct{}, error) { return struct{}{}, nil },
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, struct{}) (string, error){
			backend.LanguageGo: func(editCtx context.Context, _ CallContext, _ struct{}) (string, error) {
				writeErr := sourcebuf.WriteFile(editCtx, path, []byte("package sample\nfunc Value() int { return 3 }\n"))
				if writeErr != nil {
					return "", writeErr
				}
				if failurePoint == "handler" {
					return "", errors.New("handler failed after staging")
				}
				return "edited", nil
			},
		},
	}
	if failurePoint == "finalizer" {
		def.FinalizeOutput = func(CallContext, struct{}, string) (string, error) {
			return "", errors.New("finalizer failed after staging")
		}
	}
	return def
}

func assertRestoredDispatchBuffer(ctx context.Context, t *testing.T, path string, prior []byte, observed [][]byte) {
	t.Helper()
	current, readErr := sourcebuf.ReadFile(ctx, path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(current) != string(prior) {
		t.Fatalf("working content = %q, want prior child content %q", current, prior)
	}
	if len(observed) < 2 || string(observed[len(observed)-1]) != string(prior) {
		t.Fatalf("mutation observer saw %q last; want restored content %q", observed, prior)
	}
}
