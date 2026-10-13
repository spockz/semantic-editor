// Package java implements Java language-server operations and source transformations.
// WHY: This file applies import requests to the source buffer that JDT LS organizes.
package java

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

var javaImportNamePattern = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*(\.[A-Za-z_$][A-Za-z0-9_$]*)+$`)

func applyJavaExplicitImports(source []byte, add, remove []string) ([]byte, error) {
	for _, name := range append(append([]string(nil), add...), remove...) {
		if !javaImportNamePattern.MatchString(name) {
			return nil, fmt.Errorf("invalid Java import name %q", name)
		}
	}
	updated := append([]byte(nil), source...)
	for _, name := range remove {
		lines := bytes.SplitAfter(updated, []byte("\n"))
		var filtered bytes.Buffer
		for i := 0; i < len(lines); i++ {
			line := lines[i]
			trimmed := strings.TrimSpace(string(line))
			if trimmed == "import "+name+";" {
				if i+1 < len(lines) && strings.TrimSpace(string(lines[i+1])) == "" {
					i++
				}
				continue
			}
			filtered.Write(line)
		}
		updated = filtered.Bytes()
	}
	for _, name := range add {
		if hasJavaImport(updated, name) {
			continue
		}
		updated = insertJavaImport(updated, name)
	}
	return updated, nil
}

func hasJavaImport(source []byte, name string) bool {
	for line := range bytes.SplitSeq(source, []byte("\n")) {
		if strings.TrimSpace(string(line)) == "import "+name+";" {
			return true
		}
	}
	return false
}

func insertJavaImport(source []byte, name string) []byte {
	lines := bytes.SplitAfter(source, []byte("\n"))
	offset, insertAfter := 0, -1
	hasPackage, hasImports := false, false
	for _, line := range lines {
		trimmed := strings.TrimSpace(string(line))
		offset += len(line)
		if strings.HasPrefix(trimmed, "import ") && strings.HasSuffix(trimmed, ";") {
			hasImports = true
			insertAfter = offset
			continue
		}
		if insertAfter < 0 && strings.HasPrefix(trimmed, "package ") && strings.HasSuffix(trimmed, ";") {
			hasPackage = true
			insertAfter = offset
			continue
		}
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") && !strings.HasPrefix(trimmed, "*") {
			break
		}
	}
	if insertAfter < 0 {
		insertAfter = 0
	}
	eol := []byte("\n")
	if bytes.Contains(source, []byte("\r\n")) {
		eol = []byte("\r\n")
	}
	addition := append([]byte("import "+name+";"), eol...)
	if hasPackage && !hasImports {
		addition = append(eol, addition...)
	}
	updated := make([]byte, 0, len(source)+len(addition))
	updated = append(updated, source[:insertAfter]...)
	updated = append(updated, addition...)
	updated = append(updated, source[insertAfter:]...)
	return updated
}
