// Package javacst provides syntax-checked Java source transformations and construct selection.
// WHY: This file edits class members without selecting source by text matching.
package javacst

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"

	gts "github.com/odvcencio/gotreesitter"
)

// InsertJavaField inserts one field into the selected Java type body.
func InsertJavaField(owner, snippet []byte, fieldName, placement, targetField string) ([]byte, error) {
	lang, root, err := parse(owner)
	if err != nil {
		return nil, err
	}
	ownerNode, body, err := soleJavaTypeBody(lang, root)
	if err != nil {
		return nil, err
	}
	field, err := soleJavaField(snippet, fieldName)
	if err != nil {
		return nil, err
	}
	if hasJavaFieldNamed(lang, body, owner, fieldName) {
		return nil, fmt.Errorf("java field %q already exists", fieldName)
	}
	memberIndent := javaMemberIndent(owner, ownerNode, body, lang)
	offset, err := javaFieldInsertionOffset(owner, snippet, body, field, lang, placement, targetField)
	if err != nil {
		return nil, err
	}
	ownerIndent := lineIndent(owner, int(ownerNode.StartByte()))
	return insertJavaFieldAt(owner, body, offset, ownerIndent, memberIndent, strings.TrimSpace(string(snippet)), placement)
}

func javaFieldInsertionOffset(owner, snippet []byte, body, field *gts.Node, lang *gts.Language, placement, targetField string) (int, error) {
	var offset int
	switch placement {
	case "", "type_end":
		offset = orderedJavaFieldOffset(owner, snippet, body, field, lang)
	case "type_start":
		offset = firstJavaMemberOffset(body, lang, owner)
	case "before_symbol", "after_symbol":
		var err error
		offset, err = anchoredJavaFieldOffset(owner, body, lang, placement, targetField)
		if err != nil {
			return 0, err
		}
	default:
		return 0, fmt.Errorf("unsupported Java field placement %q", placement)
	}
	if offset < int(body.StartByte())+1 || offset >= int(body.EndByte()) {
		return 0, fmt.Errorf("java field insertion offset is outside the selected type: %w", ErrInvalidPath)
	}
	return offset, nil
}

func orderedJavaFieldOffset(owner, snippet []byte, body, field *gts.Node, lang *gts.Language) int {
	offset := int(body.EndByte()) - 1
	newRank, newVisibility := javaFieldOrder(lang, field, snippet)
	for index := range body.NamedChildCount() {
		child := body.NamedChild(index)
		if !isJavaMemberDeclaration(child.Type(lang)) {
			continue
		}
		rank, visibility := javaMemberOrder(lang, child, owner)
		if rank > newRank || rank == newRank && visibility > newVisibility {
			return attachedJavaDocStart(lang, body, index, owner)
		}
	}
	return offset
}

func anchoredJavaFieldOffset(owner []byte, body *gts.Node, lang *gts.Language, placement, target string) (int, error) {
	if strings.TrimSpace(target) == "" {
		return 0, fmt.Errorf("target_symbol is required for %s placement", placement)
	}
	for index := range body.NamedChildCount() {
		child := body.NamedChild(index)
		if child.Type(lang) == javaNodeFieldDeclaration && javaFieldHasName(lang, child, owner, target) {
			if placement == "before_symbol" {
				return attachedJavaDocStart(lang, body, index, owner), nil
			}
			return int(child.EndByte()), nil
		}
	}
	return 0, fmt.Errorf("java field anchor %q does not identify one member", target)
}

func javaMemberIndent(owner []byte, ownerNode, body *gts.Node, lang *gts.Language) string {
	indent := lineIndent(owner, int(ownerNode.StartByte())) + "\t"
	for index := range body.NamedChildCount() {
		child := body.NamedChild(index)
		if isJavaMemberDeclaration(child.Type(lang)) {
			return lineIndent(owner, int(child.StartByte()))
		}
	}
	return indent
}

func insertJavaFieldAt(owner []byte, body *gts.Node, offset int, ownerIndent, memberIndent, fieldText, placement string) ([]byte, error) {
	lineStart := bytes.LastIndexByte(owner[:offset], '\n') + 1
	indentBefore := string(owner[lineStart:offset])
	var insertion string
	closeOffset := int(body.EndByte()) - 1
	switch placement {
	case "after_symbol":
		insertion = "\n" + memberIndent + fieldText
	case "", "type_end":
		if strings.TrimSpace(indentBefore) == "" && lineStart > 0 {
			suffixIndent := memberIndent
			if offset == closeOffset {
				suffixIndent = ownerIndent
			}
			updated := spliceSource(owner, lineStart, offset, []byte(memberIndent+fieldText+"\n"+suffixIndent))
			if err := ValidateSource(updated); err != nil {
				return nil, fmt.Errorf("java field insertion produces invalid syntax: %w", err)
			}
			return updated, nil
		}
		insertion = "\n" + memberIndent + fieldText + "\n" + ownerIndent
	case "type_start", "before_symbol":
		switch {
		case offset == closeOffset:
			if strings.TrimSpace(indentBefore) == "" && lineStart > 0 {
				updated := spliceSource(owner, lineStart, offset, []byte(memberIndent+fieldText+"\n"+ownerIndent))
				if err := ValidateSource(updated); err != nil {
					return nil, fmt.Errorf("java field insertion produces invalid syntax: %w", err)
				}
				return updated, nil
			}
			insertion = "\n" + memberIndent + fieldText + "\n" + ownerIndent
		case strings.TrimSpace(indentBefore) == "" && lineStart > 0:
			insertion = fieldText + "\n" + memberIndent
		default:
			insertion = "\n" + memberIndent + fieldText + "\n" + memberIndent
		}
	}
	updated := spliceSource(owner, offset, offset, []byte(insertion))
	if err := ValidateSource(updated); err != nil {
		return nil, fmt.Errorf("java field insertion produces invalid syntax: %w", err)
	}
	return updated, nil
}

// AddJavaImplementedInterface adds one interface to the selected Java type.
func AddJavaImplementedInterface(source []byte, typeName, interfaceName string) ([]byte, bool, error) {
	if strings.TrimSpace(typeName) == "" || strings.TrimSpace(interfaceName) == "" || strings.TrimSpace(typeName) != typeName || strings.TrimSpace(interfaceName) != interfaceName {
		return nil, false, fmt.Errorf("java type and interface names must be non-empty selectors")
	}
	lang, root, err := parse(source)
	if err != nil {
		return nil, false, err
	}
	var matches []*gts.Node
	var visit func(*gts.Node)
	visit = func(node *gts.Node) {
		if node == nil {
			return
		}
		if node.Type(lang) == javaNodeClassDeclaration {
			for index := range node.NamedChildCount() {
				child := node.NamedChild(index)
				if child.Type(lang) == javaNodeIdentifier && string(source[child.StartByte():child.EndByte()]) == typeName {
					matches = append(matches, node)
					break
				}
			}
		}
		for index := range node.NamedChildCount() {
			visit(node.NamedChild(index))
		}
	}
	visit(root)
	if len(matches) != 1 {
		return nil, false, fmt.Errorf("java class %q resolves to %d declarations", typeName, len(matches))
	}
	class := matches[0]
	var body, interfaces *gts.Node
	for index := range class.NamedChildCount() {
		child := class.NamedChild(index)
		switch child.Type(lang) {
		case javaNodeClassBody:
			body = child
		case "super_interfaces":
			interfaces = child
		}
	}
	if body == nil {
		return nil, false, fmt.Errorf("java class %q has no body", typeName)
	}
	if interfaces != nil {
		text := string(source[interfaces.StartByte():interfaces.EndByte()])
		for existing := range strings.SplitSeq(strings.TrimSpace(strings.TrimPrefix(text, "implements")), ",") {
			if strings.TrimSpace(existing) == interfaceName {
				return bytes.Clone(source), false, nil
			}
		}
		return insertJavaTypeText(source, int(interfaces.EndByte()), ", "+interfaceName)
	}
	insert := "implements " + interfaceName + " "
	if int(body.StartByte()) > 0 && !isJavaWhitespace(source[int(body.StartByte())-1]) {
		insert = " " + insert
	}
	return insertJavaTypeText(source, int(body.StartByte()), insert)
}

func insertJavaTypeText(source []byte, offset int, insertion string) ([]byte, bool, error) {
	if offset < 0 || offset > len(source) {
		return nil, false, fmt.Errorf("java interface insertion offset is outside the source")
	}
	updated := make([]byte, 0, len(source)+len(insertion))
	updated = append(updated, source[:offset]...)
	updated = append(updated, insertion...)
	updated = append(updated, source[offset:]...)
	if _, _, err := parse(updated); err != nil {
		return nil, false, fmt.Errorf("validate Java implements clause: %w", err)
	}
	return updated, true, nil
}

func isJavaWhitespace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\n' || value == '\r'
}

// InsertJavaMember inserts one method, constructor, or nested type into a Java type.
func InsertJavaMember(owner, snippet []byte, memberName, placement, targetMember string) ([]byte, error) {
	lang, root, err := parse(owner)
	if err != nil {
		return nil, err
	}
	ownerNode, body, err := soleJavaTypeBody(lang, root)
	if err != nil {
		return nil, err
	}
	declarationType, names, err := declarationIdentity(snippet)
	if err != nil {
		return nil, err
	}
	switch declarationType {
	case javaNodeMethodDeclaration, javaNodeConstructorDeclaration, javaNodeClassDeclaration, javaNodeInterfaceDeclaration, javaNodeEnumDeclaration, javaNodeRecordDeclaration:
	default:
		return nil, fmt.Errorf("java member insertion requires exactly one method, constructor, or nested type declaration")
	}
	if len(names) != 1 || names[0] != memberName {
		return nil, fmt.Errorf("java member source name does not match selected name %q", memberName)
	}
	wrapped := []byte("class __Semedit {\n" + string(snippet) + "\n}")
	_, wrappedRoot, err := parse(wrapped)
	if err != nil {
		return nil, err
	}
	member, err := soleWrappedMember(lang, wrappedRoot)
	if err != nil {
		return nil, err
	}
	memberIndent := javaMemberIndent(owner, ownerNode, body, lang)
	newRank, newVisibility := javaMemberOrder(lang, member, wrapped)
	offset, err := javaMemberInsertionOffset(owner, body, lang, placement, targetMember, newRank, newVisibility)
	if err != nil {
		return nil, err
	}
	memberText := strings.TrimSpace(string(snippet))
	lineStart := bytes.LastIndexByte(owner[:offset], '\n') + 1
	indentBefore := string(owner[lineStart:offset])
	closeOffset := int(body.EndByte()) - 1
	var updated []byte
	switch {
	case (placement == "" || placement == "type_end" || placement == "type_start") && offset == closeOffset && strings.TrimSpace(indentBefore) == "" && lineStart > 0:
		updated = spliceSource(owner, lineStart, offset, []byte(memberIndent+memberText+"\n"+lineIndent(owner, int(ownerNode.StartByte()))))
	case offset == closeOffset:
		updated = spliceSource(owner, offset, offset, []byte("\n"+memberIndent+memberText+"\n"+lineIndent(owner, int(ownerNode.StartByte()))))
	case strings.TrimSpace(indentBefore) == "" && lineStart > 0:
		updated = spliceSource(owner, offset, offset, []byte(memberText+"\n"+memberIndent))
	default:
		updated = spliceSource(owner, offset, offset, []byte("\n"+memberIndent+memberText+"\n"+memberIndent))
	}
	if err := ValidateSource(updated); err != nil {
		return nil, fmt.Errorf("java member insertion produces invalid syntax: %w", err)
	}
	return updated, nil
}

func javaMemberInsertionOffset(owner []byte, body *gts.Node, lang *gts.Language, placement, target string, newRank, newVisibility int) (int, error) {
	var offset int
	switch placement {
	case "", "type_end":
		offset = orderedJavaMemberOffset(owner, body, lang, newRank, newVisibility)
	case "type_start":
		offset = firstJavaMemberOffset(body, lang, owner)
	case "before_symbol", "after_symbol":
		var err error
		offset, err = anchoredJavaMemberOffset(owner, body, lang, placement, target)
		if err != nil {
			return 0, err
		}
	default:
		return 0, fmt.Errorf("unsupported Java member placement %q", placement)
	}
	if offset < int(body.StartByte())+1 || offset >= int(body.EndByte()) {
		return 0, fmt.Errorf("java member insertion offset is outside the selected type: %w", ErrInvalidPath)
	}
	return offset, nil
}

func orderedJavaMemberOffset(owner []byte, body *gts.Node, lang *gts.Language, newRank, newVisibility int) int {
	offset := int(body.EndByte()) - 1
	for index := range body.NamedChildCount() {
		child := body.NamedChild(index)
		if !isJavaMemberDeclaration(child.Type(lang)) {
			continue
		}
		rank, visibility := javaMemberOrder(lang, child, owner)
		if rank > newRank || rank == newRank && visibility > newVisibility {
			return attachedJavaDocStart(lang, body, index, owner)
		}
	}
	return offset
}

func firstJavaMemberOffset(body *gts.Node, lang *gts.Language, owner []byte) int {
	for index := range body.NamedChildCount() {
		if isJavaMemberDeclaration(body.NamedChild(index).Type(lang)) {
			return attachedJavaDocStart(lang, body, index, owner)
		}
	}
	return int(body.EndByte()) - 1
}

func anchoredJavaMemberOffset(owner []byte, body *gts.Node, lang *gts.Language, placement, target string) (int, error) {
	if target == "" {
		return 0, fmt.Errorf("target_symbol is required for %s placement", placement)
	}
	var anchor *gts.Node
	anchorIndex := -1
	for index := range body.NamedChildCount() {
		child := body.NamedChild(index)
		if !javaMemberHasName(lang, child, owner, target) {
			continue
		}
		if anchor != nil {
			return 0, fmt.Errorf("java member anchor %q resolves to multiple declarations", target)
		}
		anchor, anchorIndex = child, index
	}
	if anchor == nil {
		return 0, fmt.Errorf("java member anchor %q resolves to 0 declarations", target)
	}
	if placement == "after_symbol" {
		return int(anchor.EndByte()), nil
	}
	return attachedJavaDocStart(lang, body, anchorIndex, owner), nil
}

// JavaDeclarationName returns the declared name of one Java member or type.
func JavaDeclarationName(source []byte) (string, error) {
	_, names, err := declarationIdentity(source)
	if err != nil {
		return "", err
	}
	if len(names) != 1 {
		return "", fmt.Errorf("java declaration must have exactly one name")
	}
	return names[0], nil
}

// ValidateJavaIdentifier rejects names that are not Java identifiers.
func ValidateJavaIdentifier(name string) error {
	if strings.TrimSpace(name) != name || name == "" {
		return fmt.Errorf("invalid Java identifier %q", name)
	}
	_, err := soleJavaField([]byte("int "+name+";"), name)
	return err
}

// ExtractedLocalIdentifier returns the local variable initialized by the selected expression.
func ExtractedLocalIdentifier(source, initializer []byte) (string, int, int, error) {
	lang, root, err := parse(source)
	if err != nil {
		return "", 0, 0, err
	}
	wanted := compact(string(initializer))
	var matches []struct {
		name       string
		start, end int
	}
	var visit func(*gts.Node)
	visit = func(node *gts.Node) {
		if node == nil {
			return
		}
		if node.Type(lang) == javaNodeVariableDeclarator && node.NamedChildCount() >= 2 {
			identifier := node.NamedChild(0)
			value := node.NamedChild(node.NamedChildCount() - 1)
			if identifier.Type(lang) == javaNodeIdentifier && compact(string(source[value.StartByte():value.EndByte()])) == wanted {
				matches = append(matches, struct {
					name       string
					start, end int
				}{string(source[identifier.StartByte():identifier.EndByte()]), int(identifier.StartByte()), int(identifier.EndByte())})
			}
		}
		for index := range node.NamedChildCount() {
			visit(node.NamedChild(index))
		}
	}
	visit(root)
	if len(matches) != 1 {
		return "", 0, 0, fmt.Errorf("extracted local initializer identifies %d declarations", len(matches))
	}
	match := matches[0]
	return match.name, match.start, match.end, nil
}

// JavaLocalDeclaration locates one local variable declaration by its name.
func JavaLocalDeclaration(source []byte, name string) (Candidate, error) {
	lang, root, err := parse(source)
	if err != nil {
		return Candidate{}, err
	}
	var matches []Candidate
	var visit func(*gts.Node, string)
	visit = func(node *gts.Node, parentPath string) {
		if node == nil {
			return
		}
		if node.Type(lang) == "local_variable_declaration" {
			found := false
			for index := range node.NamedChildCount() {
				child := node.NamedChild(index)
				if child.Type(lang) != javaNodeVariableDeclarator || child.NamedChildCount() == 0 {
					continue
				}
				identifier := child.NamedChild(0)
				if identifier.Type(lang) == javaNodeIdentifier && string(source[identifier.StartByte():identifier.EndByte()]) == name {
					found = true
					break
				}
			}
			if found {
				matches = append(matches, newCandidate(source, node, parentPath+"/local_variable_declaration", strings.TrimSpace(string(source[node.StartByte():node.EndByte()])), backend.ConstructStatementOrExpr))
			}
		}
		for index := range node.NamedChildCount() {
			visit(node.NamedChild(index), parentPath)
		}
	}
	visit(root, "")
	if len(matches) != 1 {
		return Candidate{}, fmt.Errorf("java local %q resolves to %d declarations", name, len(matches))
	}
	return matches[0], nil
}

// JavaStatementRange selects the contiguous direct statements spanning the requested lines.
func JavaStatementRange(source []byte, startLine, endLine int) (Candidate, error) {
	lang, root, err := parse(source)
	if err != nil {
		return Candidate{}, err
	}
	if startLine < 0 || endLine < startLine {
		return Candidate{}, fmt.Errorf("invalid Java statement line range %d-%d", startLine, endLine)
	}
	matches := javaStatementRangeCandidates(source, lang, root, startLine, endLine)
	if len(matches) != 1 {
		return Candidate{}, fmt.Errorf("java statement range %d-%d resolves to %d direct statement selections", startLine+1, endLine+1, len(matches))
	}
	return matches[0], nil
}

func javaStatementRangeCandidates(source []byte, lang *gts.Language, root *gts.Node, startLine, endLine int) []Candidate {
	var matches []Candidate
	var visit func(*gts.Node, string)
	visit = func(node *gts.Node, parentPath string) {
		if node == nil {
			return
		}
		if node.Type(lang) == "block" {
			var statements []*gts.Node
			for index := range node.NamedChildCount() {
				child := node.NamedChild(index)
				if isJavaStatement(child.Type(lang)) {
					statements = append(statements, child)
				}
			}
			for first := range statements {
				start := int(statements[first].StartByte())
				if backend.PositionFromByteOffset(source, start).Line != startLine {
					continue
				}
				for last := first; last < len(statements); last++ {
					end := int(statements[last].EndByte())
					if backend.PositionFromByteOffset(source, end).Line != endLine {
						continue
					}
					matches = append(matches, Candidate{
						ConstructPath: parentPath + "/statement_range",
						Syntax:        strings.TrimSpace(string(source[start:end])),
						Kind:          backend.ConstructStatementOrExpr,
						StartByte:     uint32(start), EndByte: uint32(end), // #nosec G115 -- source spans originate from parser-checked node boundaries.
						Range: backend.Range{Start: backend.PositionFromByteOffset(source, start), End: backend.PositionFromByteOffset(source, end)},
					})
				}
			}
		}
		for index := range node.NamedChildCount() {
			visit(node.NamedChild(index), parentPath+"/"+node.Type(lang))
		}
	}
	visit(root, "")
	return matches
}

func isJavaStatement(kind string) bool {
	switch kind {
	case "assert_statement", "block", "break_statement", "continue_statement", "do_statement", "empty_statement", "enhanced_for_statement", "expression_statement", "for_statement", "if_statement", "labeled_statement", "local_variable_declaration", "return_statement", "switch_expression", "switch_statement", "synchronized_statement", "throw_statement", "try_statement", "try_with_resources_statement", "while_statement", "yield_statement":
		return true
	default:
		return false
	}
}

// ExtractedMethodIdentifier returns the method name and source span for an extraction.
func ExtractedMethodIdentifier(source, selectedBody []byte) (string, int, int, error) {
	lang, root, err := parse(source)
	if err != nil {
		return "", 0, 0, err
	}
	wanted := compact(string(selectedBody))
	var matches []struct {
		name       string
		start, end int
	}
	var visit func(*gts.Node)
	visit = func(node *gts.Node) {
		if node == nil {
			return
		}
		if node.Type(lang) == javaNodeMethodDeclaration {
			for index := 0; index+1 < node.NamedChildCount(); index++ {
				identifier := node.NamedChild(index)
				if identifier.Type(lang) != javaNodeIdentifier || node.NamedChild(index+1).Type(lang) != "formal_parameters" {
					continue
				}
				body := node.NamedChild(node.NamedChildCount() - 1)
				if wanted != "" && body.Type(lang) == "block" && strings.Contains(compact(string(source[body.StartByte():body.EndByte()])), wanted) {
					matches = append(matches, struct {
						name       string
						start, end int
					}{string(source[identifier.StartByte():identifier.EndByte()]), int(identifier.StartByte()), int(identifier.EndByte())})
				}
				break
			}
		}
		for index := range node.NamedChildCount() {
			visit(node.NamedChild(index))
		}
	}
	visit(root)
	if len(matches) != 1 {
		return "", 0, 0, fmt.Errorf("extracted method body identifies %d declarations", len(matches))
	}
	match := matches[0]
	return match.name, match.start, match.end, nil
}

// SoleJavaTopLevelType returns the name when source contains exactly one top-level type.
func SoleJavaTopLevelType(source []byte) (string, error) {
	lang, root, err := parse(source)
	if err != nil {
		return "", err
	}
	var types []*gts.Node
	for index := range root.NamedChildCount() {
		child := root.NamedChild(index)
		switch child.Type(lang) {
		case javaNodeClassDeclaration, javaNodeEnumDeclaration, javaNodeRecordDeclaration, javaNodeInterfaceDeclaration:
			types = append(types, child)
		}
	}
	if len(types) != 1 {
		return "", fmt.Errorf("java field insertion without in_type requires exactly one top-level type")
	}
	for index := range types[0].NamedChildCount() {
		child := types[0].NamedChild(index)
		if child.Type(lang) == javaNodeIdentifier || child.Type(lang) == javaNodeTypeIdentifier {
			return string(source[child.StartByte():child.EndByte()]), nil
		}
	}
	return "", fmt.Errorf("java top-level type has no name")
}

func soleJavaTypeBody(lang *gts.Language, root *gts.Node) (*gts.Node, *gts.Node, error) {
	var candidates []*gts.Node
	for index := range root.NamedChildCount() {
		node := root.NamedChild(index)
		switch node.Type(lang) {
		case javaNodeClassDeclaration, javaNodeEnumDeclaration, javaNodeRecordDeclaration, javaNodeInterfaceDeclaration:
			candidates = append(candidates, node)
		}
	}
	if len(candidates) != 1 {
		return nil, nil, fmt.Errorf("java field insertion requires one selected class, enum, or record declaration")
	}
	for index := range candidates[0].NamedChildCount() {
		child := candidates[0].NamedChild(index)
		if child.Type(lang) == javaNodeClassBody || child.Type(lang) == "enum_body" || child.Type(lang) == "enum_body_declarations" || child.Type(lang) == "interface_body" {
			return candidates[0], child, nil
		}
	}
	return nil, nil, fmt.Errorf("selected Java type has no member body")
}

func soleJavaField(snippet []byte, expected string) (*gts.Node, error) {
	if err := ValidateSource([]byte("class __Semedit {\n" + string(snippet) + "\n}")); err != nil {
		return nil, err
	}
	lang, root, err := parse([]byte("class __Semedit {\n" + string(snippet) + "\n}"))
	if err != nil {
		return nil, err
	}
	member, err := soleWrappedMember(lang, root)
	if err != nil {
		return nil, err
	}
	if member.Type(lang) != javaNodeFieldDeclaration || !javaFieldHasName(lang, member, []byte("class __Semedit {\n"+string(snippet)+"\n}"), expected) {
		return nil, fmt.Errorf("java field source must declare exactly %q", expected)
	}
	return member, nil
}

func hasJavaFieldNamed(lang *gts.Language, body *gts.Node, source []byte, name string) bool {
	for index := range body.NamedChildCount() {
		child := body.NamedChild(index)
		if child.Type(lang) == javaNodeFieldDeclaration && javaFieldHasName(lang, child, source, name) {
			return true
		}
	}
	return false
}

func javaFieldHasName(lang *gts.Language, declaration *gts.Node, source []byte, name string) bool {
	for index := range declaration.NamedChildCount() {
		child := declaration.NamedChild(index)
		if child.Type(lang) != javaNodeVariableDeclarator || child.NamedChildCount() == 0 {
			continue
		}
		identifier := child.NamedChild(0)
		if identifier.Type(lang) == javaNodeIdentifier && string(source[identifier.StartByte():identifier.EndByte()]) == name {
			return true
		}
	}
	return false
}

func javaMemberHasName(lang *gts.Language, member *gts.Node, source []byte, name string) bool {
	if member.Type(lang) == javaNodeFieldDeclaration {
		return javaFieldHasName(lang, member, source, name)
	}
	for index := range member.NamedChildCount() {
		child := member.NamedChild(index)
		if (child.Type(lang) == javaNodeIdentifier || child.Type(lang) == javaNodeTypeIdentifier) && string(source[child.StartByte():child.EndByte()]) == name {
			return true
		}
	}
	return false
}

func javaFieldOrder(lang *gts.Language, field *gts.Node, source []byte) (int, int) {
	wrapper := []byte("class __Semedit {\n" + string(source) + "\n}")
	return javaMemberOrder(lang, field, wrapper)
}

func javaMemberOrder(lang *gts.Language, member *gts.Node, source []byte) (int, int) {
	switch member.Type(lang) {
	case javaNodeFieldDeclaration:
		modifiers := javaModifiers(lang, member)
		rank := 2
		if modifiers["static"] {
			rank = 1
			if modifiers["final"] {
				rank = 0
			}
		}
		visibility := javaVisibility(lang, member, source)
		return rank, visibility
	case javaNodeConstructorDeclaration:
		return 3, javaVisibility(lang, member, source)
	case javaNodeMethodDeclaration:
		return 4, javaVisibility(lang, member, source)
	case javaNodeClassDeclaration, javaNodeInterfaceDeclaration, javaNodeEnumDeclaration, javaNodeRecordDeclaration:
		return 5, javaVisibility(lang, member, source)
	default:
		return 6, 0
	}
}

func javaVisibility(lang *gts.Language, member *gts.Node, source []byte) int {
	for index := range member.NamedChildCount() {
		child := member.NamedChild(index)
		if child.Type(lang) != "modifiers" {
			continue
		}
		modifiers := strings.FieldsSeq(string(source[child.StartByte():child.EndByte()]))
		for modifier := range modifiers {
			switch modifier {
			case "public":
				return 0
			case "protected":
				return 1
			case "private":
				return 3
			}
		}
	}
	return 2
}

func attachedJavaDocStart(lang *gts.Language, body *gts.Node, memberIndex int, source []byte) int {
	member := body.NamedChild(memberIndex)
	start := int(member.StartByte())
	currentStart := start
	for index := memberIndex - 1; index >= 0; index-- {
		previous := body.NamedChild(index)
		if previous.Type(lang) != "block_comment" {
			break
		}
		gap := source[previous.EndByte():currentStart]
		if !allJavaWhitespace(gap) || bytes.Count(gap, []byte("\n")) > 1 {
			break
		}
		start = int(previous.StartByte())
		currentStart = start
	}
	return start
}

type javaMemberSelection struct {
	node   *gts.Node
	parent *gts.Node
	index  int
}

// SetJavaDocComment replaces or removes documentation on the selected Java declaration.
func SetJavaDocComment(source []byte, start, end int, expected []byte, comment string) ([]byte, error) {
	if start < 0 || end <= start || end > len(source) || !bytes.Equal(source[start:end], expected) {
		return nil, fmt.Errorf("java declaration range does not match its source preimage: %w", ErrInvalidPath)
	}
	if strings.Contains(comment, "*/") {
		return nil, fmt.Errorf("java documentation comment cannot contain */")
	}
	lang, root, err := parse(source)
	if err != nil {
		return nil, err
	}
	selected := findJavaMemberSelection(lang, root, start, end)
	if selected.node == nil {
		return nil, fmt.Errorf("java declaration range does not identify one member: %w", ErrInvalidPath)
	}
	commentStart := start
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
		commentStart = int(previous.StartByte())
		currentStart = commentStart
	}
	lineStart := bytes.LastIndexByte(source[:commentStart], '\n') + 1
	indent := string(source[lineStart:commentStart])
	if strings.TrimSpace(indent) != "" {
		return nil, fmt.Errorf("java declaration documentation must begin on its own line: %w", ErrInvalidPath)
	}
	if comment == "" {
		if commentStart == start {
			return append([]byte(nil), source...), nil
		}
		updated := spliceSource(source, commentStart, start, nil)
		if err := ValidateSource(updated); err != nil {
			return nil, fmt.Errorf("documentation removal produces invalid Java syntax: %w", err)
		}
		return updated, nil
	}
	lines := strings.Split(strings.ReplaceAll(comment, "\r\n", "\n"), "\n")
	var doc strings.Builder
	doc.WriteString("/**\n")
	for _, line := range lines {
		doc.WriteString(indent)
		doc.WriteString(" *")
		if line != "" {
			doc.WriteByte(' ')
			doc.WriteString(line)
		}
		doc.WriteByte('\n')
	}
	doc.WriteString(indent)
	doc.WriteString(" */\n")
	doc.WriteString(indent)
	updated := spliceSource(source, commentStart, start, []byte(doc.String()))
	if err := ValidateSource(updated); err != nil {
		return nil, fmt.Errorf("documentation update produces invalid Java syntax: %w", err)
	}
	return updated, nil
}

func findJavaMemberSelection(lang *gts.Language, root *gts.Node, start, end int) javaMemberSelection {
	var selected javaMemberSelection
	var find func(*gts.Node, *gts.Node, int)
	find = func(node, parent *gts.Node, index int) {
		if node == nil || selected.node != nil {
			return
		}
		if int(node.StartByte()) == start && int(node.EndByte()) == end && isJavaMemberDeclaration(node.Type(lang)) {
			selected = javaMemberSelection{node: node, parent: parent, index: index}
			return
		}
		for childIndex := range node.NamedChildCount() {
			find(node.NamedChild(childIndex), node, childIndex)
		}
	}
	find(root, nil, 0)
	return selected
}

func javaModifiers(lang *gts.Language, member *gts.Node) map[string]bool {
	result := make(map[string]bool)
	for index := range member.NamedChildCount() {
		modifiers := member.NamedChild(index)
		if modifiers.Type(lang) != "modifiers" {
			continue
		}
		for modifierIndex := range modifiers.ChildCount() {
			modifier := modifiers.Child(modifierIndex)
			switch modifier.Type(lang) {
			case "static", "final":
				result[modifier.Type(lang)] = true
			}
		}
		break
	}
	return result
}

func lineIndent(source []byte, offset int) string {
	if offset < 0 || offset > len(source) {
		return ""
	}
	start := bytes.LastIndexByte(source[:offset], '\n') + 1
	index := start
	for index < offset && (source[index] == ' ' || source[index] == '\t') {
		index++
	}
	return string(source[start:index])
}

// InferredLocal identifies the declaration, name, and var token of one method-local binding.
type InferredLocal struct {
	Declaration Candidate
	Identifier  backend.Range
	Type        Candidate
}

// JavaInferredLocal resolves one unannotated, single-declarator var in a method body.
func JavaInferredLocal(source []byte, name string) (InferredLocal, error) {
	lang, root, err := parse(source)
	if err != nil {
		return InferredLocal{}, err
	}
	declarations, forbidden := javaInferredLocalDeclarations(source, name, lang, root)
	if len(declarations) != 1 {
		return InferredLocal{}, fmt.Errorf("java inferred local %q resolves to %d declarations", name, len(declarations))
	}
	if forbidden {
		return InferredLocal{}, fmt.Errorf("java inferred local %q is inside an unsupported binding scope", name)
	}
	return javaInferredLocalTokens(source, name, lang, declarations[0])
}

// ValidateJavaExplicitType checks the native replacement without inferring or rewriting its type.
func ValidateJavaExplicitType(value string) error {
	if value == "" || value != strings.TrimSpace(value) || value == "var" || strings.ContainsAny(value, "\r\n@;{}") || strings.Contains(value, "/*") || strings.Contains(value, "//") {
		return fmt.Errorf("java explicit type is not denotable")
	}
	source := []byte("class TypeProbe { " + value + " value; }")
	lang, root, err := parse(source)
	if err != nil {
		return fmt.Errorf("java explicit type is not denotable: %w", err)
	}
	var typeNode *gts.Node
	var visit func(*gts.Node)
	visit = func(node *gts.Node) {
		if node.Type(lang) == javaNodeFieldDeclaration {
			typeNode = node.ChildByFieldName("type", lang)
			return
		}
		for index := range node.NamedChildCount() {
			visit(node.NamedChild(index))
		}
	}
	visit(root)
	if typeNode == nil || string(source[typeNode.StartByte():typeNode.EndByte()]) != value {
		return fmt.Errorf("java explicit type is not one complete denotable type")
	}
	if javaExplicitTypeContainsVar(source, typeNode) {
		return fmt.Errorf("java explicit type contains reserved var component")
	}
	switch typeNode.Type(lang) {
	case "integral_type", "floating_point_type", "boolean_type", javaNodeTypeIdentifier, "scoped_type_identifier", "generic_type", "array_type":
		return nil
	default:
		return fmt.Errorf("java explicit type %q is not supported", value)
	}
}

// JavaInferredLocalAt binds an identifier selection to its containing method and fresh CST tokens.
func JavaInferredLocalAt(source []byte, name string, selection backend.Range) (InferredLocal, error) {
	lang, root, err := parse(source)
	if err != nil {
		return InferredLocal{}, err
	}
	start, err := ByteOffsetFromPosition(source, selection.Start)
	if err != nil {
		return InferredLocal{}, err
	}
	end, err := ByteOffsetFromPosition(source, selection.End)
	if err != nil {
		return InferredLocal{}, err
	}
	var owners []*gts.Node
	var visit func(*gts.Node)
	visit = func(node *gts.Node) {
		if node.Type(lang) == javaNodeMethodDeclaration && int(node.StartByte()) <= start && end <= int(node.EndByte()) {
			owners = append(owners, node)
		}
		for index := range node.NamedChildCount() {
			visit(node.NamedChild(index))
		}
	}
	visit(root)
	if len(owners) != 1 {
		return InferredLocal{}, fmt.Errorf("java inferred local selection requires one containing method")
	}
	base := int(owners[0].StartByte())
	methodSource := source[base:owners[0].EndByte()]
	local, err := JavaInferredLocal(methodSource, name)
	if err != nil {
		return InferredLocal{}, err
	}
	localStart, err := ByteOffsetFromPosition(methodSource, local.Identifier.Start)
	if err != nil {
		return InferredLocal{}, err
	}
	localEnd, err := ByteOffsetFromPosition(methodSource, local.Identifier.End)
	if err != nil {
		return InferredLocal{}, err
	}
	local.Identifier = backend.Range{Start: backend.PositionFromByteOffset(source, base+localStart), End: backend.PositionFromByteOffset(source, base+localEnd)}
	if local.Identifier != selection {
		return InferredLocal{}, fmt.Errorf("java inferred local selection must exactly cover its identifier")
	}
	for _, candidate := range []*Candidate{&local.Declaration, &local.Type} {
		candidate.StartByte += owners[0].StartByte()
		candidate.EndByte += owners[0].StartByte()
		candidate.Range = backend.Range{Start: backend.PositionFromByteOffset(source, int(candidate.StartByte)), End: backend.PositionFromByteOffset(source, int(candidate.EndByte))}
	}
	return local, nil
}

func javaInferredLocalHasAnonymousClass(node *gts.Node, lang *gts.Language) bool {
	if node.Type(lang) == "object_creation_expression" {
		for index := range node.NamedChildCount() {
			if node.NamedChild(index).Type(lang) == javaNodeClassBody {
				return true
			}
		}
	}
	for index := range node.NamedChildCount() {
		if javaInferredLocalHasAnonymousClass(node.NamedChild(index), lang) {
			return true
		}
	}
	return false
}

func javaInferredLocalDeclarations(source []byte, name string, lang *gts.Language, root *gts.Node) ([]*gts.Node, bool) {
	var declarations []*gts.Node
	var forbiddenMatch bool
	var visit func(*gts.Node, bool)
	visit = func(node *gts.Node, forbidden bool) {
		if node == nil {
			return
		}
		switch node.Type(lang) {
		case "lambda_expression", "for_statement", "enhanced_for_statement", "while_statement", "do_statement", "resource", "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "object_creation_expression":
			forbidden = true
		}
		if node.Type(lang) == "local_variable_declaration" {
			for index := range node.NamedChildCount() {
				child := node.NamedChild(index)
				if child.Type(lang) != javaNodeVariableDeclarator {
					continue
				}
				identifier := child.ChildByFieldName("name", lang)
				if identifier != nil && string(source[identifier.StartByte():identifier.EndByte()]) == name {
					declarations = append(declarations, node)
					forbiddenMatch = forbiddenMatch || forbidden
				}
			}
		}
		for index := range node.NamedChildCount() {
			visit(node.NamedChild(index), forbidden)
		}
	}
	visit(root, false)
	return declarations, forbiddenMatch
}

func javaInferredLocalTokens(source []byte, name string, lang *gts.Language, declaration *gts.Node) (InferredLocal, error) {
	typeNode := declaration.ChildByFieldName("type", lang)
	if typeNode == nil || string(source[typeNode.StartByte():typeNode.EndByte()]) != "var" {
		return InferredLocal{}, fmt.Errorf("java local %q must have inferred var type", name)
	}
	var identifier *gts.Node
	count := 0
	for index := range declaration.NamedChildCount() {
		child := declaration.NamedChild(index)
		switch child.Type(lang) {
		case javaNodeVariableDeclarator:
			count++
			identifier = child.ChildByFieldName("name", lang)
			if child.ChildByFieldName("value", lang) == nil {
				return InferredLocal{}, fmt.Errorf("java inferred local %q has no initializer", name)
			}
		case "modifiers":
			return InferredLocal{}, fmt.Errorf("java inferred local %q must be unannotated and unmodified", name)
		}
	}
	if count != 1 || identifier == nil {
		return InferredLocal{}, fmt.Errorf("java inferred local %q requires one declarator", name)
	}
	if javaInferredLocalHasAnonymousClass(declaration, lang) {
		return InferredLocal{}, fmt.Errorf("java inferred local %q has a non-denotable anonymous class type", name)
	}
	return InferredLocal{
		Declaration: newCandidate(source, declaration, "", "", backend.ConstructStatementOrExpr),
		Identifier:  backend.Range{Start: backend.PositionFromByteOffset(source, int(identifier.StartByte())), End: backend.PositionFromByteOffset(source, int(identifier.EndByte()))},
		Type:        newCandidate(source, typeNode, "", "var", backend.ConstructStatementOrExpr),
	}, nil
}

func javaExplicitTypeContainsVar(source []byte, node *gts.Node) bool {
	if string(source[node.StartByte():node.EndByte()]) == "var" {
		return true
	}
	for index := range node.NamedChildCount() {
		if javaExplicitTypeContainsVar(source, node.NamedChild(index)) {
			return true
		}
	}
	return false
}
