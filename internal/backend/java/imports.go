// Package java implements Java language-server operations and source transformations.
// WHY: This file applies import requests to the source buffer that JDT LS organizes.
package java

import (
	"bytes"
	"fmt"
	"regexp"
	"slices"

	"github.com/spockz/semantic-editor/internal/javacst"
)

var javaImportNamePattern = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*(\.[A-Za-z_$][A-Za-z0-9_$]*)+$`)

func applyJavaExplicitImports(source []byte, add, remove []string) ([]byte, error) {
	for _, name := range append(append([]string(nil), add...), remove...) {
		if !javaImportNamePattern.MatchString(name) {
			return nil, fmt.Errorf("invalid Java import name %q", name)
		}
	}
	updated := bytes.Clone(source)
	for _, name := range remove {
		var err error
		updated, err = removeJavaImport(updated, name)
		if err != nil {
			return nil, fmt.Errorf("remove Java import %q: %w", name, err)
		}
	}
	for _, name := range add {
		exists, err := hasJavaImport(updated, name)
		if err != nil {
			return nil, fmt.Errorf("inspect Java imports before adding %q: %w", name, err)
		}
		if exists {
			continue
		}
		updated, err = insertJavaImport(updated, name)
		if err != nil {
			return nil, fmt.Errorf("insert Java import %q: %w", name, err)
		}
	}
	return updated, nil
}

func hasJavaImport(source []byte, name string) (bool, error) {
	imports, err := javacst.ParseJavaImports(source)
	if err != nil {
		return false, err
	}
	wanted := javacst.JavaImportIdentity(name, false)
	for _, declaration := range imports.Declarations {
		if declaration.Identity == wanted {
			return true, nil
		}
	}
	return false, nil
}

func removeJavaImport(source []byte, name string) ([]byte, error) {
	imports, err := javacst.ParseJavaImports(source)
	if err != nil {
		return nil, err
	}
	wanted := javacst.JavaImportIdentity(name, false)
	type removalSpan struct {
		start int
		end   int
	}
	var removals []removalSpan
	for _, declaration := range imports.Declarations {
		if declaration.Identity != wanted {
			continue
		}
		removals = append(removals, removalSpan{start: declaration.Start, end: declaration.RemovalEnd})
	}
	if len(removals) == 0 {
		return bytes.Clone(source), nil
	}
	updated := bytes.Clone(source)
	for _, removal := range slices.Backward(removals) {

		if removal.start < 0 || removal.end < removal.start || removal.end > len(updated) {
			return nil, fmt.Errorf("java import removal span [%d,%d) is outside source of %d bytes", removal.start, removal.end, len(updated))
		}
		updated = append(updated[:removal.start], updated[removal.end:]...)
	}
	return updated, nil
}

func insertJavaImport(source []byte, name string) ([]byte, error) {
	return insertJavaImportDeclaration(source, []byte("import "+name+";"))
}

func restoreJavaImportsFromPreimage(source, preimage []byte) ([]byte, error) {
	original, err := javacst.ParseJavaImports(preimage)
	if err != nil {
		return nil, fmt.Errorf("parse Java preimage imports: %w", err)
	}
	updated := bytes.Clone(source)
	for _, declaration := range original.Declarations {
		current, err := javacst.ParseJavaImports(updated)
		if err != nil {
			return nil, fmt.Errorf("parse Java postimage imports: %w", err)
		}
		found := false
		for _, existing := range current.Declarations {
			if existing.Identity == declaration.Identity {
				found = true
				break
			}
		}
		if found {
			continue
		}
		updated, err = insertJavaImportDeclaration(updated, declaration.Text)
		if err != nil {
			return nil, fmt.Errorf("restore Java import %q: %w", declaration.Identity, err)
		}
	}
	return updated, nil
}

func insertJavaImportDeclaration(source, declaration []byte) ([]byte, error) {
	imports, err := javacst.ParseJavaImports(source)
	if err != nil {
		return nil, fmt.Errorf("parse Java imports before insertion: %w", err)
	}
	eol := []byte("\n")
	if bytes.Contains(source, []byte("\r\n")) {
		eol = []byte("\r\n")
	}
	addition := make([]byte, 0, len(declaration)+2*len(eol))
	if imports.InsertionOffset > 0 && source[imports.InsertionOffset-1] != '\n' {
		addition = append(addition, eol...)
	}
	if imports.HasPackage && len(imports.Declarations) == 0 {
		addition = append(addition, eol...)
	}
	addition = append(addition, declaration...)
	addition = append(addition, eol...)
	if imports.InsertionBlankLine {
		addition = append(addition, eol...)
	}
	updated := make([]byte, 0, len(source)+len(addition))
	updated = append(updated, source[:imports.InsertionOffset]...)
	updated = append(updated, addition...)
	updated = append(updated, source[imports.InsertionWhitespaceEnd:]...)
	return updated, nil
}
