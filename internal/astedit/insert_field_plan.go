// This file plans one conservative struct-field insertion while exposing checks that require typed workspace analysis.

package astedit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"
)

type (
	// FieldPlacement names a supported position inside one existing struct.
	// FieldPlacement names a supported position inside one existing struct.
	FieldPlacement string

	// InsertFieldIntent supplies the selected struct, new field, and placement.
	// InsertFieldIntent supplies the selected struct, new field, and placement.
	InsertFieldIntent struct {
		TypeName    string
		FieldName   string
		FieldSource string
		Placement   FieldPlacement
		TargetField string
	}
)

// Field positions are scoped to the selected struct body.
// Field positions are scoped to the selected struct body.
const (
	FieldAtStart      FieldPlacement = "type_start"
	FieldAtEnd        FieldPlacement = "type_end"
	FieldBeforeSymbol FieldPlacement = "before_symbol"
	FieldAfterSymbol  FieldPlacement = "after_symbol"
)

// PlanInsertField returns one validated field insertion and its required semantic checks.
// PlanInsertField returns one validated field insertion and its required semantic checks.
func PlanInsertField(source []byte, intent InsertFieldIntent) (SourcePlan, error) {
	if err := validateInsertedField(intent); err != nil {
		return SourcePlan{}, err
	}
	fset, file, err := parsePlanSource(source)
	if err != nil {
		return SourcePlan{}, err
	}
	selected, err := findInsertFieldType(file, intent.TypeName)
	if err != nil {
		return SourcePlan{}, err
	}
	structure := selected.Type.(*ast.StructType)
	openOffset := fset.Position(structure.Fields.Opening).Offset
	closeOffset := fset.Position(structure.Fields.Closing).Offset
	fieldBody := source[openOffset+1 : closeOffset]
	emptyBody := len(structure.Fields.List) == 0 && len(bytes.TrimSpace(fieldBody)) == 0
	if !bytes.Contains(fieldBody, []byte("\n")) && !emptyBody {
		return SourcePlan{}, fmt.Errorf("struct type %q must use a multiline field layout", intent.TypeName)
	}
	if err := validateInsertFieldContext(file, structure, intent.TypeName, intent.FieldName); err != nil {
		return SourcePlan{}, err
	}
	newline := "\n"
	if bytes.Contains(source, []byte("\r\n")) {
		newline = "\r\n"
	}
	var updated []byte
	if emptyBody {
		placement := intent.Placement
		if placement == "" {
			placement = FieldAtEnd
		}
		if placement != FieldAtStart && placement != FieldAtEnd {
			return SourcePlan{}, fmt.Errorf("relative field placement requires an existing field anchor")
		}
		_, ownerIndent := fieldLineStart(source, openOffset)
		fieldIndent := ownerIndent + "\t"
		replacement := []byte(newline + fieldIndent + strings.TrimSpace(intent.FieldSource) + newline + ownerIndent)
		updated = spliceSource(source, openOffset+1, closeOffset, replacement)
	} else {
		offset, indent, err := fieldInsertionOffset(source, fset, structure, intent)
		if err != nil {
			return SourcePlan{}, err
		}
		updated = spliceSource(source, offset, offset, []byte(indent+strings.TrimSpace(intent.FieldSource)+newline))
	}
	return newSourcePlan(source, updated, PlanSelection{
		Kind:   "field",
		Owner:  intent.TypeName,
		Target: intent.TypeName + "." + intent.FieldName,
		Start:  fset.Position(selected.Pos()).Offset,
		End:    fset.Position(selected.End()).Offset,
	}, ProofCompleteSourceSet, ProofTypedMemberSet, ProofUnkeyedLiteralSet), nil
}

func fieldLineStart(source []byte, offset int) (int, string) {
	if offset > len(source) {
		offset = len(source)
	}
	start := bytes.LastIndexByte(source[:offset], '\n') + 1
	indentEnd := start
	for indentEnd < len(source) && (source[indentEnd] == ' ' || source[indentEnd] == '\t') {
		indentEnd++
	}
	return start, string(source[start:indentEnd])
}

func fieldLineEnd(source []byte, offset int) (int, string) {
	if offset > len(source) {
		offset = len(source)
	}
	end := bytes.IndexByte(source[offset:], '\n')
	if end < 0 {
		end = len(source)
	} else {
		end += offset + 1
	}
	start := bytes.LastIndexByte(source[:offset], '\n') + 1
	indentEnd := start
	for indentEnd < len(source) && (source[indentEnd] == ' ' || source[indentEnd] == '\t') {
		indentEnd++
	}
	return end, string(source[start:indentEnd])
}

func spliceSource(source []byte, start, end int, replacement []byte) []byte {
	result := make([]byte, 0, len(source)-(end-start)+len(replacement))
	result = append(result, source[:start]...)
	result = append(result, replacement...)
	result = append(result, source[end:]...)
	return result
}

func validateInsertedField(intent InsertFieldIntent) error {
	if !token.IsIdentifier(intent.TypeName) || token.Lookup(intent.TypeName).IsKeyword() {
		return fmt.Errorf("invalid struct type name %q", intent.TypeName)
	}
	if !token.IsIdentifier(intent.FieldName) || intent.FieldName == "_" || token.Lookup(intent.FieldName).IsKeyword() {
		return fmt.Errorf("field name %q must be a named Go identifier", intent.FieldName)
	}
	fieldFile, err := parser.ParseFile(token.NewFileSet(), "field.go", []byte("package p\ntype holder struct {\n"+intent.FieldSource+"\n}"), parser.ParseComments|parser.AllErrors)
	if err != nil {
		return fmt.Errorf("parse inserted field: %w", err)
	}
	if len(fieldFile.Decls) != 1 {
		return fmt.Errorf("field source must not inject additional declarations")
	}
	fieldDecl, ok := fieldFile.Decls[0].(*ast.GenDecl)
	if !ok || len(fieldDecl.Specs) != 1 {
		return fmt.Errorf("field source must contain exactly one field declaration")
	}
	fieldSpec, ok := fieldDecl.Specs[0].(*ast.TypeSpec)
	if !ok {
		return fmt.Errorf("field source must contain exactly one field declaration")
	}
	fieldType, ok := fieldSpec.Type.(*ast.StructType)
	if !ok {
		return fmt.Errorf("field source must contain exactly one field declaration")
	}
	if len(fieldType.Fields.List) != 1 || len(fieldType.Fields.List[0].Names) != 1 || fieldType.Fields.List[0].Names[0].Name != intent.FieldName {
		return fmt.Errorf("field source must declare exactly the named field %q", intent.FieldName)
	}
	if slices.ContainsFunc(fieldFile.Comments, containsDirectiveGroup) {
		return fmt.Errorf("field source must not contain compiler or build directives")
	}
	return nil
}

func findInsertFieldType(file *ast.File, typeName string) (*ast.TypeSpec, error) {
	var selected *ast.TypeSpec
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != typeName {
				continue
			}
			if selected != nil {
				return nil, fmt.Errorf("struct type %q is ambiguous", typeName)
			}
			selected = typeSpec
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("struct type %q not found", typeName)
	}
	if selected.Assign.IsValid() || selected.TypeParams != nil && len(selected.TypeParams.List) != 0 {
		return nil, fmt.Errorf("struct type %q must be non-alias and non-generic", typeName)
	}
	if _, ok := selected.Type.(*ast.StructType); !ok {
		return nil, fmt.Errorf("type %q is not a struct", typeName)
	}
	return selected, nil
}

func validateInsertFieldContext(file *ast.File, structure *ast.StructType, typeName, fieldName string) error {
	if err := validateUniqueStructField(structure, typeName, fieldName); err != nil {
		return err
	}
	if err := validateFieldMethodCollision(file, typeName, fieldName); err != nil {
		return err
	}
	if hasUnkeyedStructLiteral(file, typeName) {
		return fmt.Errorf("cannot insert field %q while this file contains an unkeyed %s literal", fieldName, typeName)
	}
	return nil
}

func validateUniqueStructField(structure *ast.StructType, typeName, fieldName string) error {
	for _, field := range structure.Fields.List {
		if len(field.Names) == 0 {
			return fmt.Errorf("struct type %q contains an embedded field", typeName)
		}
		for _, name := range field.Names {
			if name.Name == fieldName {
				return fmt.Errorf("field %q already exists on %s", fieldName, typeName)
			}
		}
	}
	return nil
}

func validateFieldMethodCollision(file *ast.File, typeName, fieldName string) error {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name.Name != fieldName {
			continue
		}
		for _, recv := range fn.Recv.List {
			base := recv.Type
			if pointer, ok := base.(*ast.StarExpr); ok {
				base = pointer.X
			}
			if ident, ok := base.(*ast.Ident); ok && ident.Name == typeName {
				return fmt.Errorf("field %q collides with a method on %s", fieldName, typeName)
			}
		}
	}
	return nil
}

func hasUnkeyedStructLiteral(file *ast.File, typeName string) bool {
	var found bool
	ast.Inspect(file, func(node ast.Node) bool {
		if found {
			return false
		}
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		ident, ok := literal.Type.(*ast.Ident)
		if !ok || ident.Name != typeName || len(literal.Elts) == 0 {
			return true
		}
		for _, element := range literal.Elts {
			if _, ok := element.(*ast.KeyValueExpr); !ok {
				found = true
				break
			}
		}
		return true
	})
	return found
}

func fieldInsertionOffset(source []byte, fset *token.FileSet, structure *ast.StructType, intent InsertFieldIntent) (int, string, error) {
	if intent.Placement == "" {
		intent.Placement = FieldAtEnd
	}
	switch intent.Placement {
	case FieldAtStart:
		if len(structure.Fields.List) == 0 {
			offset, indent := fieldLineStart(source, fset.Position(structure.Fields.Closing).Offset)
			return offset, indent + "\t", nil
		}
		anchor := structure.Fields.List[0]
		position := anchor.Pos()
		if anchor.Doc != nil {
			position = anchor.Doc.Pos()
		}
		offset, indent := fieldLineStart(source, fset.Position(position).Offset)
		return offset, indent, nil
	case FieldAtEnd:
		offset, indent := fieldLineStart(source, fset.Position(structure.Fields.Closing).Offset)
		return offset, indent + "\t", nil
	case FieldBeforeSymbol, FieldAfterSymbol:
		if intent.TargetField == "" {
			return 0, "", fmt.Errorf("target field is required for relative placement")
		}
		var anchor *ast.Field
		for _, field := range structure.Fields.List {
			for _, name := range field.Names {
				if name.Name != intent.TargetField {
					continue
				}
				if anchor != nil {
					return 0, "", fmt.Errorf("anchor field %q is ambiguous", intent.TargetField)
				}
				anchor = field
			}
		}
		if anchor == nil {
			return 0, "", fmt.Errorf("anchor field %q not found on %s", intent.TargetField, intent.TypeName)
		}
		position := anchor.Pos()
		if intent.Placement == FieldBeforeSymbol {
			if anchor.Doc != nil {
				position = anchor.Doc.Pos()
			}
			offset, indent := fieldLineStart(source, fset.Position(position).Offset)
			return offset, indent, nil
		}
		position = anchor.End()
		if anchor.Comment != nil {
			position = anchor.Comment.End()
		}
		offset, indent := fieldLineEnd(source, fset.Position(position).Offset)
		return offset, indent, nil
	default:
		return 0, "", fmt.Errorf("unsupported field placement %q", intent.Placement)
	}
}
