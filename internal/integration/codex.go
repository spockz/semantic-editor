package integration

import (
	"fmt"
	"strconv"
	"strings"
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

func codexSection(req Request) string {
	return fmt.Sprintf("[mcp_servers.semedit]\ncommand = %s\nargs = [\"mcp\", \"--profile\", %s]\n", quoteTOML(req.Binary), quoteTOML(req.Profile))
}

type tomlSection struct {
	start, end int
	lines      []string
}

func parseTOMLSections(text string) (map[string]tomlSection, error) {
	result := map[string]tomlSection{}
	offset := 0
	current := ""
	start := 0
	lines := []string{}
	for _, line := range strings.SplitAfter(text, "\n") {
		trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\n"))
		switch {
		case strings.HasPrefix(trimmed, "["):
			if !strings.HasSuffix(trimmed, "]") || strings.HasPrefix(trimmed, "[[") {
				return nil, fmt.Errorf("invalid table header %q", trimmed)
			}
			name := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			if name == "" {
				return nil, fmt.Errorf("empty table header")
			}
			if current != "" {
				result[current] = tomlSection{start: start, end: offset, lines: lines}
			}
			current, start, lines = name, offset, []string{line}
		case trimmed != "" && !strings.HasPrefix(trimmed, "#"):
			if _, _, ok := strings.Cut(trimmed, "="); !ok {
				return nil, fmt.Errorf("invalid TOML setting %q", trimmed)
			}
			if current != "" {
				lines = append(lines, line)
			}
		default:
			if current != "" {
				lines = append(lines, line)
			}
		}
		offset += len(line)
	}
	if current != "" {
		result[current] = tomlSection{start: start, end: offset, lines: lines}
	}
	return result, nil
}

func sectionRegistration(section tomlSection) (string, string, error) {
	values := map[string]string{}
	for _, line := range section.lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value, ok := strings.Cut(trimmed, "=")
		if !ok {
			return "", "", fmt.Errorf("invalid Codex MCP setting %q", trimmed)
		}
		key = strings.TrimSpace(key)
		if _, duplicate := values[key]; duplicate {
			return "", "", fmt.Errorf("duplicate Codex MCP setting %q", key)
		}
		values[key] = strings.TrimSpace(value)
	}
	binary, err := parseTOMLString(values["command"])
	if err != nil {
		return "", "", fmt.Errorf("parse Codex command: %w", err)
	}
	args, err := parseTOMLArray(values["args"])
	if err != nil {
		return "", "", fmt.Errorf("parse Codex args: %w", err)
	}
	profile := ""
	for index, arg := range args {
		if arg == "--profile" && index+1 < len(args) {
			profile = args[index+1]
		}
	}
	return binary, profile, nil
}

func replaceCodexSection(data []byte, section tomlSection, req Request) []byte {
	return append(append(append([]byte(nil), data[:section.start]...), []byte(codexSection(req))...), data[section.end:]...)
}

func parseTOMLString(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("missing string")
	}
	parsed, err := strconv.Unquote(value)
	if err != nil {
		return "", fmt.Errorf("expected quoted string")
	}
	return parsed, nil
}

func parseTOMLArray(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if len(value) < 2 || value[0] != '[' || value[len(value)-1] != ']' {
		return nil, fmt.Errorf("expected string array")
	}
	inner := strings.TrimSpace(value[1 : len(value)-1])
	if inner == "" {
		return nil, nil
	}
	parts := strings.Split(inner, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		parsed, err := parseTOMLString(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		result = append(result, parsed)
	}
	return result, nil
}
