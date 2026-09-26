// Package astedit tests exact metadata from read-only source projections.
package astedit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOutlineGoPreservesCRLFCommentsAndTags(t *testing.T) {
	root := t.TempDir()
	source := []byte("package demo\r\n/*\r\n * Public docs\r\n */\r\ntype Public struct { Value int `json:\"name\r\nvalue\"` }\r\n")
	path := filepath.Join(root, "source.go")
	if err := os.WriteFile(path, source, 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := OutlineGo(root, path, nil, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 || len(result.Files[0].Symbols) != 1 {
		t.Fatalf("outline = %#v", result)
	}
	typeSymbol := result.Files[0].Symbols[0]
	if typeSymbol.Doc == nil || *typeSymbol.Doc != "/*\r\n * Public docs\r\n */" {
		t.Fatalf("type doc = %#v", typeSymbol.Doc)
	}
	if len(typeSymbol.Children) != 1 || typeSymbol.Children[0].Tag == nil {
		t.Fatalf("type children = %#v", typeSymbol.Children)
	}
	wantTag := "`json:\"name\r\nvalue\"`"
	if *typeSymbol.Children[0].Tag != wantTag {
		t.Fatalf("tag = %q, want %q", *typeSymbol.Children[0].Tag, wantTag)
	}
}
