// Package javacst provides syntax-checked Java source transformations and construct selection.
// WHY: This file resolves method-local constructs while leaving semantic checks to JDT LS.
package javacst

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/spockz/semantic-editor/internal/backend"

	gts "github.com/odvcencio/gotreesitter"
	javagrammar "github.com/odvcencio/gotreesitter/grammars/java"
)

// ErrInvalidSource and related errors classify Java source selection failures.
var (
	ErrInvalidSource   = errors.New("invalid Java source")
	ErrUnsupportedKind = errors.New("unsupported Java construct kind")
	ErrNotFound        = errors.New("java construct not found")
	ErrAmbiguous       = errors.New("java construct selector is ambiguous")
	ErrInvalidPath     = errors.New("invalid Java construct path")
)

// ConstructSwitch and ConstructElement add Java-specific construct selectors.
const (
	ConstructSwitch                backend.ConstructKind = "switch"
	ConstructElement               backend.ConstructKind = "element"
	javaNodeIdentifier                                   = "identifier"
	javaNodeClassDeclaration                             = "class_declaration"
	javaNodeMethodDeclaration                            = "method_declaration"
	javaNodeConstructorDeclaration                       = "constructor_declaration"
	javaNodeFieldDeclaration                             = "field_declaration"
	javaNodeMethodInvocation                             = "method_invocation"
	javaNodeVariableDeclarator                           = "variable_declarator"
	javaNodeInterfaceDeclaration                         = "interface_declaration"
	javaNodeEnumDeclaration                              = "enum_declaration"
	javaNodeRecordDeclaration                            = "record_declaration"
)

// Candidate identifies one syntax-checked Java construct within its owner.
type Candidate struct {
	ConstructPath string
	Syntax        string
	Kind          backend.ConstructKind
	StartByte     uint32
	EndByte       uint32
	Range         backend.Range
}

type revisionCandidate struct {
	ConstructPath string `json:"construct_path"`
	Syntax        string `json:"syntax"`
}

type revisionDescriptor struct {
	Version       string                `json:"version"`
	Owner         string                `json:"owner"`
	Kind          backend.ConstructKind `json:"kind"`
	Discriminator string                `json:"discriminator"`
	Candidates    []revisionCandidate   `json:"candidates"`
}

// Candidates returns syntax-checked selectors for constructs matching the discriminator.
func Candidates(method []byte, kind backend.ConstructKind, discriminator string) ([]Candidate, error) {
	if len(bytes.TrimSpace(method)) == 0 {
		return nil, fmt.Errorf("empty method source: %w", ErrInvalidSource)
	}
	if kind == backend.ConstructStatementOrExpr && strings.TrimSpace(discriminator) == "" {
		return nil, fmt.Errorf("statement/expression requires a discriminator: %w", ErrNotFound)
	}
	if !supportedKind(kind) {
		return nil, fmt.Errorf("%q: %w", kind, ErrUnsupportedKind)
	}
	lang, root, err := parse(method)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	result := make([]Candidate, 0)
	var visit func(*gts.Node, string)
	visit = func(node *gts.Node, parentPath string) {
		if node == nil {
			return
		}
		nodeKind, recognized := mapKind(node, lang)
		path := parentPath
		if recognized {
			path = nextJavaCandidatePath(counts, parentPath, nodeKind)
			syntax := strings.TrimSpace(string(method[node.StartByte():node.EndByte()]))
			if nodeKind == kind && matches(method, node, discriminator) {
				result = append(result, newCandidate(method, node, path, syntax, nodeKind))
			}
		}
		result = append(result, specialJavaCandidates(method, lang, node, kind, path, parentPath, discriminator, counts)...)
		for index := range node.NamedChildCount() {
			visit(node.NamedChild(index), path)
		}
	}
	visit(root, "")
	if len(result) == 0 {
		return []Candidate{}, nil
	}
	return result, nil
}

func nextJavaCandidatePath(counts map[string]int, parent string, kind backend.ConstructKind) string {
	key := parent + "|" + string(kind)
	index := counts[key]
	counts[key] = index + 1
	segment := fmt.Sprintf("%s[%d]", kind, index)
	if parent == "" {
		return segment
	}
	return parent + "/" + segment
}

func specialJavaCandidates(source []byte, lang *gts.Language, node *gts.Node, kind backend.ConstructKind, path, parentPath, discriminator string, counts map[string]int) []Candidate {
	if kind == backend.ConstructTryCatch && node.Type(lang) == "try_with_resources_statement" && hasNamedChild(node, "catch_clause", lang) && matches(source, node, discriminator) {
		candidatePath := nextJavaCandidatePath(counts, parentPath, kind)
		return []Candidate{newCandidate(source, node, candidatePath, strings.TrimSpace(string(source[node.StartByte():node.EndByte()])), kind)}
	}
	if kind == backend.ConstructElse {
		return javaElseCandidate(source, lang, node, path, discriminator, counts)
	}
	if kind == ConstructElement {
		return javaElementCandidates(source, lang, node, path, discriminator, counts)
	}
	if kind == backend.ConstructStatementOrExpr && isStatementOrExpression(node.Type(lang)) && matches(source, node, discriminator) {
		candidatePath := nextJavaCandidatePath(counts, path, kind)
		return []Candidate{newCandidate(source, node, candidatePath, strings.TrimSpace(string(source[node.StartByte():node.EndByte()])), kind)}
	}
	return nil
}

func javaElseCandidate(source []byte, lang *gts.Language, node *gts.Node, path, discriminator string, counts map[string]int) []Candidate {
	if node.Type(lang) != "if_statement" {
		return nil
	}
	branches := make([]*gts.Node, 0, 2)
	for index := range node.NamedChildCount() {
		if index > 0 {
			branches = append(branches, node.NamedChild(index))
		}
	}
	if len(branches) <= 1 {
		return nil
	}
	alternative := branches[1]
	start := elseKeywordOffset(source, branches[0].EndByte(), alternative.StartByte())
	if start < 0 || !matches(source, node, discriminator) {
		return nil
	}
	candidatePath := nextJavaCandidatePath(counts, path, backend.ConstructElse)
	candidate := Candidate{ConstructPath: candidatePath, Syntax: strings.TrimSpace(string(source[start:alternative.EndByte()])), Kind: backend.ConstructElse, StartByte: uint32(start), EndByte: alternative.EndByte()} // #nosec G115 -- start is bounded by parser-provided uint32 offsets.
	candidate.Range = backend.Range{Start: backend.PositionFromByteOffset(source, start), End: backend.PositionFromByteOffset(source, int(alternative.EndByte()))}
	return []Candidate{candidate}
}

func javaElementCandidates(source []byte, lang *gts.Language, node *gts.Node, path, discriminator string, counts map[string]int) []Candidate {
	if node.Type(lang) == "array_initializer" || node.Type(lang) == "element_value_array_initializer" {
		candidates := make([]Candidate, 0, node.NamedChildCount())
		for index := range node.NamedChildCount() {
			child := node.NamedChild(index)
			candidatePath := nextJavaCandidatePath(counts, path, ConstructElement)
			if matches(source, child, discriminator) {
				candidates = append(candidates, newCandidate(source, child, candidatePath, strings.TrimSpace(string(source[child.StartByte():child.EndByte()])), ConstructElement))
			}
		}
		return candidates
	}
	if node.Type(lang) == javaNodeMethodInvocation && javaCollectionFactory(lang, source, node) != "" && matches(source, node, discriminator) {
		candidatePath := nextJavaCandidatePath(counts, path, ConstructElement)
		return []Candidate{newCandidate(source, node, candidatePath, strings.TrimSpace(string(source[node.StartByte():node.EndByte()])), ConstructElement)}
	}
	return nil
}

// ValidateSource reports whether source parses without Java syntax errors.
func ValidateSource(source []byte) error {
	_, _, err := parse(source)
	return err
}

// ValidateMethodConstructSnippet checks that a snippet is one method-level construct.
func ValidateMethodConstructSnippet(kind backend.ConstructKind, snippet []byte) error {
	trimmed := bytes.TrimSpace(snippet)
	if len(trimmed) == 0 {
		return fmt.Errorf("java construct source is empty: %w", ErrInvalidSource)
	}
	switch kind {
	case backend.ConstructLoop, backend.ConstructIf, ConstructSwitch, backend.ConstructTryWithResources,
		backend.ConstructSynchronized, backend.ConstructStatementOrExpr:
	default:
		return fmt.Errorf("%q: %w", kind, ErrUnsupportedKind)
	}
	prefix := []byte("class __Semedit { void __method() {\n")
	wrapped := make([]byte, 0, len(prefix)+len(trimmed)+5)
	wrapped = append(wrapped, prefix...)
	wrapped = append(wrapped, trimmed...)
	wrapped = append(wrapped, []byte("\n} }\n")...)
	if err := ValidateSource(wrapped); err != nil {
		return fmt.Errorf("java construct snippet is invalid: %w", err)
	}
	discriminator := ""
	if kind == backend.ConstructStatementOrExpr {
		discriminator = string(trimmed)
	}
	candidates, err := Candidates(wrapped, kind, discriminator)
	if err != nil {
		return err
	}
	want := compact(string(trimmed))
	matches := 0
	for _, candidate := range candidates {
		if compact(candidate.Syntax) == want {
			matches++
		}
	}
	if matches != 1 {
		return fmt.Errorf("java source must contain exactly one %s construct", kind)
	}
	return nil
}

// ValidateDeclarationReplacement ensures a replacement preserves declaration kind and name.
func ValidateDeclarationReplacement(existing, replacement []byte) error {
	existingType, existingNames, err := declarationIdentity(existing)
	if err != nil {
		return fmt.Errorf("inspect existing Java declaration: %w", err)
	}
	replacementType, replacementNames, err := declarationIdentity(replacement)
	if err != nil {
		return fmt.Errorf("inspect replacement Java declaration: %w", err)
	}
	if existingType != replacementType || strings.Join(existingNames, "\x00") != strings.Join(replacementNames, "\x00") {
		return fmt.Errorf("java declaration replacement must preserve declaration kind and name")
	}
	return nil
}

// ValidateDeclarationKind checks that Java source matches the requested declaration kind.
func ValidateDeclarationKind(source []byte, kind backend.ConstructKind) error {
	declarationType, _, err := declarationIdentity(source)
	if err != nil {
		return err
	}
	var valid bool
	switch kind {
	case backend.ConstructMethod:
		valid = declarationType == javaNodeMethodDeclaration
	case backend.ConstructConstructor:
		valid = declarationType == javaNodeConstructorDeclaration
	case backend.ConstructField:
		valid = declarationType == javaNodeFieldDeclaration
	case backend.ConstructType:
		valid = declarationType == javaNodeClassDeclaration || declarationType == javaNodeInterfaceDeclaration || declarationType == javaNodeEnumDeclaration || declarationType == javaNodeRecordDeclaration
	case backend.ConstructClass:
		valid = declarationType == javaNodeClassDeclaration
	case backend.ConstructInterface:
		valid = declarationType == javaNodeInterfaceDeclaration
	case backend.ConstructEnum:
		valid = declarationType == javaNodeEnumDeclaration
	default:
		valid = false
	}
	if !valid {
		return fmt.Errorf("java declaration kind %q does not match the selected %s", kind, declarationType)
	}
	return nil
}

// IsPrivateJavaMethod reports whether a method declaration is private.
func IsPrivateJavaMethod(source []byte) (bool, error) {
	return IsPrivateJavaDeclaration(source, backend.ConstructMethod)
}

// RemoveCandidate removes one selected method-local construct from its source.
func RemoveCandidate(method []byte, candidate Candidate) ([]byte, error) {
	start, end := int(candidate.StartByte), int(candidate.EndByte)
	if start < 0 || end <= start || end > len(method) {
		return nil, fmt.Errorf("invalid Java construct range: %w", ErrInvalidPath)
	}
	lineStart := bytes.LastIndexByte(method[:start], '\n') + 1
	if len(bytes.TrimSpace(method[lineStart:start])) == 0 {
		start = lineStart
		if end < len(method) && method[end] == '\r' {
			end++
		}
		if end < len(method) && method[end] == '\n' {
			end++
		}
	} else {
		for end < len(method) && (method[end] == ' ' || method[end] == '\t') {
			end++
		}
	}
	updated := make([]byte, 0, len(method)-(end-start))
	updated = append(updated, method[:start]...)
	updated = append(updated, method[end:]...)
	if err := ValidateSource(updated); err != nil {
		return nil, fmt.Errorf("removal produces invalid Java syntax: %w", err)
	}
	return updated, nil
}

// RemoveJavaDeclaration removes a declaration only when its selected bytes still match.
func RemoveJavaDeclaration(source []byte, start, end int, expected []byte) ([]byte, error) {
	if start < 0 || end <= start || end > len(source) || !bytes.Equal(source[start:end], expected) {
		return nil, fmt.Errorf("java declaration range does not match its source preimage: %w", ErrInvalidPath)
	}
	lang, root, err := parse(source)
	if err != nil {
		return nil, err
	}
	type nodeWithParent struct {
		node   *gts.Node
		parent *gts.Node
		index  int
	}
	var selected nodeWithParent
	var find func(*gts.Node, *gts.Node, int)
	find = func(node, parent *gts.Node, index int) {
		if node == nil || selected.node != nil {
			return
		}
		if int(node.StartByte()) == start && int(node.EndByte()) == end && isJavaMemberDeclaration(node.Type(lang)) {
			selected = nodeWithParent{node: node, parent: parent, index: index}
			return
		}
		for childIndex := range node.NamedChildCount() {
			find(node.NamedChild(childIndex), node, childIndex)
		}
	}
	find(root, nil, 0)
	if selected.node == nil {
		return nil, fmt.Errorf("java declaration range does not identify one member: %w", ErrInvalidPath)
	}
	removeStart := start
	currentStart := start
	for index := selected.index - 1; index >= 0; index-- {
		previous := selected.parent.NamedChild(index)
		if previous.Type(lang) != "block_comment" || !bytes.HasPrefix(source[previous.StartByte():previous.EndByte()], []byte("/**")) {
			break
		}
		gap := source[previous.EndByte():currentStart]
		if !allJavaWhitespace(gap) || bytes.Count(gap, []byte("\n")) > 1 {
			break
		}
		removeStart = int(previous.StartByte())
		currentStart = int(previous.StartByte())
	}
	lineStart := bytes.LastIndexByte(source[:removeStart], '\n') + 1
	if len(bytes.TrimSpace(source[lineStart:removeStart])) == 0 {
		removeStart = lineStart
		if end < len(source) && source[end] == '\r' {
			end++
		}
		if end < len(source) && source[end] == '\n' {
			end++
		}
	}
	updated := spliceSource(source, removeStart, end, nil)
	if err := ValidateSource(updated); err != nil {
		return nil, fmt.Errorf("declaration removal produces invalid Java syntax: %w", err)
	}
	return updated, nil
}

func isJavaMemberDeclaration(typeName string) bool {
	switch typeName {
	case javaNodeMethodDeclaration, javaNodeConstructorDeclaration, javaNodeFieldDeclaration, javaNodeClassDeclaration, javaNodeInterfaceDeclaration, javaNodeEnumDeclaration, javaNodeRecordDeclaration:
		return true
	default:
		return false
	}
}

func allJavaWhitespace(source []byte) bool {
	for _, value := range source {
		if value != ' ' && value != '\t' && value != '\r' && value != '\n' {
			return false
		}
	}
	return true
}

func spliceSource(source []byte, start, end int, replacement []byte) []byte {
	updated := make([]byte, 0, len(source)-(end-start)+len(replacement))
	updated = append(updated, source[:start]...)
	updated = append(updated, replacement...)
	updated = append(updated, source[end:]...)
	return updated
}

func soleWrappedMember(lang *gts.Language, root *gts.Node) (*gts.Node, error) {
	for index := range root.NamedChildCount() {
		class := root.NamedChild(index)
		if class.Type(lang) != javaNodeClassDeclaration {
			continue
		}
		for childIndex := range class.NamedChildCount() {
			body := class.NamedChild(childIndex)
			if body.Type(lang) != javaNodeClassBody {
				continue
			}
			var members []*gts.Node
			for memberIndex := range body.NamedChildCount() {
				member := body.NamedChild(memberIndex)
				if member.Type(lang) != "line_comment" && member.Type(lang) != "block_comment" {
					members = append(members, member)
				}
			}
			if len(members) == 1 {
				return members[0], nil
			}
			return nil, fmt.Errorf("source must contain exactly one Java member declaration: %w", ErrInvalidSource)
		}
	}
	return nil, fmt.Errorf("wrapped source has no class body: %w", ErrInvalidSource)
}

func declarationIdentity(source []byte) (string, []string, error) {
	trimmed := bytes.TrimSpace(source)
	if len(trimmed) == 0 {
		return "", nil, fmt.Errorf("empty declaration: %w", ErrInvalidSource)
	}
	prefix := []byte("class __Semedit {\n")
	wrapped := make([]byte, 0, len(prefix)+len(trimmed)+4)
	wrapped = append(wrapped, prefix...)
	wrapped = append(wrapped, trimmed...)
	wrapped = append(wrapped, []byte("\n}\n")...)
	lang, root, err := parse(wrapped)
	if err != nil {
		return "", nil, err
	}
	var classBody *gts.Node
	for index := range root.NamedChildCount() {
		child := root.NamedChild(index)
		if child.Type(lang) != javaNodeClassDeclaration {
			continue
		}
		for childIndex := range child.NamedChildCount() {
			if candidate := child.NamedChild(childIndex); candidate.Type(lang) == javaNodeClassBody {
				classBody = candidate
				break
			}
		}
	}
	if classBody == nil {
		return "", nil, fmt.Errorf("wrapped source has no class body: %w", ErrInvalidSource)
	}
	declaration, err := soleWrappedMember(lang, root)
	if err != nil {
		return "", nil, err
	}
	names := make([]string, 0, 1)
	if declaration.Type(lang) == javaNodeFieldDeclaration {
		for index := range declaration.NamedChildCount() {
			child := declaration.NamedChild(index)
			if child.Type(lang) != javaNodeVariableDeclarator {
				continue
			}
			if child.NamedChildCount() == 0 || child.NamedChild(0).Type(lang) != javaNodeIdentifier {
				return "", nil, fmt.Errorf("field declaration lacks an identifier: %w", ErrInvalidSource)
			}
			names = append(names, string(wrapped[child.NamedChild(0).StartByte():child.NamedChild(0).EndByte()]))
		}
	} else {
		for index := range declaration.NamedChildCount() {
			child := declaration.NamedChild(index)
			if child.Type(lang) == javaNodeIdentifier {
				names = append(names, string(wrapped[child.StartByte():child.EndByte()]))
				break
			}
		}
	}
	if len(names) == 0 {
		return "", nil, fmt.Errorf("java member declaration has no stable name: %w", ErrInvalidSource)
	}
	return declaration.Type(lang), names, nil
}

// ReplaceCandidate replaces one selected method-local construct and validates the result.
func ReplaceCandidate(method []byte, candidate Candidate, replacement []byte) ([]byte, error) {
	start, end := int(candidate.StartByte), int(candidate.EndByte)
	if start < 0 || end < start || end > len(method) {
		return nil, fmt.Errorf("candidate range [%d,%d) is outside method source: %w", start, end, ErrInvalidPath)
	}
	updated := make([]byte, 0, len(method)-(end-start)+len(replacement))
	updated = append(updated, method[:start]...)
	updated = append(updated, replacement...)
	updated = append(updated, method[end:]...)
	if err := ValidateSource(updated); err != nil {
		return nil, fmt.Errorf("replacement produces invalid Java syntax: %w", err)
	}
	return updated, nil
}

func parse(source []byte) (*gts.Language, *gts.Node, error) {
	lang := javagrammar.Language()
	tree, err := gts.NewParser(lang).Parse(source)
	if err != nil {
		return lang, nil, fmt.Errorf("parse Java source: %w", err)
	}
	root := tree.RootNode()
	if root == nil || root.HasErrorOrMissing() {
		return lang, root, fmt.Errorf("tree-sitter reported syntax errors: %w", ErrInvalidSource)
	}
	return lang, root, nil
}

func elseKeywordOffset(source []byte, start, end uint32) int {
	if end < start || int(end) > len(source) {
		return -1
	}
	between := source[start:end]
	index := bytes.Index(between, []byte("else"))
	if index < 0 {
		return -1
	}
	before := int(start) + index
	after := before + 4
	if before > 0 && isJavaIdentifierByte(source[before-1]) {
		return -1
	}
	if after < len(source) && isJavaIdentifierByte(source[after]) {
		return -1
	}
	return before
}

func isJavaIdentifierByte(value byte) bool {
	return value == '_' || value == '$' || value >= '0' && value <= '9' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func newCandidate(source []byte, node *gts.Node, path, syntax string, kind backend.ConstructKind) Candidate {
	start, end := int(node.StartByte()), int(node.EndByte())
	return Candidate{
		ConstructPath: path,
		Syntax:        syntax,
		Kind:          kind,
		StartByte:     node.StartByte(),
		EndByte:       node.EndByte(),
		Range:         backend.Range{Start: backend.PositionFromByteOffset(source, start), End: backend.PositionFromByteOffset(source, end)},
	}
}

func isStatementOrExpression(typeName string) bool {
	return strings.HasSuffix(typeName, "_statement") || isExpressionNode(typeName)
}

// Resolve selects one construct by kind, exact syntax, and optional inspected path.
func Resolve(method []byte, kind backend.ConstructKind, discriminator, constructPath string) (Candidate, []Candidate, error) {
	candidates, err := Candidates(method, kind, discriminator)
	if err != nil {
		return Candidate{}, nil, err
	}
	if constructPath != "" {
		for _, candidate := range candidates {
			if candidate.ConstructPath == constructPath {
				return candidate, candidates, nil
			}
		}
		return Candidate{}, candidates, fmt.Errorf("%q: %w", constructPath, ErrInvalidPath)
	}
	if len(candidates) == 0 {
		return Candidate{}, candidates, ErrNotFound
	}
	if len(candidates) > 1 {
		return Candidate{}, candidates, ErrAmbiguous
	}
	return candidates[0], candidates, nil
}

// ResolveExpression selects an expression by exact syntax and optional inspected path.
func ResolveExpression(method []byte, discriminator, constructPath string) (Candidate, []Candidate, error) {
	if strings.TrimSpace(discriminator) == "" {
		return Candidate{}, nil, fmt.Errorf("java expression requires a discriminator: %w", ErrNotFound)
	}
	lang, root, err := parse(method)
	if err != nil {
		return Candidate{}, nil, err
	}
	counts := make(map[string]int)
	exact := make([]Candidate, 0)
	var visit func(*gts.Node, string)
	visit = func(node *gts.Node, parentPath string) {
		if node == nil {
			return
		}
		path := parentPath
		if nodeKind, recognized := mapKind(node, lang); recognized {
			key := parentPath + "|" + string(nodeKind)
			index := counts[key]
			counts[key] = index + 1
			segment := fmt.Sprintf("%s[%d]", nodeKind, index)
			if path == "" {
				path = segment
			} else {
				path += "/" + segment
			}
		}
		if isExpressionNode(node.Type(lang)) {
			key := path + "|" + string(backend.ConstructStatementOrExpr)
			index := counts[key]
			counts[key] = index + 1
			segment := fmt.Sprintf("%s[%d]", backend.ConstructStatementOrExpr, index)
			candidatePath := segment
			if path != "" {
				candidatePath = path + "/" + segment
			}
			syntax := strings.TrimSpace(string(method[node.StartByte():node.EndByte()]))
			if compact(syntax) == compact(discriminator) {
				exact = append(exact, newCandidate(method, node, candidatePath, syntax, backend.ConstructStatementOrExpr))
			}
		}
		for index := range node.NamedChildCount() {
			visit(node.NamedChild(index), path)
		}
	}
	visit(root, "")
	if constructPath != "" {
		for _, candidate := range exact {
			if candidate.ConstructPath == constructPath {
				return candidate, exact, nil
			}
		}
		return Candidate{}, exact, fmt.Errorf("%q: %w", constructPath, ErrInvalidPath)
	}
	if len(exact) == 0 {
		return Candidate{}, exact, ErrNotFound
	}
	if len(exact) > 1 {
		return Candidate{}, exact, ErrAmbiguous
	}
	return exact[0], exact, nil
}

func isExpressionNode(typeName string) bool {
	return strings.HasSuffix(typeName, "_expression") || strings.HasSuffix(typeName, "_literal") || typeName == javaNodeIdentifier ||
		typeName == javaNodeMethodInvocation || typeName == "method_reference" || typeName == "field_access" || typeName == "array_access" ||
		typeName == "this" || typeName == "super" || typeName == "true" || typeName == "false"
}

// RenameJavaIdentifiersInRange renames matching identifier nodes within a checked byte range.
func RenameJavaIdentifiersInRange(source []byte, start, end int, oldName, newName string) ([]byte, error) {
	if start < 0 || end <= start || end > len(source) || oldName == "" || newName == "" || oldName == newName {
		return nil, fmt.Errorf("invalid Java identifier rename range: %w", ErrInvalidPath)
	}
	language, root, err := parse(source)
	if err != nil {
		return nil, err
	}
	positions := make([][2]int, 0, 2)
	var visit func(*gts.Node)
	visit = func(node *gts.Node) {
		if node == nil {
			return
		}
		nodeStart, nodeEnd := int(node.StartByte()), int(node.EndByte())
		if (node.Type(language) == javaNodeIdentifier || node.Type(language) == javaNodeTypeIdentifier) && nodeStart >= start && nodeEnd <= end && string(source[nodeStart:nodeEnd]) == oldName {
			positions = append(positions, [2]int{nodeStart, nodeEnd})
		}
		for index := range node.NamedChildCount() {
			visit(node.NamedChild(index))
		}
	}
	visit(root)
	if len(positions) == 0 {
		return nil, fmt.Errorf("java rename range contains no identifier %q: %w", oldName, ErrNotFound)
	}
	updated := make([]byte, 0, end-start+len(positions)*(len(newName)-len(oldName)))
	cursor := start
	for _, position := range positions {
		updated = append(updated, source[cursor:position[0]]...)
		updated = append(updated, newName...)
		cursor = position[1]
	}
	updated = append(updated, source[cursor:end]...)
	return updated, nil
}

// SelectionRevision fingerprints the owner and candidate set used for a selection.
func SelectionRevision(owner string, kind backend.ConstructKind, discriminator string, candidates []Candidate) (string, error) {
	revisionCandidates := make([]revisionCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		revisionCandidates = append(revisionCandidates, revisionCandidate{ConstructPath: candidate.ConstructPath, Syntax: candidate.Syntax})
	}
	descriptor := revisionDescriptor{Version: "semedit.construct-selection/v1", Owner: owner, Kind: kind, Discriminator: compact(discriminator), Candidates: revisionCandidates}
	encoded, err := json.Marshal(descriptor)
	if err != nil {
		return "", fmt.Errorf("encode Java construct selection revision: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

// ByteOffsetFromPosition converts an LSP position to a UTF-8 byte offset.
func ByteOffsetFromPosition(source []byte, position backend.Position) (int, error) {
	if position.Line < 0 || position.Character < 0 {
		return 0, fmt.Errorf("negative Java source position")
	}
	lineStart := 0
	for range position.Line {
		relative := bytes.IndexByte(source[lineStart:], '\n')
		if relative < 0 {
			return 0, fmt.Errorf("java source line %d is outside the document", position.Line)
		}
		lineStart += relative + 1
	}
	lineEnd := len(source)
	if relative := bytes.IndexByte(source[lineStart:], '\n'); relative >= 0 {
		lineEnd = lineStart + relative
	}
	for offset, units := lineStart, 0; ; {
		if units == position.Character {
			return offset, nil
		}
		if offset >= lineEnd {
			return 0, fmt.Errorf("UTF-16 character %d is outside Java source line %d", position.Character, position.Line)
		}
		runeValue, width := utf8.DecodeRune(source[offset:lineEnd])
		if runeValue == utf8.RuneError && width == 1 {
			return 0, fmt.Errorf("invalid UTF-8 in Java source line %d", position.Line)
		}
		unitCount := 1
		if runeValue > 0xffff {
			unitCount = 2
		}
		if units+unitCount > position.Character {
			return 0, fmt.Errorf("UTF-16 character %d splits a surrogate pair", position.Character)
		}
		offset += width
		units += unitCount
	}
}

func supportedKind(kind backend.ConstructKind) bool {
	switch kind {
	case backend.ConstructLoop, backend.ConstructIf, backend.ConstructElse, backend.ConstructCase,
		backend.ConstructStatementOrExpr, backend.ConstructTryCatch, backend.ConstructTryWithResources,
		backend.ConstructSynchronized, ConstructSwitch, ConstructElement:
		return true
	default:
		return false
	}
}

func mapKind(node *gts.Node, lang *gts.Language) (backend.ConstructKind, bool) {
	typeName := node.Type(lang)
	var mapped backend.ConstructKind
	switch typeName {
	case "for_statement", "enhanced_for_statement", "while_statement", "do_statement":
		mapped = backend.ConstructLoop
	case "if_statement":
		mapped = backend.ConstructIf
	case "switch_statement", "switch_expression":
		mapped = ConstructSwitch
	case "switch_block_statement_group", "switch_rule":
		mapped = backend.ConstructCase
	case "try_with_resources_statement":
		mapped = backend.ConstructTryWithResources
	case "try_statement":
		if hasNamedChild(node, "catch_clause", lang) || hasNamedChild(node, "finally_clause", lang) {
			mapped = backend.ConstructTryCatch
		}
	case "synchronized_statement":
		mapped = backend.ConstructSynchronized
	}
	return mapped, mapped != ""
}

func hasNamedChild(node *gts.Node, typeName string, lang *gts.Language) bool {
	for i := range node.NamedChildCount() {
		if node.NamedChild(i).Type(lang) == typeName {
			return true
		}
	}
	return false
}

func matches(source []byte, node *gts.Node, discriminator string) bool {
	if strings.TrimSpace(discriminator) == "" {
		return true
	}
	selected := compact(string(source[node.StartByte():node.EndByte()]))
	needle := compact(discriminator)
	return strings.Contains(selected, needle)
}

func compact(value string) string {
	fields := strings.Fields(value)
	return strings.Join(fields, "")
}

// InsertIntoMethod inserts one validated snippet at a selected method-body anchor.
func InsertIntoMethod(method []byte, anchor, discriminator, constructPath string, snippet []byte) ([]byte, error) {
	if strings.TrimSpace(anchor) == "" {
		return nil, fmt.Errorf("java method insertion anchor is required")
	}
	if len(bytes.TrimSpace(snippet)) == 0 {
		return nil, fmt.Errorf("java method insertion source is empty")
	}
	lang, root, err := parse(method)
	if err != nil {
		return nil, err
	}
	bodyNode, err := soleJavaMethodBody(lang, root)
	if err != nil {
		return nil, err
	}
	bodyStart, bodyEnd := int(bodyNode.StartByte())+1, int(bodyNode.EndByte())-1
	firstContent := bodyStart
	for firstContent < bodyEnd && (method[firstContent] == ' ' || method[firstContent] == '\t' || method[firstContent] == '\r' || method[firstContent] == '\n') {
		firstContent++
	}
	indentAt := javaInsertionIndent(method)
	indent := indentAt(firstContent)
	offset, prefix, suffix, err := javaMethodInsertionPosition(method, bodyStart, bodyEnd, firstContent, anchor, discriminator, constructPath, indent)
	if err != nil {
		return nil, err
	}
	if offset < bodyStart || offset > bodyEnd {
		return nil, fmt.Errorf("java method insertion anchor is outside the method body: %w", ErrInvalidPath)
	}
	insertion := prefix
	insertion = append(insertion, bytes.TrimSpace(snippet)...)
	insertion = append(insertion, suffix...)
	updated := make([]byte, 0, len(method)+len(insertion))
	updated = append(updated, method[:offset]...)
	updated = append(updated, insertion...)
	updated = append(updated, method[offset:]...)
	if err := ValidateSource(updated); err != nil {
		return nil, fmt.Errorf("insertion produces invalid Java syntax: %w", err)
	}
	return updated, nil
}

func javaInsertionIndent(source []byte) func(int) []byte {
	return func(offset int) []byte {
		lineStart := bytes.LastIndexByte(source[:offset], '\n') + 1
		lineEnd := bytes.IndexByte(source[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(source) - lineStart
		}
		line := source[lineStart : lineStart+lineEnd]
		indentEnd := 0
		for indentEnd < len(line) && (line[indentEnd] == ' ' || line[indentEnd] == '\t') {
			indentEnd++
		}
		if indentEnd == 0 && len(bytes.TrimSpace(line)) > 0 {
			return []byte("\t")
		}
		return bytes.Clone(line[:indentEnd])
	}
}

func javaMethodInsertionPosition(source []byte, bodyStart, bodyEnd, firstContent int, anchor, discriminator, constructPath string, indent []byte) (int, []byte, []byte, error) {
	indentAt := javaInsertionIndent(source)
	suffix := append([]byte{'\n'}, indent...)
	switch anchor {
	case "start":
		if firstContent == bodyEnd {
			return bodyStart, []byte("\n"), suffix, nil
		}
		prefix := []byte(nil)
		if !bytes.Contains(source[bodyStart:firstContent], []byte("\n")) {
			prefix = []byte("\n\t")
		}
		return firstContent, prefix, suffix, nil
	case "end":
		offset := bodyEnd
		for offset > bodyStart && isJavaWhitespace(source[offset-1]) {
			offset--
		}
		return offset, append([]byte{'\n'}, indent...), nil, nil
	case "before", "after":
		candidate, _, err := Resolve(source, backend.ConstructStatementOrExpr, discriminator, constructPath)
		if err != nil {
			return 0, nil, nil, err
		}
		if anchor == "after" {
			offset := int(candidate.EndByte)
			return offset, append([]byte{'\n'}, indentAt(offset)...), nil, nil
		}
		offset := int(candidate.StartByte)
		return offset, nil, append([]byte{'\n'}, indentAt(offset)...), nil
	case "before_return":
		candidates, err := Candidates(source, backend.ConstructStatementOrExpr, "return")
		if err != nil {
			return 0, nil, nil, err
		}
		offset := -1
		for _, candidate := range candidates {
			if strings.HasPrefix(strings.TrimSpace(candidate.Syntax), "return") && int(candidate.StartByte) > offset {
				offset = int(candidate.StartByte)
			}
		}
		if offset < 0 {
			return 0, nil, nil, fmt.Errorf("java method has no return statement")
		}
		return offset, nil, append([]byte{'\n'}, indentAt(offset)...), nil
	default:
		return 0, nil, nil, fmt.Errorf("unsupported Java method insertion anchor %q", anchor)
	}
}

// InsertJavaSwitchCase inserts one validated clause into a selected switch body.
func InsertJavaSwitchCase(method []byte, discriminator, constructPath string, clause []byte, placement, targetCase string) ([]byte, error) {
	if len(bytes.TrimSpace(clause)) == 0 {
		return nil, fmt.Errorf("java switch case source is empty")
	}
	if placement == "" {
		placement = "before_default"
	}
	switchCandidate, _, err := Resolve(method, ConstructSwitch, discriminator, constructPath)
	if err != nil {
		return nil, err
	}
	lang, root, err := parse(method)
	if err != nil {
		return nil, err
	}
	switchNode := findSelectedJavaSwitch(lang, root, switchCandidate)
	if switchNode == nil {
		return nil, fmt.Errorf("selected Java switch no longer exists: %w", ErrInvalidPath)
	}
	block, groups, err := selectedJavaSwitchGroups(lang, switchNode)
	if err != nil {
		return nil, err
	}
	newLabelKey, err := parseJavaSwitchCase(clause)
	if err != nil {
		return nil, err
	}
	if newLabelKey == "" {
		return nil, fmt.Errorf("java case label is empty")
	}
	defaultIndex, targetIndex, err := javaSwitchCaseIndexes(method, lang, groups, newLabelKey, targetCase)
	if err != nil {
		return nil, err
	}
	position, prefix, suffix, err := javaSwitchCaseInsertion(method, switchNode, block, groups, placement, targetIndex, defaultIndex)
	if err != nil {
		return nil, err
	}
	if position < int(block.StartByte())+1 || position > int(block.EndByte())-1 {
		return nil, fmt.Errorf("java case insertion position is outside the switch body: %w", ErrInvalidPath)
	}
	inserted := prefix
	inserted = append(inserted, bytes.TrimSpace(clause)...)
	inserted = append(inserted, suffix...)
	updated := make([]byte, 0, len(method)+len(inserted))
	updated = append(updated, method[:position]...)
	updated = append(updated, inserted...)
	updated = append(updated, method[position:]...)
	if err := ValidateSource(updated); err != nil {
		return nil, fmt.Errorf("java case insertion produces invalid syntax: %w", err)
	}
	return updated, nil
}

func findSelectedJavaSwitch(lang *gts.Language, root *gts.Node, candidate Candidate) *gts.Node {
	var selected *gts.Node
	var visit func(*gts.Node)
	visit = func(node *gts.Node) {
		if node == nil || selected != nil {
			return
		}
		kind := node.Type(lang)
		if (kind == "switch_statement" || kind == "switch_expression") && node.StartByte() == candidate.StartByte && node.EndByte() == candidate.EndByte {
			selected = node
			return
		}
		for index := range node.NamedChildCount() {
			visit(node.NamedChild(index))
		}
	}
	visit(root)
	return selected
}

func selectedJavaSwitchGroups(lang *gts.Language, switchNode *gts.Node) (*gts.Node, []*gts.Node, error) {
	var block *gts.Node
	for index := range switchNode.NamedChildCount() {
		child := switchNode.NamedChild(index)
		if child.Type(lang) == "switch_block" {
			block = child
			break
		}
	}
	if block == nil || block.EndByte() <= block.StartByte()+1 {
		return nil, nil, fmt.Errorf("selected Java switch has no editable body")
	}
	groups := make([]*gts.Node, 0)
	for index := range block.NamedChildCount() {
		child := block.NamedChild(index)
		if child.Type(lang) == "switch_block_statement_group" || child.Type(lang) == "switch_rule" {
			groups = append(groups, child)
		}
	}
	return block, groups, nil
}

func javaSwitchCaseIndexes(source []byte, lang *gts.Language, groups []*gts.Node, newLabelKey, targetCase string) (int, int, error) {
	defaultIndex, targetIndex := -1, -1
	targetLabel := ""
	if targetCase != "" {
		targetLabel, _ = parseJavaSwitchCase([]byte("case " + targetCase + ": break;"))
	}
	for index, group := range groups {
		label := javaSwitchLabel(group, lang)
		if label == nil {
			continue
		}
		labelText := compact(string(source[int(label.StartByte()):int(label.EndByte())]))
		if strings.Contains(labelText, "default") && defaultIndex < 0 {
			defaultIndex = index
		}
		if labelText == newLabelKey {
			return -1, -1, fmt.Errorf("java switch already contains case label %q", labelText)
		}
		if targetLabel != "" && labelText == targetLabel {
			targetIndex = index
		}
	}
	return defaultIndex, targetIndex, nil
}

func javaSwitchCaseInsertion(source []byte, switchNode, block *gts.Node, groups []*gts.Node, placement string, targetIndex, defaultIndex int) (int, []byte, []byte, error) {
	indentAt := func(offset int) []byte {
		lineStart := bytes.LastIndexByte(source[:offset], '\n') + 1
		lineEnd := bytes.IndexByte(source[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(source) - lineStart
		}
		line := source[lineStart : lineStart+lineEnd]
		indentEnd := 0
		for indentEnd < len(line) && (line[indentEnd] == ' ' || line[indentEnd] == '\t') {
			indentEnd++
		}
		return bytes.Clone(source[lineStart : lineStart+indentEnd])
	}
	if len(groups) == 0 && (placement == "first" || placement == "last" || placement == "before_default" && defaultIndex < 0) {
		return emptyJavaSwitchInsertion(source, switchNode, block)
	}
	switch placement {
	case "first":
		position := int(groups[0].StartByte())
		return position, nil, append([]byte{'\n'}, indentAt(position)...), nil
	case "last":
		position := int(groups[len(groups)-1].EndByte())
		return position, append([]byte{'\n'}, indentAt(position)...), nil, nil
	case "before_default":
		if defaultIndex >= 0 {
			position := int(groups[defaultIndex].StartByte())
			return position, nil, append([]byte{'\n'}, indentAt(position)...), nil
		}
		position := int(groups[len(groups)-1].EndByte())
		return position, append([]byte{'\n'}, indentAt(position)...), nil, nil
	case "before", "after":
		if targetIndex < 0 {
			return 0, nil, nil, fmt.Errorf("java case placement %q requires one exact existing target-case", placement)
		}
		position := int(groups[targetIndex].StartByte())
		indent := indentAt(position)
		if placement == "after" {
			position = int(groups[targetIndex].EndByte())
			return position, append([]byte{'\n'}, indent...), nil, nil
		}
		return position, nil, append([]byte{'\n'}, indent...), nil
	default:
		return 0, nil, nil, fmt.Errorf("unsupported Java case placement %q", placement)
	}
}

func emptyJavaSwitchInsertion(source []byte, switchNode, block *gts.Node) (int, []byte, []byte, error) {
	position := int(block.StartByte()) + 1
	switchIndent := lineIndent(source, int(switchNode.StartByte()))
	return position, []byte("\n"), append([]byte{'\n'}, []byte(switchIndent)...), nil
}

func parseJavaSwitchCase(clause []byte) (string, error) {
	wrapper := append([]byte("class __Case { void __insert() { switch (0) {\n"), clause...)
	wrapper = append(wrapper, []byte("\n} } }")...)
	lang, root, err := parse(wrapper)
	if err != nil {
		return "", fmt.Errorf("parse Java case clause: %w", err)
	}
	var block *gts.Node
	var findBlock func(*gts.Node)
	findBlock = func(node *gts.Node) {
		if node == nil || block != nil {
			return
		}
		if node.Type(lang) == "switch_block" {
			block = node
			return
		}
		for index := range node.NamedChildCount() {
			findBlock(node.NamedChild(index))
		}
	}
	findBlock(root)
	if block == nil {
		return "", fmt.Errorf("java case clause must be one complete switch group")
	}
	var groups []*gts.Node
	for index := range block.NamedChildCount() {
		child := block.NamedChild(index)
		if child.Type(lang) == "switch_block_statement_group" || child.Type(lang) == "switch_rule" {
			groups = append(groups, child)
		}
	}
	if len(groups) != 1 || javaSwitchLabel(groups[0], lang) == nil {
		return "", fmt.Errorf("java case source must contain exactly one complete case clause")
	}
	label := javaSwitchLabel(groups[0], lang)
	return compact(string(wrapper[int(label.StartByte()):int(label.EndByte())])), nil
}

func javaSwitchLabel(node *gts.Node, lang *gts.Language) *gts.Node {
	if node == nil {
		return nil
	}
	for index := range node.NamedChildCount() {
		child := node.NamedChild(index)
		if child.Type(lang) == "switch_label" {
			return child
		}
	}
	return nil
}

// ReplaceMethodBody replaces the body of one Java method while preserving its signature.
func ReplaceMethodBody(method, replacement []byte) ([]byte, error) {
	if err := ValidateSource(method); err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(replacement)) == 0 {
		replacement = nil
	}
	lang, root, err := parse(method)
	if err != nil {
		return nil, err
	}
	body, err := soleJavaMethodBody(lang, root)
	if err != nil {
		return nil, err
	}
	start, end := int(body.StartByte())+1, int(body.EndByte())-1
	updated := make([]byte, 0, len(method)-(end-start)+len(replacement)+2)
	updated = append(updated, method[:start]...)
	if len(replacement) > 0 {
		indent := []byte("\t")
		if body.NamedChildCount() > 0 {
			child := body.NamedChild(0)
			lineStart := bytes.LastIndexByte(method[:int(child.StartByte())], '\n') + 1
			candidateIndent := method[lineStart:int(child.StartByte())]
			if strings.TrimSpace(string(candidateIndent)) == "" {
				indent = append([]byte(nil), candidateIndent...)
			}
		}
		updated = append(updated, '\n')
		for line := range bytes.SplitSeq(bytes.TrimSpace(replacement), []byte{'\n'}) {
			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}
			updated = append(updated, indent...)
			updated = append(updated, line...)
			updated = append(updated, '\n')
		}
		lineStart := bytes.LastIndexByte(method[:end], '\n') + 1
		closingIndent := method[lineStart:end]
		if strings.TrimSpace(string(closingIndent)) == "" {
			updated = append(updated, closingIndent...)
		}
	}
	updated = append(updated, method[end:]...)
	if err := ValidateSource(updated); err != nil {
		return nil, fmt.Errorf("replacement produces invalid Java method syntax: %w", err)
	}
	return updated, nil
}

func soleJavaMethodBody(lang *gts.Language, root *gts.Node) (*gts.Node, error) {
	var owners []*gts.Node
	var visit func(*gts.Node)
	visit = func(node *gts.Node) {
		if node == nil {
			return
		}
		if node.Type(lang) == javaNodeMethodDeclaration || node.Type(lang) == javaNodeConstructorDeclaration {
			owners = append(owners, node)
		}
		for index := range node.NamedChildCount() {
			visit(node.NamedChild(index))
		}
	}
	visit(root)
	if len(owners) != 1 {
		return nil, fmt.Errorf("java method source must contain exactly one method or constructor")
	}
	for index := range owners[0].NamedChildCount() {
		body := owners[0].NamedChild(index)
		if body.Type(lang) == "block" && body.EndByte() > body.StartByte()+1 {
			return body, nil
		}
	}
	return nil, fmt.Errorf("java method has no replaceable body")
}

// IsPrivateJavaDeclaration reports whether one supported declaration has private visibility.
func IsPrivateJavaDeclaration(source []byte, kind backend.ConstructKind) (bool, error) {
	if err := ValidateDeclarationKind(source, kind); err != nil {
		return false, fmt.Errorf("validate Java declaration visibility target: %w", err)
	}
	trimmed := bytes.TrimSpace(source)
	prefix := []byte("class __Semedit {\n")
	wrapped := make([]byte, 0, len(prefix)+len(trimmed)+4)
	wrapped = append(wrapped, prefix...)
	wrapped = append(wrapped, trimmed...)
	wrapped = append(wrapped, []byte("\n}\n")...)
	lang, root, err := parse(wrapped)
	if err != nil {
		return false, err
	}
	declaration, err := soleWrappedMember(lang, root)
	if err != nil {
		return false, err
	}
	for index := range declaration.NamedChildCount() {
		child := declaration.NamedChild(index)
		if child.Type(lang) != "modifiers" {
			continue
		}
		if slices.Contains(strings.Fields(string(wrapped[child.StartByte():child.EndByte()])), "private") {
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

// IsPublicTopLevelType reports whether the selected name is a public top-level Java type.
func IsPublicTopLevelType(source []byte, name string) (bool, error) {
	if err := ValidateSource(source); err != nil {
		return false, fmt.Errorf("validate Java top-level type source: %w", err)
	}
	if name == "" {
		return false, fmt.Errorf("java top-level type name is required")
	}
	lang, root, err := parse(source)
	if err != nil {
		return false, err
	}
	if root.Type(lang) != "program" {
		return false, fmt.Errorf("java source root is %q, want program: %w", root.Type(lang), ErrInvalidSource)
	}
	typeCount := 0
	selectedPublic := false
	for index := range root.NamedChildCount() {
		declaration := root.NamedChild(index)
		switch declaration.Type(lang) {
		case javaNodeClassDeclaration, javaNodeInterfaceDeclaration, javaNodeEnumDeclaration, javaNodeRecordDeclaration, "annotation_type_declaration":
		default:
			continue
		}
		typeCount++
		declarationName := ""
		isPublic := false
		for childIndex := range declaration.NamedChildCount() {
			child := declaration.NamedChild(childIndex)
			switch child.Type(lang) {
			case javaNodeIdentifier:
				declarationName = string(source[child.StartByte():child.EndByte()])
			case "modifiers":
				for modifier := range strings.FieldsSeq(string(source[child.StartByte():child.EndByte()])) {
					if modifier == "public" {
						isPublic = true
					}
				}
			}
		}
		selectedPublic = declarationName == name && isPublic
	}
	return typeCount == 1 && selectedPublic, nil
}

// JavaImportDeclaration preserves the source and CST boundaries of one import.
type JavaImportDeclaration struct {
	Text       []byte
	Identity   string
	Start      int
	End        int
	RemovalEnd int
}

// JavaImportSet describes top-level imports and a safe insertion boundary.
type JavaImportSet struct {
	Declarations           []JavaImportDeclaration
	InsertionOffset        int
	InsertionWhitespaceEnd int
	InsertionBlankLine     bool
	HasPackage             bool
}

// ParseJavaImports reads top-level Java import declarations and their insertion point.
func ParseJavaImports(source []byte) (JavaImportSet, error) {
	lang, root, err := parse(source)
	if err != nil {
		return JavaImportSet{}, err
	}
	var imports JavaImportSet
	packageInsertionOffset, lastImportOffset := 0, -1
	packageChildIndex, lastImportChildIndex := -1, -1
	for index := range root.NamedChildCount() {
		child := root.NamedChild(index)
		start, end := int(child.StartByte()), int(child.EndByte())
		if start < 0 || end < start || end > len(source) {
			return JavaImportSet{}, fmt.Errorf("java top-level node range [%d,%d) is outside source", start, end)
		}
		switch child.Type(lang) {
		case "package_declaration":
			imports.HasPackage = true
			packageInsertionOffset = end
			packageChildIndex = index
		case "import_declaration":
			identity, identityErr := javaImportNodeIdentity(source, lang, child)
			if identityErr != nil {
				return JavaImportSet{}, fmt.Errorf("parse java import at byte %d: %w", start, identityErr)
			}
			removalEnd := javaImportRemovalEnd(source, root, index, end)
			imports.Declarations = append(imports.Declarations, JavaImportDeclaration{
				Text:       bytes.Clone(source[start:end]),
				Identity:   identity,
				Start:      start,
				End:        end,
				RemovalEnd: removalEnd,
			})
			lastImportOffset = end
			lastImportChildIndex = index
		}
	}
	setJavaImportInsertionBoundary(source, root, &imports, packageInsertionOffset, packageChildIndex, lastImportOffset, lastImportChildIndex)
	return imports, nil
}

func javaImportNodeIdentity(source []byte, lang *gts.Language, node *gts.Node) (string, error) {
	tokens := javaImportNodeTokens(source, lang, node)
	if len(tokens) < 2 || tokens[0] != "import" {
		return "", fmt.Errorf("unexpected import tokens %q", tokens)
	}
	isStatic := tokens[1] == "static"
	nameStart := 1
	if isStatic {
		nameStart++
	}
	nameEnd := len(tokens)
	if tokens[nameEnd-1] == ";" {
		nameEnd--
	}
	if nameEnd <= nameStart {
		return "", fmt.Errorf("import has no qualified name")
	}
	return JavaImportIdentity(strings.Join(tokens[nameStart:nameEnd], ""), isStatic), nil
}

func javaImportNodeTokens(source []byte, lang *gts.Language, node *gts.Node) []string {
	if strings.Contains(node.Type(lang), "comment") {
		return nil
	}
	if node.ChildCount() == 0 {
		start, end := int(node.StartByte()), int(node.EndByte())
		if start < 0 || end < start || end > len(source) {
			return nil
		}
		return []string{string(source[start:end])}
	}
	var tokens []string
	for index := range node.ChildCount() {
		tokens = append(tokens, javaImportNodeTokens(source, lang, node.Child(index))...)
	}
	return tokens
}

func javaImportRemovalEnd(source []byte, root *gts.Node, index, end int) int {
	if index+1 < root.NamedChildCount() {
		nextStart := int(root.NamedChild(index + 1).StartByte())
		if nextStart >= end && nextStart <= len(source) && len(bytes.TrimSpace(source[end:nextStart])) == 0 {
			return nextStart
		}
	} else if len(bytes.TrimSpace(source[end:])) == 0 {
		return len(source)
	}
	return end
}

func setJavaImportInsertionBoundary(source []byte, root *gts.Node, imports *JavaImportSet, packageOffset, packageIndex, lastImportOffset, lastImportIndex int) {
	imports.InsertionOffset = packageOffset
	boundaryChildIndex := packageIndex
	if lastImportOffset >= 0 {
		imports.InsertionOffset = lastImportOffset
		boundaryChildIndex = lastImportIndex
	}
	imports.InsertionWhitespaceEnd = imports.InsertionOffset
	if imports.InsertionOffset == 0 {
		return
	}
	if boundaryChildIndex >= 0 && boundaryChildIndex+1 < root.NamedChildCount() {
		imports.InsertionWhitespaceEnd = int(root.NamedChild(boundaryChildIndex + 1).StartByte())
	} else {
		imports.InsertionWhitespaceEnd = len(source)
	}
	if imports.InsertionWhitespaceEnd < imports.InsertionOffset || imports.InsertionWhitespaceEnd > len(source) {
		imports.InsertionWhitespaceEnd = imports.InsertionOffset
		return
	}
	gap := source[imports.InsertionOffset:imports.InsertionWhitespaceEnd]
	if len(bytes.TrimSpace(gap)) != 0 {
		imports.InsertionWhitespaceEnd = imports.InsertionOffset
		return
	}
	imports.InsertionBlankLine = bytes.Count(gap, []byte("\n")) > 1
}

// JavaImportIdentity returns a token-boundary-safe identity for a qualified import name.
func JavaImportIdentity(name string, isStatic bool) string {
	prefix := "ordinary:"
	if isStatic {
		prefix = "static:"
	}
	return prefix + strings.Join(strings.Split(name, "."), "\x00.\x00")
}

const (
	javaNodeClassBody      = "class_body"
	javaNodeTypeIdentifier = "type_identifier"
)
