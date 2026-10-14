// WHY: This test guarantees that skills/semedit/SKILL.md and skills/semedit/references/ accurately reflect the operation registry without documentation or schema drift.

package operation_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"semedit/internal/backend"
	gobackend "semedit/internal/backend/golang"
	javabackend "semedit/internal/backend/java"
	"semedit/internal/operation"
)

var (
	toolNameRegex   = regexp.MustCompile(`\b(semantic_[a-z0-9_]+|report_feedback)\b`)
	backtickedRegex = regexp.MustCompile("`([^`]+)`")
	kindValueRegex  = regexp.MustCompile(`kind:\s*"([^"]+)"`)
	placeValueRegex = regexp.MustCompile(`placement:\s*"([^"]+)"`)

	mcpProtocolTools = map[string]bool{
		"semantic_batch":  true,
		"semantic_reload": true,
		"report_feedback": true,
	}
)

func TestSkillSchemaSynchronisation_AllMCPToolsDocumented(t *testing.T) {
	t.Parallel()

	rootDir := findRepoRoot(t)
	skillPath := filepath.Join(rootDir, "skills", "semedit", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v", skillPath, err)
	}
	skillText := string(content)

	registry := operation.DefaultRegistry()
	for _, entry := range registry.All() {
		if entry.MCPName == "" {
			continue
		}
		if !strings.Contains(skillText, entry.MCPName) {
			t.Errorf("skills/semedit/SKILL.md does not document registered MCP tool %q", entry.MCPName)
		}
	}
}

func TestSkillSchemaSynchronisation_NoUnknownToolsDocumented(t *testing.T) {
	t.Parallel()

	rootDir := findRepoRoot(t)
	skillPath := filepath.Join(rootDir, "skills", "semedit", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v", skillPath, err)
	}

	registry := operation.DefaultRegistry()
	matches := toolNameRegex.FindAllString(string(content), -1)
	seen := make(map[string]bool)

	for _, toolName := range matches {
		if seen[toolName] {
			continue
		}
		seen[toolName] = true

		if mcpProtocolTools[toolName] {
			continue
		}
		if _, ok := registry.LookupMCP(toolName); !ok {
			t.Errorf("skills/semedit/SKILL.md mentions unknown or unadvertised tool %q", toolName)
		}
	}
}

func TestSkillSchemaSynchronisation_RoutingTableParametersMatchRegistry(t *testing.T) {
	t.Parallel()

	rootDir := findRepoRoot(t)
	skillPath := filepath.Join(rootDir, "skills", "semedit", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v", skillPath, err)
	}

	lines := strings.Split(string(content), "\n")
	tableLines := extractRoutingTableLines(lines)
	if len(tableLines) == 0 {
		t.Fatal("no routing table found in skills/semedit/SKILL.md")
	}

	registry := operation.DefaultRegistry()

	for _, line := range tableLines {
		cols := strings.Split(line, "|")
		if len(cols) < 4 {
			continue
		}
		toolCol := cols[2]
		toolMatches := toolNameRegex.FindAllString(toolCol, -1)
		if len(toolMatches) == 0 {
			continue
		}

		// The primary tool is the first tool mentioned in the recommendation column.
		primaryTool := toolMatches[0]
		if mcpProtocolTools[primaryTool] {
			continue
		}

		entry, ok := registry.LookupMCP(primaryTool)
		if !ok {
			t.Errorf("routing table mentions unregistered tool %q", primaryTool)
			continue
		}

		validParams := collectValidParameterNames(entry)
		validEnums := collectValidEnumValues(entry)
		tokens := extractBacktickedTokens(toolCol)

		for _, token := range tokens {
			if shouldSkipParameterToken(token, primaryTool) {
				continue
			}
			if slices.Contains(validEnums, token) {
				continue
			}
			if !slices.Contains(validParams, token) {
				t.Errorf("tool %q in routing table references unknown parameter %q; valid parameters: %v", primaryTool, token, validParams)
			}
		}
	}
}

func TestSkillSchemaSynchronisation_EnumValuesMatchRegistry(t *testing.T) {
	t.Parallel()

	rootDir := findRepoRoot(t)
	skillPath := filepath.Join(rootDir, "skills", "semedit", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v", skillPath, err)
	}

	registry := operation.DefaultRegistry()

	kindMatches := kindValueRegex.FindAllStringSubmatch(string(content), -1)
	for _, match := range kindMatches {
		val := match[1]
		assertEnumValuePresent(t, registry, "kind", val)
	}

	placeMatches := placeValueRegex.FindAllStringSubmatch(string(content), -1)
	for _, match := range placeMatches {
		val := match[1]
		assertEnumValuePresent(t, registry, "placement", val)
	}
}

func TestSkillSchemaSynchronisation_LanguageReferencesCoverRegisteredTools(t *testing.T) {
	t.Parallel()

	rootDir := findRepoRoot(t)
	referencesDir := filepath.Join(rootDir, "skills", "semedit", "references")
	entries, err := os.ReadDir(referencesDir)
	if err != nil {
		t.Fatalf("read references dir %s: %v", referencesDir, err)
	}

	registry := operation.DefaultRegistry()

	for _, refEntry := range entries {
		if refEntry.IsDir() || !strings.HasSuffix(refEntry.Name(), ".md") {
			continue
		}

		t.Run(refEntry.Name(), func(t *testing.T) {
			t.Parallel()
			refPath := filepath.Join(referencesDir, refEntry.Name())
			data, readErr := os.ReadFile(refPath)
			if readErr != nil {
				t.Fatalf("read reference file %s: %v", refPath, readErr)
			}
			matches := toolNameRegex.FindAllString(string(data), -1)
			for _, toolName := range matches {
				if mcpProtocolTools[toolName] {
					continue
				}
				if _, ok := registry.LookupMCP(toolName); !ok {
					t.Errorf("reference file %s references unknown tool %q", refEntry.Name(), toolName)
				}
			}
		})
	}
}

func extractRoutingTableLines(lines []string) []string {
	var tableLines []string
	inTable := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "| Requested change |") {
			inTable = true
			continue
		}
		if inTable {
			if !strings.HasPrefix(trimmed, "|") {
				break
			}
			if strings.Contains(trimmed, "---") {
				continue
			}
			tableLines = append(tableLines, trimmed)
		}
	}
	return tableLines
}

func collectValidParameterNames(entry operation.Entry) []string {
	var names []string
	addName := func(n string) {
		if n != "" && !slices.Contains(names, n) {
			names = append(names, n)
		}
	}

	var walkSchema func(s *operation.ValueSchema)
	walkSchema = func(s *operation.ValueSchema) {
		if s == nil {
			return
		}
		for propName, propSchema := range s.Properties {
			addName(propName)
			walkSchema(&propSchema)
		}
		if s.Items != nil {
			walkSchema(s.Items)
		}
	}

	for _, p := range entry.Params {
		addName(p.JSONName)
		addName(p.CLIName)
		for _, alias := range p.InputAliases {
			addName(alias)
		}
		walkSchema(p.Schema)
	}
	return names
}

func collectValidEnumValues(entry operation.Entry) []string {
	var enums []string
	addEnum := func(e string) {
		if e != "" && !slices.Contains(enums, e) {
			enums = append(enums, e)
		}
	}
	backends := []backend.Backend{gobackend.GoBackend{}, &javabackend.JavaBackend{}}
	for _, p := range entry.Params {
		for _, e := range p.Enums {
			addEnum(e)
		}
		if p.Schema != nil {
			for _, item := range p.Schema.Enum {
				if s, ok := item.(string); ok {
					addEnum(s)
				}
			}
		}
		if p.DynamicEnums != nil {
			for _, b := range backends {
				for _, e := range p.DynamicEnums(b) {
					addEnum(e)
				}
			}
		}
	}
	return enums
}

func extractBacktickedTokens(s string) []string {
	matches := backtickedRegex.FindAllStringSubmatch(s, -1)
	tokens := make([]string, 0, len(matches))
	for _, m := range matches {
		tok := strings.TrimSpace(m[1])
		tokens = append(tokens, tok)
	}
	return tokens
}

func shouldSkipParameterToken(token, toolName string) bool {
	if token == toolName || strings.HasPrefix(token, "semantic_") || mcpProtocolTools[token] {
		return true
	}
	if strings.ContainsAny(token, " ()`\"=,./:*") {
		return true
	}
	skipKeywords := []string{
		"auto", "go", "java", "rust", "scala", "kotlin", "haskell",
		"true", "false", "dependencyManagement",
	}
	return slices.Contains(skipKeywords, token)
}

func assertEnumValuePresent(t *testing.T, registry *operation.Registry, paramName, expectedVal string) {
	t.Helper()
	found := false
	backends := []backend.Backend{gobackend.GoBackend{}, &javabackend.JavaBackend{}}
	for _, entry := range registry.All() {
		for _, param := range entry.Params {
			if param.JSONName == paramName || param.CLIName == paramName {
				if slices.Contains(param.Enums, expectedVal) {
					found = true
					break
				}
				if param.DynamicEnums != nil {
					for _, b := range backends {
						if slices.Contains(param.DynamicEnums(b), expectedVal) {
							found = true
							break
						}
					}
				}
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Errorf("documented %s value %q is not present in registry enums for any operation", paramName, expectedVal)
	}
}
