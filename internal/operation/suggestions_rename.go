// Package operation records native Java rename identities without losing
// overload signatures or the resource path of a renamed containing type.
package operation

import (
	"strings"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

func recordJavaRenameAlias(buffer *sourcebuf.Buffer, file, selector, to, kind, newFile string) {
	if to == "" {
		return
	}
	end := strings.IndexByte(selector, '(')
	if end < 0 {
		end = len(selector)
	}
	start := strings.LastIndexByte(selector[:end], '.') + 1
	if selector[start:end] == to {
		return
	}
	buffer.RecordSymbolRename(sourcebuf.SymbolAlias{
		Language: "java", File: file, NewFile: newFile,
		OldSelector: selector, NewSelector: selector[:start] + to + selector[end:], Kind: kind,
	})
}
