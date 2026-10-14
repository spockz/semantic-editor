// Package javacst provides syntax-checked Java source transformations and construct selection.
// WHY: This file transforms parameters and call sites for parameter-object refactorings.
package javacst

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"

	gts "github.com/odvcencio/gotreesitter"
)

// JavaParameter describes one name and type in a Java method signature.
type JavaParameter struct {
	Name string
	Type string
}

// JavaParameterField maps one original parameter to a field in an object type.
type JavaParameterField struct {
	SourceIndex   int
	ParameterName string
	Name          string
	Type          string
}

// ParseJavaMethodParameters returns the ordered parameters from one Java method.
func ParseJavaMethodParameters(method []byte) ([]JavaParameter, error) {
	lang, root, err := parse(method)
	if err != nil {
		return nil, err
	}
	declaration := findFirstNode(root, lang, javaNodeMethodDeclaration)
	if declaration == nil {
		return nil, fmt.Errorf("java source does not contain a method declaration: %w", ErrInvalidSource)
	}
	parameters := findFirstNode(declaration, lang, "formal_parameters")
	if parameters == nil {
		return nil, fmt.Errorf("java method has no formal parameter list: %w", ErrInvalidSource)
	}
	result := make([]JavaParameter, 0, parameters.NamedChildCount())
	for index := range parameters.NamedChildCount() {
		parameter := parameters.NamedChild(index)
		if parameter.Type(lang) != "formal_parameter" {
			return nil, fmt.Errorf("unsupported Java parameter node %q", parameter.Type(lang))
		}
		var name, typeName string
		for childIndex := range parameter.NamedChildCount() {
			child := parameter.NamedChild(childIndex)
			switch child.Type(lang) {
			case javaNodeIdentifier:
				name = string(method[child.StartByte():child.EndByte()])
			case "type", "integral_type", "floating_point_type", "boolean_type", "void_type", "array_type", "generic_type", "scoped_type_identifier", "type_identifier":
				if typeName == "" {
					typeName = strings.TrimSpace(string(method[child.StartByte():child.EndByte()]))
				}
			default:
				return nil, fmt.Errorf("java parameter %d has unsupported syntax %q", index, child.Type(lang))
			}
		}
		if name == "" || typeName == "" {
			return nil, fmt.Errorf("java parameter %d has an unsupported declaration shape", index)
		}
		result = append(result, JavaParameter{Name: name, Type: typeName})
	}
	return result, nil
}

// JavaPackageName returns the package declared by Java source, or an empty name.
func JavaPackageName(source []byte) (string, error) {
	lang, root, err := parse(source)
	if err != nil {
		return "", err
	}
	packageNode := findFirstNode(root, lang, "package_declaration")
	if packageNode == nil {
		return "", nil
	}
	text := strings.TrimSpace(string(source[packageNode.StartByte():packageNode.EndByte()]))
	text = strings.TrimSpace(strings.TrimPrefix(text, "package"))
	text = strings.TrimSuffix(text, ";")
	return strings.TrimSpace(text), nil
}

// ReplaceJavaPackageName changes or removes a Java package declaration.
func ReplaceJavaPackageName(source []byte, packageName string) ([]byte, error) {
	lang, root, err := parse(source)
	if err != nil {
		return nil, err
	}
	packageNode := findFirstNode(root, lang, "package_declaration")
	updated := bytes.Clone(source)
	switch {
	case packageNode == nil:
		if packageName != "" {
			prefix := []byte("package " + packageName + ";\n\n")
			updated = append(prefix, updated...)
		}
	case packageName == "":
		updated = append(bytes.Clone(source[:packageNode.StartByte()]), source[packageNode.EndByte():]...)
	default:
		declaration := []byte("package " + packageName + ";")
		updated = append(bytes.Clone(source[:packageNode.StartByte()]), append(declaration, source[packageNode.EndByte():]...)...)
	}
	resolved, err := JavaPackageName(updated)
	if err != nil {
		return nil, fmt.Errorf("validate updated Java package declaration: %w", err)
	}
	if resolved != packageName {
		return nil, fmt.Errorf("updated Java package is %q, want %q: %w", resolved, packageName, ErrInvalidSource)
	}
	return updated, nil
}

// JavaMethodExposesOutsidePackage reports whether a method is public or protected.
func JavaMethodExposesOutsidePackage(method []byte) (bool, error) {
	lang, root, err := parse(method)
	if err != nil {
		return false, err
	}
	declaration := findFirstNode(root, lang, javaNodeMethodDeclaration)
	if declaration == nil {
		return false, fmt.Errorf("java source does not contain a method declaration: %w", ErrInvalidSource)
	}
	for index := range declaration.NamedChildCount() {
		child := declaration.NamedChild(index)
		if child.Type(lang) != "modifiers" {
			continue
		}
		for modifier := range strings.FieldsSeq(string(method[child.StartByte():child.EndByte()])) {
			if modifier == "public" || modifier == "protected" {
				return true, nil
			}
		}
	}
	return false, nil
}

// InsertJavaParameterObject inserts a value type with fields for the selected parameters.
func InsertJavaParameterObject(source []byte, name string, fields []JavaParameterField) ([]byte, error) {
	if err := ValidateJavaIdentifier(name); err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("java parameter object requires at least one field")
	}
	lang, root, err := parse(source)
	if err != nil {
		return nil, err
	}
	if javaTypeExists(root, lang, source, name) {
		return nil, fmt.Errorf("java type %q already exists", name)
	}
	seen := make(map[string]bool, len(fields))
	var declaration strings.Builder
	fmt.Fprintf(&declaration, "\nfinal class %s {\n", name)
	for _, field := range fields {
		if err := ValidateJavaIdentifier(field.Name); err != nil {
			return nil, fmt.Errorf("invalid Java parameter-object field %q: %w", field.Name, err)
		}
		if strings.TrimSpace(field.Type) == "" || seen[field.Name] {
			return nil, fmt.Errorf("invalid or repeated Java parameter-object field %q", field.Name)
		}
		seen[field.Name] = true
		fmt.Fprintf(&declaration, "  private final %s %s;\n", field.Type, field.Name)
	}
	fmt.Fprintf(&declaration, "\n  %s(", name)
	for index, field := range fields {
		if index > 0 {
			declaration.WriteString(", ")
		}
		fmt.Fprintf(&declaration, "%s %s", field.Type, field.Name)
	}
	declaration.WriteString(") {\n")
	for _, field := range fields {
		fmt.Fprintf(&declaration, "    this.%s = %s;\n", field.Name, field.Name)
	}
	declaration.WriteString("  }\n")
	for _, field := range fields {
		fmt.Fprintf(&declaration, "\n  %s %s() {\n    return this.%s;\n  }\n", field.Type, field.Name, field.Name)
	}
	declaration.WriteString("}\n\n")
	updated := bytes.Clone(source)
	if len(updated) > 0 && updated[len(updated)-1] != '\n' {
		updated = append(updated, '\n')
	}
	updated = append(updated, []byte(declaration.String())...)
	if err := ValidateSource(updated); err != nil {
		return nil, fmt.Errorf("java parameter-object declaration is invalid: %w", err)
	}
	return updated, nil
}

// RewriteJavaParameterReferences replaces bound method parameters with value-object accessors.
func RewriteJavaParameterReferences(method []byte, objectName string, fields []JavaParameterField) ([]byte, error) {
	if err := ValidateJavaIdentifier(objectName); err != nil {
		return nil, err
	}
	fieldBySource, err := javaParameterObjectFieldNames(fields)
	if err != nil {
		return nil, err
	}
	lang, root, err := parse(method)
	if err != nil {
		return nil, err
	}
	declaration := findFirstNode(root, lang, javaNodeMethodDeclaration)
	if declaration == nil {
		return nil, fmt.Errorf("java source does not contain a method declaration: %w", ErrInvalidSource)
	}
	body := findFirstNode(declaration, lang, "block")
	if body == nil {
		return nil, fmt.Errorf("java method has no body")
	}
	spans, err := javaParameterReferenceSpans(method, lang, body, declaration, objectName, fieldBySource)
	if err != nil {
		return nil, err
	}
	if len(spans) == 0 {
		return bytes.Clone(method), nil
	}
	return applySourceSpans(method, spans), nil
}

func javaParameterObjectFieldNames(fields []JavaParameterField) (map[string]string, error) {
	fieldBySource := make(map[string]string, len(fields))
	for index, field := range fields {
		if field.SourceIndex != index || field.ParameterName == "" || field.Name == "" {
			return nil, fmt.Errorf("invalid Java parameter-object field mapping")
		}
		if err := ValidateJavaIdentifier(field.Name); err != nil {
			return nil, err
		}
		if _, exists := fieldBySource[field.ParameterName]; exists {
			return nil, fmt.Errorf("java parameter-object mapping repeats parameter %q", field.ParameterName)
		}
		fieldBySource[field.ParameterName] = field.Name
	}
	return fieldBySource, nil
}

func javaParameterReferenceSpans(source []byte, lang *gts.Language, body, declaration *gts.Node, objectName string, fieldNames map[string]string) ([]sourceSpan, error) {
	var spans []sourceSpan
	var walk func(*gts.Node, *gts.Node) error
	walk = func(node, parent *gts.Node) error {
		if node == nil {
			return nil
		}
		if node.Type(lang) == javaNodeIdentifier {
			name := string(source[node.StartByte():node.EndByte()])
			if field, exists := fieldNames[name]; exists {
				if isJavaShadowDeclaration(node, parent, lang) {
					return fmt.Errorf("java parameter %q has an unsupported shadowing declaration", name)
				}
				if isJavaIdentifierUse(node, parent, lang) {
					spans = append(spans, sourceSpan{start: int(node.StartByte()), end: int(node.EndByte()), text: objectName + "." + field + "()"})
				}
			}
		}
		for index := range node.NamedChildCount() {
			if err := walk(node.NamedChild(index), node); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(body, declaration); err != nil {
		return nil, err
	}
	return spans, nil
}

type sourceSpan struct {
	start int
	end   int
	text  string
}

func applySourceSpans(source []byte, spans []sourceSpan) []byte {
	result := bytes.Clone(source)
	for _, span := range slices.Backward(spans) {

		result = append(result[:span.start], append([]byte(span.text), result[span.end:]...)...)
	}
	return result
}

// JavaMethodCallArgumentsAt returns arguments for the invocation identified by a JDT LS range.
func JavaMethodCallArgumentsAt(source []byte, reference backend.Range) ([]string, error) {
	start, err := ByteOffsetFromPosition(source, reference.Start)
	if err != nil {
		return nil, err
	}
	end, err := ByteOffsetFromPosition(source, reference.End)
	if err != nil || end <= start {
		return nil, fmt.Errorf("java reference range is invalid")
	}
	lang, root, err := parse(source)
	if err != nil {
		return nil, err
	}
	var arguments []string
	var found bool
	var walk func(*gts.Node)
	walk = func(node *gts.Node) {
		if node == nil || found {
			return
		}
		if node.Type(lang) == javaNodeMethodInvocation {
			var matched bool
			arguments, matched, err = javaMethodCallArguments(source, lang, node, start, end)
			if matched || err != nil {
				found = matched
				return
			}
		}
		for index := range node.NamedChildCount() {
			walk(node.NamedChild(index))
		}
	}
	walk(root)
	if !found {
		return nil, fmt.Errorf("java reference [%d,%d) does not identify a method invocation", start, end)
	}
	if err != nil {
		return nil, err
	}
	return arguments, nil
}

func javaMethodCallArguments(source []byte, lang *gts.Language, invocation *gts.Node, start, end int) ([]string, bool, error) {
	name, argumentList := javaInvocationNameAndArguments(lang, invocation, start, end)
	invocationRange := int(invocation.StartByte()) == start && int(invocation.EndByte()) == end
	nameThroughInvocationRange := int(invocation.EndByte()) == end && javaHasIdentifierAtStart(invocation, lang, start)
	if argumentList == nil || !name && !invocationRange && !nameThroughInvocationRange {
		return nil, false, nil
	}
	if hasJavaComment(argumentList, lang) {
		return nil, true, fmt.Errorf("java call arguments containing comments cannot be rewritten safely")
	}
	arguments := make([]string, 0, argumentList.NamedChildCount())
	for index := range argumentList.NamedChildCount() {
		child := argumentList.NamedChild(index)
		arguments = append(arguments, strings.TrimSpace(string(source[child.StartByte():child.EndByte()])))
	}
	return arguments, true, nil
}

func javaInvocationNameAndArguments(lang *gts.Language, invocation *gts.Node, start, end int) (bool, *gts.Node) {
	var matchesName bool
	var argumentList *gts.Node
	for index := range invocation.NamedChildCount() {
		child := invocation.NamedChild(index)
		if child.Type(lang) == javaNodeIdentifier && int(child.StartByte()) == start && int(child.EndByte()) == end {
			matchesName = true
		}
		if child.Type(lang) == "argument_list" {
			argumentList = child
		}
	}
	return matchesName, argumentList
}

func javaHasIdentifierAtStart(node *gts.Node, lang *gts.Language, start int) bool {
	if node.Type(lang) == javaNodeIdentifier && int(node.StartByte()) == start {
		return true
	}
	for index := range node.NamedChildCount() {
		if javaHasIdentifierAtStart(node.NamedChild(index), lang, start) {
			return true
		}
	}
	return false
}

func hasJavaComment(node *gts.Node, language *gts.Language) bool {
	if node == nil {
		return false
	}
	if strings.Contains(node.Type(language), "comment") {
		return true
	}
	for index := range node.NamedChildCount() {
		if hasJavaComment(node.NamedChild(index), language) {
			return true
		}
	}
	return false
}

// ReplaceJavaMethodCallsWithParameterObject rewrites calls to construct one parameter object.
func ReplaceJavaMethodCallsWithParameterObject(source []byte, methodName, objectType string, callArguments [][]string) ([]byte, error) {
	lang, root, err := parse(source)
	if err != nil {
		return nil, err
	}
	placeholder := compact("new " + objectType + "()")
	var argumentLists []*gts.Node
	var walk func(*gts.Node)
	walk = func(node *gts.Node) {
		if node == nil {
			return
		}
		if node.Type(lang) == javaNodeMethodInvocation {
			nameMatches := false
			var argumentList *gts.Node
			for index := range node.NamedChildCount() {
				child := node.NamedChild(index)
				if child.Type(lang) == javaNodeIdentifier && string(source[child.StartByte():child.EndByte()]) == methodName {
					nameMatches = true
				}
				if child.Type(lang) == "argument_list" {
					argumentList = child
				}
			}
			if nameMatches && argumentList != nil && argumentList.NamedChildCount() == 1 &&
				compact(string(source[argumentList.NamedChild(0).StartByte():argumentList.NamedChild(0).EndByte()])) == placeholder {
				argumentLists = append(argumentLists, argumentList)
			}
		}
		for index := range node.NamedChildCount() {
			walk(node.NamedChild(index))
		}
	}
	walk(root)
	if len(argumentLists) != len(callArguments) {
		return nil, fmt.Errorf("java parameter-object call-site count mismatch: found %d placeholder calls, have %d resolved references", len(argumentLists), len(callArguments))
	}
	spans := make([]sourceSpan, 0, len(argumentLists))
	for index, argumentList := range argumentLists {
		if index >= len(callArguments) {
			return nil, fmt.Errorf("java parameter-object call-site count mismatch: missing arguments for reference %d", index)
		}
		arguments := strings.Join(callArguments[index], ", ") // #nosec G602 -- lengths are checked equal above and the index is bounded by argumentLists.
		spans = append(spans, sourceSpan{start: int(argumentList.StartByte()), end: int(argumentList.EndByte()), text: "(new " + objectType + "(" + arguments + "))"})
	}
	return applySourceSpans(source, spans), nil
}

func findFirstNode(node *gts.Node, language *gts.Language, kind string) *gts.Node {
	if node == nil {
		return nil
	}
	if node.Type(language) == kind {
		return node
	}
	for index := range node.NamedChildCount() {
		if found := findFirstNode(node.NamedChild(index), language, kind); found != nil {
			return found
		}
	}
	return nil
}

func javaTypeExists(node *gts.Node, language *gts.Language, source []byte, name string) bool {
	if node == nil {
		return false
	}
	switch node.Type(language) {
	case javaNodeClassDeclaration, javaNodeInterfaceDeclaration, javaNodeEnumDeclaration, javaNodeRecordDeclaration:
		for index := range node.NamedChildCount() {
			child := node.NamedChild(index)
			if child.Type(language) == javaNodeIdentifier && string(source[child.StartByte():child.EndByte()]) == name {
				return true
			}
		}
	}
	for index := range node.NamedChildCount() {
		if javaTypeExists(node.NamedChild(index), language, source, name) {
			return true
		}
	}
	return false
}

func isJavaIdentifierUse(node, parent *gts.Node, language *gts.Language) bool {
	if parent == nil {
		return false
	}
	last := parent.NamedChildCount() - 1
	for index := range parent.NamedChildCount() {
		if parent.NamedChild(index) == node {
			if (parent.Type(language) == javaNodeMethodInvocation || parent.Type(language) == "field_access" || parent.Type(language) == "method_reference") && index == last {
				return false
			}
			break
		}
	}
	switch parent.Type(language) {
	case javaNodeVariableDeclarator, "formal_parameter", "catch_formal_parameter", javaNodeMethodDeclaration, javaNodeConstructorDeclaration, javaNodeClassDeclaration, javaNodeInterfaceDeclaration, javaNodeEnumDeclaration, javaNodeRecordDeclaration, "type_parameter", "labeled_statement", "break_statement", "continue_statement":
		return false
	}
	return true
}

func isJavaShadowDeclaration(node, parent *gts.Node, language *gts.Language) bool {
	if node == nil || parent == nil {
		return false
	}
	if parent.Type(language) == javaNodeVariableDeclarator || parent.Type(language) == "formal_parameter" ||
		parent.Type(language) == "catch_formal_parameter" || parent.Type(language) == "type_parameter" ||
		parent.Type(language) == "enhanced_for_statement" {
		return true
	}
	if parent.Type(language) == "lambda_expression" || parent.Type(language) == "inferred_parameters" {
		return true
	}
	return false
}
