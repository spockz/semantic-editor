// Package integration validates and edits Codex TOML without rewriting unrelated source.
package integration

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

func mergeCodex(data []byte, req Request, exists bool) ([]byte, bool, error) {
	if !exists || len(strings.TrimSpace(string(data))) == 0 {
		return []byte(codexSection(req)), false, nil
	}
	sections, err := parseTOMLSections(string(data))
	if err != nil {
		return nil, false, fmt.Errorf("parse Codex config: %w", err)
	}
	section, ok := sections["mcp_servers.semedit"]
	if !ok {
		separator := ""
		if !strings.HasSuffix(string(data), "\n") {
			separator = "\n"
		}
		return append(data, []byte(separator+codexSection(req))...), false, nil
	}
	currentBinary, currentProfile, err := sectionRegistration(section)
	if err != nil {
		return nil, false, err
	}
	if currentBinary == req.Binary && currentProfile == req.Profile {
		return data, true, nil
	}
	if !req.Replace {
		return nil, false, fmt.Errorf("%w: Codex registration %q differs from requested command/profile", ErrConflict, serverName)
	}
	return replaceCodexSection(data, section, req), false, nil
}

func inspectCodex(data []byte) (string, string, bool, error) {
	sections, err := parseTOMLSections(string(data))
	if err != nil {
		return "", "", false, err
	}
	section, ok := sections["mcp_servers.semedit"]
	if !ok {
		return "", "", false, nil
	}
	binary, profile, err := sectionRegistration(section)
	if err != nil {
		return "", "", false, err
	}
	return binary, profile, binary != "", nil
}

func removeCodex(data []byte) ([]byte, bool, error) {
	sections, err := parseTOMLSections(string(data))
	if err != nil {
		return nil, false, err
	}
	section, ok := sections["mcp_servers.semedit"]
	if !ok {
		return data, false, nil
	}
	start, end := section.start, section.end
	updated := append([]byte(nil), data[:start]...)
	updated = append(updated, data[end:]...)
	return updated, true, nil
}

func currentCodexRegistration(data []byte) ([]byte, error) {
	sections, err := parseTOMLSections(string(data))
	if err != nil {
		return nil, err
	}
	section, ok := sections["mcp_servers.semedit"]
	if !ok {
		return nil, fmt.Errorf("semedit registration not found")
	}
	return []byte(strings.TrimSpace(strings.Join(section.lines, "")) + "\n"), nil
}

func codexSection(req Request) string {
	return fmt.Sprintf("[mcp_servers.semedit]\ncommand = %s\nargs = [\"mcp\", \"--profile\", %s]\n", quoteTOML(req.Binary), quoteTOML(req.Profile))
}

type tomlSection struct {
	start, end int
	lines      []string
}

type tomlSectionParser struct {
	sections map[string]tomlSection
	current  string
	start    int
	offset   int
	lines    []string
}

func parseTOMLSections(text string) (map[string]tomlSection, error) {
	var decoded map[string]any
	if _, err := toml.Decode(text, &decoded); err != nil {
		return nil, err
	}
	parser := tomlSectionParser{sections: make(map[string]tomlSection)}
	for _, line := range strings.SplitAfter(text, "\n") {
		if err := parser.consume(line); err != nil {
			return nil, err
		}
	}
	if err := parser.closeCurrent(); err != nil {
		return nil, err
	}
	return parser.sections, nil
}

func (p *tomlSectionParser) consume(line string) error {
	trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\n"))
	header := strings.TrimSpace(stripTOMLComment(trimmed))
	var err error
	if strings.HasPrefix(header, "[") {
		err = p.consumeHeader(header, trimmed, line)
	} else {
		err = p.consumeSetting(trimmed, line)
	}
	p.offset += len(line)
	return err
}

func (p *tomlSectionParser) consumeHeader(header, trimmed, line string) error {
	if !strings.HasSuffix(header, "]") {
		return fmt.Errorf("invalid table header %q", trimmed)
	}
	if strings.HasPrefix(header, "[[") {
		name := normalizeTOMLTableName(strings.TrimSpace(header[2 : len(header)-2]))
		if name == "mcp_servers.semedit" {
			return fmt.Errorf("semedit registration cannot be an array table")
		}
		if err := p.closeCurrent(); err != nil {
			return err
		}
		p.current, p.lines = "", nil
		return nil
	}
	name := normalizeTOMLTableName(strings.TrimSpace(header[1 : len(header)-1]))
	if name == "" {
		return fmt.Errorf("empty table header")
	}
	if err := p.closeCurrent(); err != nil {
		return err
	}
	p.current, p.start, p.lines = name, p.offset, []string{line}
	return nil
}

func (p *tomlSectionParser) consumeSetting(trimmed, line string) error {
	if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
		if _, _, ok := strings.Cut(trimmed, "="); !ok {
			return fmt.Errorf("invalid TOML setting %q", trimmed)
		}
	}
	if p.current != "" {
		p.lines = append(p.lines, line)
	}
	return nil
}

func (p *tomlSectionParser) closeCurrent() error {
	if p.current == "" {
		return nil
	}
	if _, duplicate := p.sections[p.current]; duplicate {
		return fmt.Errorf("duplicate TOML table %q", p.current)
	}
	p.sections[p.current] = tomlSection{start: p.start, end: p.offset, lines: p.lines}
	p.current, p.lines = "", nil
	return nil
}

func normalizeTOMLTableName(name string) string {
	name = strings.ReplaceAll(name, `"`, "")
	return strings.ReplaceAll(name, `'`, "")
}

func stripTOMLComment(line string) string {
	inString, escaped := false, false
	for index, char := range line {
		if inString {
			switch {
			case escaped:
				escaped = false
			case char == '\\':
				escaped = true
			case char == '"':
				inString = false
			}
			continue
		}
		switch char {
		case '"':
			inString = true
		case '#':
			return line[:index]
		}
	}
	return line
}

func sectionRegistration(section tomlSection) (string, string, error) {
	var decoded map[string]any
	if _, err := toml.Decode(strings.Join(section.lines, ""), &decoded); err != nil {
		return "", "", err
	}
	root, ok := decoded["mcp_servers"].(map[string]any)
	if !ok {
		return "", "", fmt.Errorf("mcp_servers is not a table")
	}
	registration, ok := root["semedit"].(map[string]any)
	if !ok {
		return "", "", fmt.Errorf("semedit registration is not a table")
	}
	binary, ok := registration["command"].(string)
	if !ok || binary == "" {
		return "", "", fmt.Errorf("codex command must be a string")
	}
	args, err := stringArray(registration["args"])
	if err != nil {
		return "", "", err
	}
	profile := ""
	for index, arg := range args {
		if arg == "--profile" && index+1 < len(args) {
			profile = args[index+1]
		}
	}
	return binary, profile, nil
}

func stringArray(value any) ([]string, error) {
	switch typed := value.(type) {
	case []string:
		return typed, nil
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("codex args must be a string array")
			}
			result = append(result, text)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("codex args must be a string array")
	}
}

func replaceCodexSection(data []byte, section tomlSection, req Request) []byte {
	return append(append(append([]byte(nil), data[:section.start]...), []byte(codexSection(req))...), data[section.end:]...)
}
