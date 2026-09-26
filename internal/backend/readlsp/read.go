// Package readlsp projects validated language-server symbols onto exact source snapshots.
package readlsp

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"semedit/internal/backend"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"
)

// InspectMatch pairs a selected symbol with the range semantics claimed by its backend.
type InspectMatch struct {
	Symbol       backend.ReadSymbol
	SourceExtent string
}

// Document is one immutable source snapshot with backend-validated mapped symbols.
type Document struct {
	Root, File  string
	Language    backend.LanguageID
	Source      []byte
	Symbols     []backend.ReadSymbol
	Package     string
	Doc         *string
	Imports     []backend.ReadImport
	Complete    bool
	Limitations []string
}

// ValidateOutlineRequest rejects scope and filter options unsupported by external read backends.
func ValidateOutlineRequest(request backend.OutlineRequest, directory bool) error {
	if strings.TrimSpace(request.Path) == "" {
		return fmt.Errorf("outline path is required")
	}
	if directory {
		return fmt.Errorf("external outlines support selected files only")
	}
	if !request.IncludeUnexported {
		return fmt.Errorf("external outlines do not support visibility filtering")
	}
	for _, kind := range request.Kinds {
		if !canonicalReadKind(kind) {
			return fmt.Errorf("unknown outline kind %q", kind)
		}
	}
	return nil
}

// Outline projects a validated selected-file snapshot into the shared read DTO.
func Outline(document Document, request backend.OutlineRequest) (*backend.OutlineResult, error) {
	if err := ValidateOutlineRequest(request, false); err != nil {
		return nil, err
	}
	prepared, err := prepareDocument(document)
	if err != nil {
		return nil, err
	}
	symbols, err := copySymbols(document.Symbols, prepared)
	if err != nil {
		return nil, err
	}
	kinds := make(map[string]bool, len(request.Kinds))
	for _, kind := range request.Kinds {
		kinds[kind] = true
	}
	projected := filterSymbols(symbols, kinds)
	sortSymbols(projected)
	imports := copyImports(document.Imports)
	return &backend.OutlineResult{Scope: backend.ReadScope{Kind: "selected_file", Path: prepared.displayFile, Complete: document.Complete, Limitations: slices.Clone(document.Limitations)}, Files: []backend.OutlineFile{{File: prepared.displayFile, Language: document.Language, Revision: prepared.revision, Package: document.Package, Doc: copyString(document.Doc), Imports: imports, Symbols: projected}}}, nil
}

// Inspect slices each selected declaration from the exact validated source snapshot.
func Inspect(document Document, matches []InspectMatch) (*backend.InspectResult, error) {
	if len(matches) == 0 {
		return nil, fmt.Errorf("inspection requires at least one matched declaration")
	}
	prepared, err := prepareDocument(document)
	if err != nil {
		return nil, err
	}
	if _, err := copySymbols(document.Symbols, prepared); err != nil {
		return nil, err
	}
	inspected := make([]backend.InspectedSymbol, 0, len(matches))
	for _, match := range matches {
		if match.SourceExtent != "declaration" && match.SourceExtent != "server_range" {
			return nil, fmt.Errorf("unsupported source extent %q", match.SourceExtent)
		}
		symbol, err := copySymbol(match.Symbol, prepared)
		if err != nil {
			return nil, err
		}
		start, err := sourceByteOffset(document.Source, prepared.index, symbol.Range.Start)
		if err != nil {
			return nil, fmt.Errorf("map declaration start: %w", err)
		}
		end, err := sourceByteOffset(document.Source, prepared.index, symbol.Range.End)
		if err != nil {
			return nil, fmt.Errorf("map declaration end: %w", err)
		}
		if start >= end {
			return nil, fmt.Errorf("declaration %q has an empty source range", symbol.QualifiedName)
		}
		sortSymbols(symbol.Children)
		inspected = append(inspected, backend.InspectedSymbol{ReadSymbol: symbol, Source: string(document.Source[start:end]), Revision: prepared.revision, SourceExtent: match.SourceExtent, Package: document.Package, Imports: copyImports(document.Imports)})
	}
	sort.SliceStable(inspected, func(i, j int) bool { return symbolLess(inspected[i].ReadSymbol, inspected[j].ReadSymbol) })
	return &backend.InspectResult{Scope: backend.ReadScope{Kind: "selected_file", Path: prepared.displayFile, Complete: document.Complete, Limitations: slices.Clone(document.Limitations)}, Matches: inspected}, nil
}

// ByteOffset maps one zero-based UTF-16 position to a byte offset in its source snapshot.
func ByteOffset(source []byte, position backend.Position) (int, error) {
	index, err := newSourceIndex(source)
	if err != nil {
		return 0, err
	}
	return sourceByteOffset(source, index, position)
}

type preparedDocument struct {
	root, file, displayFile string
	index                   *sourceIndex
	revision                string
}

type sourceLine struct{ start, end int }

type sourceIndex struct {
	source []byte
	lines  []sourceLine
}

func prepareDocument(document Document) (preparedDocument, error) {
	if !filepath.IsAbs(document.Root) || !filepath.IsAbs(document.File) {
		return preparedDocument{}, fmt.Errorf("read projection root and file must be absolute")
	}
	root := filepath.Clean(document.Root)
	file := filepath.Clean(document.File)
	index, err := newSourceIndex(document.Source)
	if err != nil {
		return preparedDocument{}, err
	}
	display := filepath.ToSlash(file)
	if relative, err := filepath.Rel(root, file); err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative) {
		display = filepath.ToSlash(relative)
	}
	digest := sha256.Sum256(document.Source)
	return preparedDocument{root: root, file: file, displayFile: display, index: index, revision: fmt.Sprintf("%x", digest)}, nil
}

func newSourceIndex(source []byte) (*sourceIndex, error) {
	if !utf8.Valid(source) {
		return nil, fmt.Errorf("source snapshot is not valid UTF-8")
	}
	lines := make([]sourceLine, 0, bytes.Count(source, []byte("\n"))+1)
	start := 0
	for offset := 0; offset < len(source); {
		if source[offset] != 13 && source[offset] != 10 {
			offset++
			continue
		}
		end := offset
		if source[offset] == 13 && offset+1 < len(source) && source[offset+1] == 10 {
			offset += 2
		} else {
			offset++
		}
		lines = append(lines, sourceLine{start: start, end: end})
		start = offset
	}
	lines = append(lines, sourceLine{start: start, end: len(source)})
	return &sourceIndex{source: source, lines: lines}, nil
}

func sourceByteOffset(source []byte, index *sourceIndex, position backend.Position) (int, error) {
	if position.Line < 0 || position.Character < 0 || position.Line >= len(index.lines) {
		return 0, fmt.Errorf("source position is outside the snapshot")
	}
	line := index.lines[position.Line]
	units := 0
	for offset := line.start; offset < line.end; {
		if units == position.Character {
			return offset, nil
		}
		r, size := utf8.DecodeRune(source[offset:line.end])
		if r == utf8.RuneError && size == 1 {
			return 0, fmt.Errorf("source snapshot is not valid UTF-8")
		}
		width := 1
		if r > 0xffff {
			width = 2
		}
		if position.Character > units && position.Character < units+width {
			return 0, fmt.Errorf("source position splits a UTF-16 surrogate pair")
		}
		units += width
		offset += size
	}
	if units == position.Character {
		return line.end, nil
	}
	return 0, fmt.Errorf("source character is outside the line")
}

func copySymbols(symbols []backend.ReadSymbol, document preparedDocument) ([]backend.ReadSymbol, error) {
	copied := make([]backend.ReadSymbol, 0, len(symbols))
	for _, symbol := range symbols {
		clone, err := copySymbol(symbol, document)
		if err != nil {
			return nil, err
		}
		copied = append(copied, clone)
	}
	return copied, nil
}

func copySymbol(symbol backend.ReadSymbol, document preparedDocument) (backend.ReadSymbol, error) {
	if symbol.Name == "" || !canonicalReadKind(symbol.Kind) {
		return backend.ReadSymbol{}, fmt.Errorf("mapped symbol has an unsupported name or kind")
	}
	mappedFile := symbol.File
	if !filepath.IsAbs(mappedFile) {
		mappedFile = filepath.Join(document.root, mappedFile)
	}
	mappedFile, err := filepath.Abs(mappedFile)
	if err != nil {
		return backend.ReadSymbol{}, fmt.Errorf("resolve mapped symbol file: %w", err)
	}
	if filepath.Clean(mappedFile) != document.file {
		return backend.ReadSymbol{}, fmt.Errorf("mapped symbol %q belongs to %q, not selected file %q", symbol.Name, mappedFile, document.file)
	}
	rangeStart, err := sourceByteOffset(document.index.source, document.index, symbol.Range.Start)
	if err != nil {
		return backend.ReadSymbol{}, fmt.Errorf("map range start for %q: %w", symbol.Name, err)
	}
	rangeEnd, err := sourceByteOffset(document.index.source, document.index, symbol.Range.End)
	if err != nil {
		return backend.ReadSymbol{}, fmt.Errorf("map range end for %q: %w", symbol.Name, err)
	}
	selectionStart, err := sourceByteOffset(document.index.source, document.index, symbol.SelectionRange.Start)
	if err != nil {
		return backend.ReadSymbol{}, fmt.Errorf("map selection start for %q: %w", symbol.Name, err)
	}
	selectionEnd, err := sourceByteOffset(document.index.source, document.index, symbol.SelectionRange.End)
	if err != nil {
		return backend.ReadSymbol{}, fmt.Errorf("map selection end for %q: %w", symbol.Name, err)
	}
	if rangeStart > rangeEnd || selectionStart >= selectionEnd || selectionStart < rangeStart || selectionEnd > rangeEnd {
		return backend.ReadSymbol{}, fmt.Errorf("mapped symbol %q has invalid or uncontained source ranges", symbol.Name)
	}
	clone := symbol
	clone.File = document.displayFile
	clone.Signature = copyString(symbol.Signature)
	clone.Doc = copyString(symbol.Doc)
	clone.Type = copyString(symbol.Type)
	clone.Tag = copyString(symbol.Tag)
	clone.ServerDetail = copyString(symbol.ServerDetail)
	if symbol.Children != nil {
		children, err := copySymbols(symbol.Children, document)
		if err != nil {
			return backend.ReadSymbol{}, err
		}
		clone.Children = children
	}
	return clone, nil
}

func canonicalReadKind(kind string) bool {
	return backend.IsReadOutlineKind(kind)
}

func copyString(value *string) *string {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func copyImports(imports []backend.ReadImport) []backend.ReadImport {
	if imports == nil {
		return nil
	}
	copied := make([]backend.ReadImport, len(imports))
	for i, item := range imports {
		copied[i] = backend.ReadImport{Path: item.Path, Name: copyString(item.Name)}
	}
	return copied
}

func filterSymbols(symbols []backend.ReadSymbol, kinds map[string]bool) []backend.ReadSymbol {
	if len(kinds) == 0 {
		return symbols
	}
	filtered := make([]backend.ReadSymbol, 0, len(symbols))
	for _, symbol := range symbols {
		children := filterSymbols(symbol.Children, kinds)
		if kinds[symbol.Kind] || len(children) > 0 {
			symbol.Children = children
			filtered = append(filtered, symbol)
		}
	}
	return filtered
}

func sortSymbols(symbols []backend.ReadSymbol) {
	sort.SliceStable(symbols, func(i, j int) bool { return symbolLess(symbols[i], symbols[j]) })
	for i := range symbols {
		sortSymbols(symbols[i].Children)
	}
}

func symbolLess(left, right backend.ReadSymbol) bool {
	if left.File != right.File {
		return left.File < right.File
	}
	if left.SelectionRange.Start.Line != right.SelectionRange.Start.Line {
		return left.SelectionRange.Start.Line < right.SelectionRange.Start.Line
	}
	if left.SelectionRange.Start.Character != right.SelectionRange.Start.Character {
		return left.SelectionRange.Start.Character < right.SelectionRange.Start.Character
	}
	if left.SelectionRange.End.Line != right.SelectionRange.End.Line {
		return left.SelectionRange.End.Line < right.SelectionRange.End.Line
	}
	if left.SelectionRange.End.Character != right.SelectionRange.End.Character {
		return left.SelectionRange.End.Character < right.SelectionRange.End.Character
	}
	if left.Name != right.Name {
		return left.Name < right.Name
	}
	return left.Kind < right.Kind
}
