// Package astedit plans one encapsulation edit set from typed reference ranges supplied by the operation layer.
// WHY: Keep source splicing deterministic while the operation layer owns type identity, reference completeness, and publication safety.
package astedit

import (
	"bytes"
	"fmt"
	"go/format"
	"go/token"
	"strings"
)

// FieldReferenceRewrite identifies one source range already proven to belong to the selected field.
type FieldReferenceRewrite struct {
	File        string
	Start, End  int
	Replacement string
}

// EncapsulateFieldIntent describes private accessor declarations and typed reference rewrites.
type EncapsulateFieldIntent struct {
	DestinationFile string
	OwnerType       string
	FieldName       string
	FieldType       string
	Getter          string
	Setter          string
}

// PlanEncapsulateField combines proven reference rewrites and both accessors without mutating sources.
func PlanEncapsulateField(files []SignatureSourceFile, intent EncapsulateFieldIntent, rewrites []FieldReferenceRewrite) (SignaturePlan, error) {
	if err := validateEncapsulateFieldIntent(intent); err != nil {
		return SignaturePlan{}, err
	}
	accessors, err := formatEncapsulateFieldAccessors(intent)
	if err != nil {
		return SignaturePlan{}, err
	}
	sources, err := encapsulateSourceMap(files, intent.DestinationFile)
	if err != nil {
		return SignaturePlan{}, err
	}
	byFile, err := groupEncapsulateRewrites(sources, rewrites)
	if err != nil {
		return SignaturePlan{}, err
	}
	filesByPath, err := applyEncapsulateEdits(sources, byFile, intent.DestinationFile, accessors)
	if err != nil {
		return SignaturePlan{}, err
	}
	return signaturePlanFromSources(sources, filesByPath)
}

func encapsulateSourceMap(files []SignatureSourceFile, destination string) (map[string][]byte, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("encapsulate field: complete source set is required")
	}
	sources := make(map[string][]byte, len(files))
	for _, file := range files {
		if file.File == "" {
			return nil, fmt.Errorf("encapsulate field: source path is required")
		}
		if _, exists := sources[file.File]; exists {
			return nil, fmt.Errorf("encapsulate field: duplicate source path %q", file.File)
		}
		sources[file.File] = bytes.Clone(file.Source)
	}
	if _, ok := sources[destination]; !ok {
		return nil, fmt.Errorf("encapsulate field: destination %q is not in the analyzed source set", destination)
	}
	return sources, nil
}

func groupEncapsulateRewrites(sources map[string][]byte, rewrites []FieldReferenceRewrite) (map[string][]FieldReferenceRewrite, error) {
	byFile := make(map[string][]FieldReferenceRewrite)
	for _, rewrite := range rewrites {
		if rewrite.Replacement == "" || rewrite.File == "" {
			return nil, fmt.Errorf("encapsulate field: rewrite path and replacement are required")
		}
		if _, ok := sources[rewrite.File]; !ok {
			return nil, fmt.Errorf("encapsulate field: rewrite source %q is outside the analyzed source set", rewrite.File)
		}
		if rewrite.Start < 0 || rewrite.End <= rewrite.Start || rewrite.End > len(sources[rewrite.File]) {
			return nil, fmt.Errorf("encapsulate field: invalid rewrite range in %q", rewrite.File)
		}
		byFile[rewrite.File] = append(byFile[rewrite.File], rewrite)
	}
	return byFile, nil
}

func applyEncapsulateEdits(sources map[string][]byte, byFile map[string][]FieldReferenceRewrite, destination string, accessors []byte) (map[string][]byte, error) {
	filesByPath := make(map[string][]byte)
	for path, source := range sources {
		edits := make([]signatureSourceEdit, 0, len(byFile[path])+1)
		for _, edit := range byFile[path] {
			edits = append(edits, signatureSourceEdit{Start: edit.Start, End: edit.End, Replacement: edit.Replacement})
		}
		if path == destination {
			edits = append(edits, signatureSourceEdit{Start: len(source), End: len(source), Replacement: string(accessorAppend(source, accessors))})
		}
		updated, err := signatureApplyEdits(source, edits)
		if err != nil {
			return nil, fmt.Errorf("encapsulate field: apply edits to %q: %w", path, err)
		}
		if !bytes.Equal(source, updated) {
			filesByPath[path] = updated
		}
	}
	return filesByPath, nil
}
func validateEncapsulateFieldIntent(intent EncapsulateFieldIntent) error {
	for label, value := range map[string]string{"destination file": intent.DestinationFile, "owner type": intent.OwnerType, "field name": intent.FieldName, "field type": intent.FieldType} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("encapsulate field: %s is required", label)
		}
	}
	if !privateMethodName(intent.Getter) || !privateMethodName(intent.Setter) || intent.Getter == intent.Setter {
		return fmt.Errorf("encapsulate field: getter and setter must be distinct private Go method names")
	}
	if !token.IsIdentifier(intent.OwnerType) || token.Lookup(intent.OwnerType).IsKeyword() || !token.IsIdentifier(intent.FieldName) || token.Lookup(intent.FieldName).IsKeyword() {
		return fmt.Errorf("encapsulate field: owner type and field name must be Go identifiers")
	}
	return nil
}

func privateMethodName(name string) bool {
	return name != "" && token.IsIdentifier(name) && !token.Lookup(name).IsKeyword() && name[0] >= 'a' && name[0] <= 'z'
}

func formatEncapsulateFieldAccessors(intent EncapsulateFieldIntent) ([]byte, error) {
	source := fmt.Sprintf("package p\nfunc (receiver *%s) %s() %s { return receiver.%s }\nfunc (receiver *%s) %s(value %s) { receiver.%s = value }\n",
		intent.OwnerType, intent.Getter, intent.FieldType, intent.FieldName,
		intent.OwnerType, intent.Setter, intent.FieldType, intent.FieldName)
	formatted, err := format.Source([]byte(source))
	if err != nil {
		return nil, fmt.Errorf("format generated field accessors: %w", err)
	}
	return bytes.TrimPrefix(formatted, []byte("package p\n")), nil
}

func accessorAppend(source, accessors []byte) []byte {
	separator := "\n"
	if bytes.Contains(source, []byte("\r\n")) {
		separator = "\r\n"
	}
	var out bytes.Buffer
	out.WriteString(separator)
	if len(source) > 0 && source[len(source)-1] != '\n' {
		out.WriteString(separator)
	}
	trimmed := bytes.TrimSpace(accessors)
	out.Write(trimmed)
	out.WriteString(separator)
	return out.Bytes()
}
