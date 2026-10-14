// Package javacst provides syntax-checked Java source transformations and construct selection.
// WHY: This file extracts interface signatures and appends validated top-level contracts.
package javacst

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"

	gts "github.com/odvcencio/gotreesitter"
)

// JavaInterfaceMethodSignature returns the public method signature for interface extraction.
func JavaInterfaceMethodSignature(source []byte) (string, error) {
	lang, root, err := parse(source)
	if err != nil {
		return "", err
	}
	methods := javaMethodDeclarations(lang, root)
	if len(methods) != 1 {
		return "", fmt.Errorf("java interface extraction requires exactly one method declaration")
	}
	method := methods[0]
	modifiers, body := javaMethodModifiersAndBody(lang, method)
	if body == nil {
		return "", fmt.Errorf("java interface extraction requires a method with a body")
	}
	if modifiers == nil {
		return "", fmt.Errorf("java interface extraction selects public methods only")
	}
	if err := validateJavaInterfaceMethodModifiers(source, modifiers); err != nil {
		return "", err
	}
	start, end := int(method.StartByte()), int(body.StartByte())
	if start < 0 || end <= start || end > len(source) {
		return "", fmt.Errorf("java method signature range is invalid")
	}
	signature := strings.TrimSpace(string(source[start:end]))
	if signature == "" || strings.ContainsRune(signature, '{') || strings.Contains(signature, "@") {
		return "", fmt.Errorf("java method signature contains unsupported syntax")
	}
	return signature + ";", nil
}

func javaMethodDeclarations(language *gts.Language, root *gts.Node) []*gts.Node {
	var methods []*gts.Node
	var visit func(*gts.Node)
	visit = func(node *gts.Node) {
		if node == nil {
			return
		}
		if node.Type(language) == javaNodeMethodDeclaration {
			methods = append(methods, node)
		}
		for index := range node.NamedChildCount() {
			visit(node.NamedChild(index))
		}
	}
	visit(root)
	return methods
}

func javaMethodModifiersAndBody(language *gts.Language, method *gts.Node) (*gts.Node, *gts.Node) {
	var modifiers, body *gts.Node
	for index := range method.NamedChildCount() {
		child := method.NamedChild(index)
		switch child.Type(language) {
		case "modifiers":
			modifiers = child
		case "block":
			body = child
		}
	}
	return modifiers, body
}

func validateJavaInterfaceMethodModifiers(source []byte, modifiers *gts.Node) error {
	modifiersText := string(source[modifiers.StartByte():modifiers.EndByte()])
	if strings.Contains(modifiersText, "@") {
		return fmt.Errorf("java method modifier %q cannot be represented by an abstract interface signature", "annotation")
	}
	publicMethod := false
	for word := range strings.FieldsSeq(modifiersText) {
		if word == "public" {
			publicMethod = true
		}
		switch word {
		case "static", "final", "default", "native", "synchronized", "private", "protected":
			return fmt.Errorf("java method modifier %q cannot be represented by an abstract interface signature", word)
		}
	}
	if !publicMethod {
		return fmt.Errorf("java interface extraction selects public methods only")
	}
	return nil
}

// AppendJavaInterface appends one validated top-level interface to Java source.
func AppendJavaInterface(source, declaration []byte, name string) ([]byte, error) {
	if err := ValidateJavaIdentifier(name); err != nil {
		return nil, err
	}
	declarationType, names, err := declarationIdentity(declaration)
	if err != nil {
		return nil, fmt.Errorf("parse extracted Java interface: %w", err)
	}
	if declarationType != javaNodeInterfaceDeclaration || len(names) != 1 || names[0] != name {
		return nil, fmt.Errorf("extracted Java contract must be one interface named %q", name)
	}
	lang, root, err := parse(source)
	if err != nil {
		return nil, err
	}
	for index := range root.NamedChildCount() {
		child := root.NamedChild(index)
		if !isJavaTopLevelType(child.Type(lang)) {
			continue
		}
		_, existingNames, identityErr := declarationIdentity(source[int(child.StartByte()):int(child.EndByte())])
		if identityErr != nil {
			continue
		}
		if slices.Contains(existingNames, name) {
			return nil, fmt.Errorf("java type %q already exists in the selected file", name)
		}
	}
	base := bytes.TrimRight(source, " \t\r\n")
	updated := make([]byte, 0, len(base)+len(declaration)+3)
	updated = append(updated, base...)
	updated = append(updated, '\n', '\n')
	updated = append(updated, bytes.TrimSpace(declaration)...)
	updated = append(updated, '\n')
	if err := ValidateSource(updated); err != nil {
		return nil, fmt.Errorf("append Java interface produces invalid syntax: %w", err)
	}
	return updated, nil
}

// JavaPublicTopLevelTypeName returns the public top-level type declared in source.
func JavaPublicTopLevelTypeName(source []byte) (string, bool, error) {
	lang, root, err := parse(source)
	if err != nil {
		return "", false, err
	}
	for index := range root.NamedChildCount() {
		child := root.NamedChild(index)
		if !isJavaTopLevelType(child.Type(lang)) {
			continue
		}
		var modifiers *gts.Node
		for childIndex := range child.NamedChildCount() {
			member := child.NamedChild(childIndex)
			if member.Type(lang) == "modifiers" {
				modifiers = member
				break
			}
		}
		if modifiers == nil || !slices.Contains(strings.Fields(string(source[modifiers.StartByte():modifiers.EndByte()])), "public") {
			continue
		}
		_, names, identityErr := declarationIdentity(source[int(child.StartByte()):int(child.EndByte())])
		if identityErr != nil {
			return "", false, fmt.Errorf("resolve public Java top-level type name: %w", identityErr)
		}
		if len(names) != 1 {
			return "", false, fmt.Errorf("public Java top-level declaration has %d names", len(names))
		}
		return names[0], true, nil
	}
	return "", false, nil
}

// JavaTopLevelTypeNamePosition returns the syntax-tree position of a top-level type name.
func JavaTopLevelTypeNamePosition(source []byte, name string) (backend.Position, error) {
	if err := ValidateJavaIdentifier(name); err != nil {
		return backend.Position{}, err
	}
	lang, root, err := parse(source)
	if err != nil {
		return backend.Position{}, err
	}
	for index := range root.NamedChildCount() {
		declaration := root.NamedChild(index)
		if !isJavaTopLevelType(declaration.Type(lang)) {
			continue
		}
		for childIndex := range declaration.NamedChildCount() {
			child := declaration.NamedChild(childIndex)
			if child.Type(lang) == javaNodeIdentifier && string(source[child.StartByte():child.EndByte()]) == name {
				return backend.PositionFromByteOffset(source, int(child.StartByte())), nil
			}
		}
	}
	return backend.Position{}, fmt.Errorf("java top-level type %q has no declaration name", name)
}

func isJavaTopLevelType(kind string) bool {
	switch kind {
	case javaNodeClassDeclaration, javaNodeInterfaceDeclaration, javaNodeEnumDeclaration, javaNodeRecordDeclaration, "annotation_type_declaration":
		return true
	default:
		return false
	}
}
