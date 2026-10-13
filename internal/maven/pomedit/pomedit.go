// Package pomedit plans safe dependency changes across Maven POM files.
// WHY: This file preserves unrelated XML while editing dependency blocks.
package pomedit

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Maven dependency planning errors distinguish invalid input from unsafe parent traversal.
var (
	ErrInvalidCoordinate = errors.New("invalid Maven dependency coordinate")
	ErrInvalidPOM        = errors.New("invalid Maven POM")
	ErrNoVersion         = errors.New("maven dependency version is required")
	ErrParentOutsideRoot = errors.New("maven parent POM escapes workspace root")
)

// Dependency is a parsed Maven dependency coordinate.
type Dependency struct {
	GroupID    string
	ArtifactID string
	Version    string
	Scope      string
	Classifier string
}

// FileEdit contains the replacement bytes for one POM file.
type FileEdit struct {
	Path string
	Data []byte
}

type pomModel struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
	Parent     struct {
		RelativePath *string `xml:"relativePath"`
	} `xml:"parent"`
}

type xmlNode struct {
	Name        string
	Start       int64
	OpenEnd     int64
	CloseStart  int64
	End         int64
	SelfClosing bool
	Text        strings.Builder
	Parent      *xmlNode
	Children    []*xmlNode
}

// ParseCoordinate validates and splits a Maven dependency coordinate.
func ParseCoordinate(raw string) (Dependency, error) {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	if len(parts) < 2 || len(parts) > 5 {
		return Dependency{}, fmt.Errorf("%w: expected groupId:artifactId[:version][:scope][:classifier]", ErrInvalidCoordinate)
	}
	dependency := Dependency{GroupID: parts[0], ArtifactID: parts[1]}
	for _, part := range parts {
		if strings.TrimSpace(part) != part {
			return Dependency{}, fmt.Errorf("%w: whitespace-padded component", ErrInvalidCoordinate)
		}
	}
	if len(parts) >= 3 {
		dependency.Version = parts[2]
	}
	if len(parts) >= 4 {
		dependency.Scope = parts[3]
	}
	if len(parts) == 5 {
		dependency.Classifier = parts[4]
	}
	if !validCoordinatePart(dependency.GroupID) || !validCoordinatePart(dependency.ArtifactID) {
		return Dependency{}, fmt.Errorf("%w: invalid groupId or artifactId", ErrInvalidCoordinate)
	}
	if strings.ContainsAny(dependency.Version, " \t\r\n:") || (dependency.Classifier != "" && !validCoordinatePart(dependency.Classifier)) {
		return Dependency{}, fmt.Errorf("%w: invalid version or classifier", ErrInvalidCoordinate)
	}
	if dependency.Scope != "" && !oneOf(dependency.Scope, "compile", "provided", "runtime", "test", "system", "import") {
		return Dependency{}, fmt.Errorf("%w: unsupported dependency scope %q", ErrInvalidCoordinate, dependency.Scope)
	}
	return dependency, nil
}

// Plan returns the atomic POM edits needed to add a dependency to the target module.
func Plan(root, target string, dependency Dependency) ([]FileEdit, error) {
	root, target, err := resolvePOMTarget(root, target)
	if err != nil {
		return nil, err
	}
	chain, err := parentChain(root, target)
	if err != nil {
		return nil, err
	}
	targetData, targetTree, err := readPOM(target)
	if err != nil {
		return nil, err
	}
	if dependencyIn(targetTree, dependency, "dependencies") {
		return nil, nil
	}
	return planDependencyEdits(chain, target, targetData, targetTree, dependency)
}

func resolvePOMTarget(root, target string) (string, string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", "", fmt.Errorf("resolve Maven workspace root: %w", err)
	}
	if resolved, evalErr := filepath.EvalSymlinks(root); evalErr == nil {
		root = resolved
	}
	root = filepath.Clean(root)
	if target == "" {
		target = filepath.Join(root, "pom.xml")
	} else if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	if info, statErr := os.Stat(target); statErr == nil && info.IsDir() {
		target = filepath.Join(target, "pom.xml")
	}
	target, err = canonicalExisting(target)
	if err != nil {
		return "", "", fmt.Errorf("resolve target Maven POM: %w", err)
	}
	if !within(root, target) || filepath.Base(target) != "pom.xml" {
		return "", "", fmt.Errorf("target POM %q is outside Maven root or is not pom.xml", target)
	}
	rootPOM, err := canonicalExisting(filepath.Join(root, "pom.xml"))
	if err != nil {
		return "", "", fmt.Errorf("read Maven root POM %q: %w", rootPOM, err)
	}
	if !within(root, rootPOM) {
		return "", "", ErrParentOutsideRoot
	}
	return root, target, nil
}

func planDependencyEdits(chain []string, target string, targetData []byte, targetTree *xmlNode, dependency Dependency) ([]FileEdit, error) {
	managedAt, err := findManagedDependency(chain, dependency)
	if err != nil {
		return nil, err
	}
	if managedAt != "" {
		return addManagedTargetDependency(target, targetData, targetTree, dependency)
	}
	managementAt, err := findDependencyManagementPOM(chain)
	if err != nil {
		return nil, err
	}
	if managementAt != "" {
		return addManagedDependency(managementAt, target, targetData, targetTree, dependency)
	}
	if dependency.Version == "" {
		return nil, ErrNoVersion
	}
	updated, err := addDependency(targetData, targetTree, dependencyTarget, dependency)
	if err != nil {
		return nil, err
	}
	return []FileEdit{{Path: target, Data: updated}}, nil
}

func findManagedDependency(chain []string, dependency Dependency) (string, error) {
	for _, pomPath := range chain {
		_, tree, err := readPOM(pomPath)
		if err != nil {
			return "", err
		}
		if dependencyIn(tree, dependency, "dependencyManagement") {
			return pomPath, nil
		}
	}
	return "", nil
}

func findDependencyManagementPOM(chain []string) (string, error) {
	for _, pomPath := range chain {
		_, tree, err := readPOM(pomPath)
		if err != nil {
			return "", err
		}
		if directChild(child(tree, "project"), "dependencyManagement") != nil {
			return pomPath, nil
		}
	}
	return "", nil
}

func addManagedTargetDependency(target string, targetData []byte, targetTree *xmlNode, dependency Dependency) ([]FileEdit, error) {
	dependency.Version = ""
	updated, err := addDependency(targetData, targetTree, dependencyTarget, dependency)
	if err != nil {
		return nil, err
	}
	return []FileEdit{{Path: target, Data: updated}}, nil
}

func addManagedDependency(managementAt, target string, targetData []byte, targetTree *xmlNode, dependency Dependency) ([]FileEdit, error) {
	if dependency.Version == "" {
		return nil, ErrNoVersion
	}
	managementData, managementTree, err := readPOM(managementAt)
	if err != nil {
		return nil, err
	}
	managedData, err := addDependency(managementData, managementTree, managementTarget, dependency)
	if err != nil {
		return nil, err
	}
	targetDependency := dependency
	targetDependency.Version = ""
	targetUpdated, err := addDependency(targetData, targetTree, dependencyTarget, targetDependency)
	if err != nil {
		return nil, err
	}
	if managementAt == target {
		combinedTree, err := parseXML(targetUpdated)
		if err != nil {
			return nil, err
		}
		combined, err := addDependency(targetUpdated, combinedTree, managementTarget, dependency)
		if err != nil {
			return nil, err
		}
		return []FileEdit{{Path: target, Data: combined}}, nil
	}
	return []FileEdit{{Path: managementAt, Data: managedData}, {Path: target, Data: targetUpdated}}, nil
}

const (
	dependencyTarget = "dependencies"
	managementTarget = "dependencyManagement"
)

func parentChain(root, target string) ([]string, error) {
	chain := []string{target}
	seen := map[string]bool{target: true}
	current := target
parentLoop:
	for current != filepath.Join(root, "pom.xml") {
		_, _, model, err := readPOMModel(current)
		if err != nil {
			return nil, err
		}
		switch {
		case model.Parent.RelativePath == nil:
			current = filepath.Join(filepath.Dir(current), "..", "pom.xml")
		case strings.TrimSpace(*model.Parent.RelativePath) == "":
			break parentLoop
		default:
			parent := *model.Parent.RelativePath
			if !filepath.IsAbs(parent) {
				parent = filepath.Join(filepath.Dir(current), parent)
			}
			if filepath.Ext(parent) != ".xml" {
				parent = filepath.Join(parent, "pom.xml")
			}
			current = parent
		}
		current, err = canonicalExisting(current)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				break
			}
			return nil, fmt.Errorf("resolve Maven parent of %q: %w", chain[len(chain)-1], err)
		}
		if !within(root, current) {
			return nil, fmt.Errorf("parent POM %q: %w", current, ErrParentOutsideRoot)
		}
		if seen[current] {
			return nil, fmt.Errorf("maven parent cycle at %q", current)
		}
		seen[current] = true
		chain = append(chain, current)
	}
	return chain, nil
}

func readPOM(path string) ([]byte, *xmlNode, error) {
	data, tree, _, err := readPOMModel(path)
	if err != nil {
		return nil, nil, err
	}
	if child(tree, "project") == nil {
		return nil, nil, fmt.Errorf("POM %q has no project root: %w", path, ErrInvalidPOM)
	}
	return data, tree, nil
}

func readPOMModel(path string) ([]byte, *xmlNode, pomModel, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- callers validate each POM path against the selected workspace root.
	if err != nil {
		return nil, nil, pomModel{}, fmt.Errorf("read Maven POM %q: %w", path, err)
	}
	tree, err := parseXML(data)
	if err != nil {
		return nil, nil, pomModel{}, fmt.Errorf("parse Maven POM %q: %w", path, err)
	}
	var model pomModel
	if err := xml.Unmarshal(data, &model); err != nil {
		return nil, nil, pomModel{}, fmt.Errorf("decode Maven POM %q: %w", path, err)
	}
	return data, tree, model, nil
}

func parseXML(data []byte) (*xmlNode, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var root *xmlNode
	stack := make([]*xmlNode, 0)
	for {
		start := decoder.InputOffset()
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidPOM, err)
		}
		end := decoder.InputOffset()
		switch value := token.(type) {
		case xml.StartElement:
			node := &xmlNode{Name: value.Name.Local, Start: start, OpenEnd: end, SelfClosing: bytes.HasSuffix(bytes.TrimSpace(data[start:end]), []byte("/>"))}
			switch {
			case len(stack) > 0:
				node.Parent = stack[len(stack)-1]
				node.Parent.Children = append(node.Parent.Children, node)
			case root == nil:
				root = node
			default:
				return nil, fmt.Errorf("%w: multiple root elements", ErrInvalidPOM)
			}
			stack = append(stack, node)
		case xml.EndElement:
			if len(stack) == 0 || stack[len(stack)-1].Name != value.Name.Local {
				return nil, fmt.Errorf("%w: unexpected closing element %s", ErrInvalidPOM, value.Name.Local)
			}
			node := stack[len(stack)-1]
			node.CloseStart, node.End = start, end
			if node.SelfClosing {
				node.CloseStart, node.End = node.OpenEnd, node.OpenEnd
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].Text.Write([]byte(value))
			}
		}
	}
	if root == nil || len(stack) != 0 {
		return nil, fmt.Errorf("%w: incomplete XML document", ErrInvalidPOM)
	}
	return root, nil
}

func addDependency(data []byte, tree *xmlNode, target string, dependency Dependency) ([]byte, error) {
	project := child(tree, "project")
	containerParent := project
	if target == managementTarget {
		containerParent = directChild(project, "dependencyManagement")
		if containerParent == nil {
			return nil, fmt.Errorf("POM has no dependencyManagement section")
		}
	}
	dependencies := directChild(containerParent, "dependencies")
	unit := indentUnit(data)
	parentIndent := lineIndent(data, containerParent.Start)
	if dependencies == nil {
		dependencyIndent := parentIndent + unit + unit
		content := renderDependency(dependency, dependencyIndent, unit, lineEnding(data))
		block := renderContainer("dependencies", content, parentIndent+unit, lineEnding(data))
		if containerParent.SelfClosing {
			return expandSelfClosing(data, containerParent, block, parentIndent, lineEnding(data)), nil
		}
		return insertBeforeClose(data, containerParent, block, parentIndent, lineEnding(data)), nil
	}
	if dependencies.SelfClosing {
		indent := lineIndent(data, dependencies.Start)
		childIndent := indent + unit
		content := renderDependency(dependency, childIndent, unit, lineEnding(data))
		return expandSelfClosing(data, dependencies, content, indent, lineEnding(data)), nil
	}
	closeIndent := lineIndent(data, dependencies.CloseStart)
	childIndent := closeIndent + unit
	content := renderDependency(dependency, childIndent, unit, lineEnding(data))
	return insertBeforeClose(data, dependencies, content, closeIndent, lineEnding(data)), nil
}

func renderDependency(dependency Dependency, indent, unit, eol string) string {
	fields := []struct{ name, value string }{{"groupId", dependency.GroupID}, {"artifactId", dependency.ArtifactID}}
	if dependency.Version != "" {
		fields = append(fields, struct{ name, value string }{"version", dependency.Version})
	}
	if dependency.Scope != "" {
		fields = append(fields, struct{ name, value string }{"scope", dependency.Scope})
	}
	if dependency.Classifier != "" {
		fields = append(fields, struct{ name, value string }{"classifier", dependency.Classifier})
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "%s<dependency>%s", indent, eol)
	for _, field := range fields {
		fmt.Fprintf(&builder, "%s%s<%s>%s</%s>%s", indent+unit, "", field.name, html.EscapeString(field.value), field.name, eol)
	}
	fmt.Fprintf(&builder, "%s</dependency>%s", indent, eol)
	return builder.String()
}

func renderContainer(name, content, indent, eol string) string {
	return indent + "<" + name + ">" + eol + content + indent + "</" + name + ">" + eol
}

func insertBeforeClose(data []byte, node *xmlNode, content, closeIndent, eol string) []byte {
	insertAt := lineStart(data, node.CloseStart)
	if bytes.TrimSpace(data[insertAt:node.CloseStart]) != nil && len(bytes.TrimSpace(data[insertAt:node.CloseStart])) != 0 {
		content = eol + content + closeIndent
		insertAt = int(node.CloseStart)
	}
	result := make([]byte, 0, len(data)+len(content))
	result = append(result, data[:insertAt]...)
	result = append(result, content...)
	result = append(result, data[insertAt:]...)
	return result
}

func expandSelfClosing(data []byte, node *xmlNode, content, indent, eol string) []byte {
	open := append([]byte(nil), data[node.Start:node.OpenEnd]...)
	before, _, ok := bytes.CutLast(open, []byte("/>"))
	if !ok {
		return data
	}
	opening := append([]byte(nil), before...)
	opening = append(opening, '>')
	closing := []byte(indent + "</" + node.Name + ">")
	replacement := opening
	replacement = append(replacement, eol...)
	replacement = append(replacement, []byte(content)...)
	replacement = append(replacement, closing...)
	return append(append(append([]byte(nil), data[:node.Start]...), replacement...), data[node.OpenEnd:]...)
}

func dependencyIn(tree *xmlNode, dependency Dependency, target string) bool {
	project := child(tree, "project")
	if target == managementTarget {
		project = directChild(project, "dependencyManagement")
	}
	container := directChild(project, "dependencies")
	if container == nil {
		return false
	}
	for _, entry := range container.Children {
		if entry.Name != "dependency" {
			continue
		}
		if textChild(entry, "groupId") == dependency.GroupID && textChild(entry, "artifactId") == dependency.ArtifactID && textChild(entry, "classifier") == dependency.Classifier {
			return true
		}
	}
	return false
}

func child(node *xmlNode, name string) *xmlNode {
	if node == nil {
		return nil
	}
	if node.Name == name {
		return node
	}
	for _, candidate := range node.Children {
		if candidate.Name == name {
			return candidate
		}
	}
	return nil
}

func directChild(node *xmlNode, name string) *xmlNode {
	if node == nil {
		return nil
	}
	for _, candidate := range node.Children {
		if candidate.Name == name {
			return candidate
		}
	}
	return nil
}

func textChild(node *xmlNode, name string) string {
	return strings.TrimSpace(child(node, name).textString())
}

func (node *xmlNode) textString() string {
	if node == nil {
		return ""
	}
	return node.Text.String()
}

func validCoordinatePart(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && !strings.ContainsRune("._-", char) {
			return false
		}
	}
	return true
}

func oneOf(value string, options ...string) bool {
	return slices.Contains(options, value)
}

func canonicalExisting(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}

func within(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func lineStart(data []byte, offset int64) int {
	if offset > int64(len(data)) {
		offset = int64(len(data))
	}
	start := bytes.LastIndexByte(data[:offset], '\n') + 1
	return start
}

func lineIndent(data []byte, offset int64) string {
	start := lineStart(data, offset)
	end := min(int(offset), len(data))
	length := 0
	for start+length < end && (data[start+length] == ' ' || data[start+length] == '\t') {
		length++
	}
	return string(data[start : start+length])
}

func indentUnit(data []byte) string {
	for line := range bytes.SplitSeq(data, []byte("\n")) {
		trimmed := bytes.TrimLeft(line, " \t")
		if len(trimmed) == len(line) || len(trimmed) == 0 {
			continue
		}
		indent := line[:len(line)-len(trimmed)]
		if len(indent) > 0 {
			return string(indent)
		}
	}
	return "  "
}

func lineEnding(data []byte) string {
	if bytes.Contains(data, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}
