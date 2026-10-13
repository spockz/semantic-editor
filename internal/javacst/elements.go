// Package javacst provides syntax-checked Java source transformations and construct selection.
// WHY: This file appends initializer elements without rewriting unrelated source.
package javacst

import (
	"bytes"
	"fmt"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
)

type javaElementContainer struct {
	node      *gts.Node
	arguments *gts.Node
	form      string
}

// AppendJavaElement appends one initializer element matching the supplied discriminator.
func AppendJavaElement(declaration []byte, discriminator string, addition []byte) ([]byte, error) {
	if strings.TrimSpace(discriminator) == "" || len(bytes.TrimSpace(addition)) == 0 {
		return nil, fmt.Errorf("java element insertion requires a discriminator and source")
	}
	prefix := []byte("class __Semedit {\n")
	suffix := []byte("\n}")
	wrapped := make([]byte, 0, len(prefix)+len(declaration)+len(suffix))
	wrapped = append(wrapped, prefix...)
	wrapped = append(wrapped, declaration...)
	wrapped = append(wrapped, suffix...)
	lang, root, err := parse(wrapped)
	if err != nil {
		return nil, err
	}
	member, err := soleWrappedMember(lang, root)
	if err != nil {
		return nil, err
	}
	containers := findJavaElementContainers(lang, wrapped, member, discriminator)
	if len(containers) == 0 {
		return nil, fmt.Errorf("java element target %q is not a supported Map/List/Set factory or array initializer: %w", discriminator, ErrNotFound)
	}
	if len(containers) != 1 {
		return nil, fmt.Errorf("java element target %q resolves to %d initializers: %w", discriminator, len(containers), ErrAmbiguous)
	}
	selected := containers[0]
	if err := validateJavaElementAddition(selected.form, addition); err != nil {
		return nil, err
	}
	if selected.arguments == nil || selected.arguments.EndByte() <= selected.arguments.StartByte()+1 {
		return nil, fmt.Errorf("java element target has no appendable delimiter: %w", ErrInvalidSource)
	}
	childCount := selected.arguments.NamedChildCount()
	insertAt := int(selected.arguments.EndByte()) - 1
	relative := insertAt - len(prefix)
	if relative < 0 || relative > len(declaration) {
		return nil, fmt.Errorf("java element insertion offset is outside the selected declaration: %w", ErrInvalidSource)
	}
	insert := bytes.Clone(addition)
	if childCount > 0 {
		insert = append([]byte(", "), insert...)
	}
	updated := make([]byte, 0, len(declaration)+len(insert))
	updated = append(updated, declaration[:relative]...)
	updated = append(updated, insert...)
	updated = append(updated, declaration[relative:]...)
	check := append(append(bytes.Clone(prefix), updated...), suffix...)
	if err := ValidateSource(check); err != nil {
		return nil, fmt.Errorf("java element insertion produced invalid source: %w", err)
	}
	return updated, nil
}

func findJavaElementContainers(lang *gts.Language, source []byte, root *gts.Node, discriminator string) []javaElementContainer {
	needle := compact(discriminator)
	containers := make([]javaElementContainer, 0, 1)
	var visit func(*gts.Node)
	visit = func(node *gts.Node) {
		if node == nil {
			return
		}
		if !strings.Contains(compact(string(source[node.StartByte():node.EndByte()])), needle) {
			visitJavaNamedChildren(node, visit)
			return
		}
		if node.Type(lang) == "array_initializer" {
			containers = append(containers, javaElementContainer{node: node, arguments: node, form: "array"})
		} else if node.Type(lang) == javaNodeMethodInvocation {
			form := javaCollectionFactory(lang, source, node)
			if form != "" {
				containers = append(containers, javaElementContainer{node: node, arguments: node.ChildByFieldName("arguments", lang), form: form})
			}
		}
		visitJavaNamedChildren(node, visit)
	}
	visit(root)
	return containers
}

func visitJavaNamedChildren(node *gts.Node, visit func(*gts.Node)) {
	for index := range node.NamedChildCount() {
		visit(node.NamedChild(index))
	}
}

func javaCollectionFactory(lang *gts.Language, source []byte, invocation *gts.Node) string {
	name := invocation.ChildByFieldName("name", lang)
	object := invocation.ChildByFieldName("object", lang)
	arguments := invocation.ChildByFieldName("arguments", lang)
	if name == nil || object == nil || arguments == nil {
		return ""
	}
	method := string(source[name.StartByte():name.EndByte()])
	qualifier := string(source[object.StartByte():object.EndByte()])
	qualifier = qualifier[strings.LastIndex(qualifier, ".")+1:]
	switch qualifier {
	case "Map":
		if method == "of" || method == "ofEntries" {
			return "map." + method
		}
	case "List", "Set":
		if method == "of" {
			return strings.ToLower(qualifier) + ".of"
		}
	}
	return ""
}

func validateJavaElementAddition(form string, addition []byte) error {
	var validation []byte
	var want int
	switch form {
	case "array":
		validation = append([]byte("class __Semedit { Object[] __value = new Object[]{"), addition...)
		validation = append(validation, []byte("}; }")...)
		want = 1
	case "map.of":
		validation = append([]byte("class __Semedit { Object __value = Map.of("), addition...)
		validation = append(validation, []byte("); }")...)
		want = 2
	case "map.ofEntries":
		validation = append([]byte("class __Semedit { Object __value = Map.ofEntries("), addition...)
		validation = append(validation, []byte("); }")...)
		want = 1
	case "list.of", "set.of":
		factory := "List"
		if form == "set.of" {
			factory = "Set"
		}
		validation = append([]byte("class __Semedit { Object __value = "+factory+".of("), addition...)
		validation = append(validation, []byte("); }")...)
		want = 1
	default:
		return fmt.Errorf("unsupported Java element container %q", form)
	}
	lang, root, err := parse(validation)
	if err != nil {
		return fmt.Errorf("java element source is not a valid expression list: %w", err)
	}
	var args *gts.Node
	var array *gts.Node
	var find func(*gts.Node)
	find = func(node *gts.Node) {
		if node == nil || args != nil || array != nil {
			return
		}
		if form == "array" && node.Type(lang) == "array_initializer" {
			array = node
			return
		}
		if form != "array" && node.Type(lang) == "argument_list" {
			args = node
			return
		}
		for index := range node.NamedChildCount() {
			find(node.NamedChild(index))
		}
	}
	find(root)
	count := -1
	if args != nil {
		count = args.NamedChildCount()
	} else if array != nil {
		count = array.NamedChildCount()
	}
	if count != want {
		return fmt.Errorf("java %s insertion requires %d expression(s), got %d", form, want, count)
	}
	return nil
}
