// Package operation verifies lifecycle operations preserve evidence that invalidates stale rename suggestions.
package operation

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func TestInvalidateSuggestionAliasesTombstonesContainingTypeRename(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "reader.go")
	buffer, err := sourcebuf.New(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{File: file, OldSelector: "Parser", NewSelector: "Reader", Kind: "type"})
	InvalidateSuggestionAliases(buffer, string(backend.OperationReplaceConstruct), map[string]any{"file": file, "symbol": "Reader.ParseConfig"})
	if aliases := buffer.SymbolAliases(); len(aliases) != 0 {
		t.Fatalf("live aliases after replacement = %#v", aliases)
	}
	if invalidated := buffer.InvalidatedSymbolAliases(); len(invalidated) != 1 || invalidated[0].OldSelector != "Parser" {
		t.Fatalf("replacement tombstones = %#v", invalidated)
	}
}
