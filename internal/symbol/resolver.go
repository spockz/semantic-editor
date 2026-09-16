// Package symbol provides AST symbol resolution and coordinate translation for Go source files.
package symbol

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrNotFound indicates the queried symbol does not exist in scope.
	ErrNotFound = errors.New("symbol not found")
	// ErrAmbiguous indicates multiple matching symbols were discovered.
	ErrAmbiguous = errors.New("ambiguous symbol query")
)

// Symbol represents an identified symbol in source code with exact coordinates.
type Symbol struct {
	Name          string `json:"name"`
	Receiver      string `json:"receiver,omitempty"`
	QualifiedName string `json:"qualified_name,omitempty"`
	Kind          string `json:"kind"`
	File          string `json:"file"`
	Line          int    `json:"line"`
	Column        int    `json:"column"`
	Offset        int    `json:"offset"`
}

// BuildQualifiedName returns the formatted name including receiver if present.
func (s *Symbol) BuildQualifiedName() string {
	if s.Receiver != "" {
		return s.Receiver + "." + s.Name
	}
	return s.Name
}

// LookupResult holds the resolved symbol information or ambiguous candidate options.
type LookupResult struct {
	Symbol     string    `json:"symbol,omitempty"`
	File       string    `json:"file,omitempty"`
	Line       int       `json:"line,omitempty"`
	Column     int       `json:"column,omitempty"`
	Offset     int       `json:"offset,omitempty"`
	Kind       string    `json:"kind,omitempty"`
	Receiver   string    `json:"receiver,omitempty"`
	Ambiguous  bool      `json:"ambiguous,omitempty"`
	Candidates []*Symbol `json:"candidates,omitempty"`
}

// ParseIdentifier decomposes a query identifier into receiver and symbol name.
func ParseIdentifier(raw string) (receiver string, name string, err error) {
	clean := strings.Trim(strings.TrimSpace(raw), `"'`)
	if clean == "" {
		return "", "", fmt.Errorf("empty identifier: %w", ErrNotFound)
	}

	parts := strings.Split(clean, ".")
	switch len(parts) {
	case 1:
		return "", parts[0], nil
	case 2:
		recv := strings.TrimPrefix(parts[0], "*")
		recv = strings.TrimPrefix(recv, "(")
		recv = strings.TrimSuffix(recv, ")")
		recv = strings.TrimPrefix(recv, "*")
		return recv, parts[1], nil
	default:
		return "", "", fmt.Errorf("invalid qualified identifier %q: too many segments", raw)
	}
}

// Resolve locates a symbol within a target file or workspace directory.
func Resolve(rootDir string, filePath string, query string) (*LookupResult, error) {
	recv, name, err := ParseIdentifier(query)
	if err != nil {
		return nil, err
	}

	var targetFiles []string
	if filePath != "" {
		resolvedPath := filePath
		if !filepath.IsAbs(resolvedPath) && rootDir != "" {
			resolvedPath = filepath.Join(rootDir, filePath)
		}
		targetFiles = append(targetFiles, resolvedPath)
	} else {
		searchRoot := rootDir
		if searchRoot == "" {
			searchRoot = "."
		}
		err := filepath.WalkDir(searchRoot, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				base := d.Name()
				if strings.HasPrefix(base, ".") || base == "vendor" || base == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(d.Name(), ".go") && !strings.HasSuffix(d.Name(), "_test.go") {
				targetFiles = append(targetFiles, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to scan workspace: %w", err)
		}
	}

	var matches []*Symbol
	for _, file := range targetFiles {
		syms, err := scanFile(file, recv, name)
		if err != nil {
			return nil, fmt.Errorf("failed scanning %s: %w", file, err)
		}
		matches = append(matches, syms...)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, query)
	}

	for _, m := range matches {
		m.QualifiedName = m.BuildQualifiedName()
		if rootDir != "" {
			if rel, err := filepath.Rel(rootDir, m.File); err == nil && !strings.HasPrefix(rel, "..") {
				m.File = rel
			}
		}
	}

	if len(matches) > 1 {
		return &LookupResult{
			Ambiguous:  true,
			Candidates: matches,
		}, nil
	}

	match := matches[0]
	return &LookupResult{
		Symbol:    match.QualifiedName,
		File:      match.File,
		Line:      match.Line,
		Column:    match.Column,
		Offset:    match.Offset,
		Kind:      match.Kind,
		Receiver:  match.Receiver,
		Ambiguous: false,
	}, nil
}

func scanFile(filePath string, targetRecv string, targetName string) ([]*Symbol, error) {
	cleanPath := filepath.Clean(filePath)
	// #nosec G304 -- reading verified target go source files
	src, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse file: %w", err)
	}

	var results []*Symbol

	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			recvName := extractReceiver(d.Recv)
			if d.Name.Name == targetName {
				if targetRecv == "" || targetRecv == recvName {
					pos := fset.Position(d.Name.Pos())
					kind := "function"
					if recvName != "" {
						kind = "method"
					}
					results = append(results, &Symbol{
						Name:     d.Name.Name,
						Receiver: recvName,
						Kind:     kind,
						File:     filePath,
						Line:     pos.Line,
						Column:   pos.Column,
						Offset:   pos.Offset,
					})
				}
			}
		case *ast.GenDecl:
			if targetRecv != "" {
				continue
			}
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.Name == targetName {
						pos := fset.Position(s.Name.Pos())
						results = append(results, &Symbol{
							Name:   s.Name.Name,
							Kind:   "type",
							File:   filePath,
							Line:   pos.Line,
							Column: pos.Column,
							Offset: pos.Offset,
						})
					}
				case *ast.ValueSpec:
					for _, ident := range s.Names {
						if ident.Name == targetName {
							pos := fset.Position(ident.Pos())
							results = append(results, &Symbol{
								Name:   ident.Name,
								Kind:   "value",
								File:   filePath,
								Line:   pos.Line,
								Column: pos.Column,
								Offset: pos.Offset,
							})
						}
					}
				}
			}
		}
	}

	return results, nil
}

func extractReceiver(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	t := recv.List[0].Type
	for {
		switch node := t.(type) {
		case *ast.StarExpr:
			t = node.X
		case *ast.IndexExpr:
			t = node.X
		case *ast.IndexListExpr:
			t = node.X
		case *ast.Ident:
			return node.Name
		default:
			return ""
		}
	}
}
