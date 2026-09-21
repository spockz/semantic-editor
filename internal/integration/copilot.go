package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func copilotRegistration(req Request) map[string]any {
	return map[string]any{
		"type":    "stdio",
		"command": req.Binary,
		"args":    []string{"mcp", "--profile", req.Profile},
	}
}

func mergeCopilot(data []byte, req Request, exists bool) ([]byte, bool, error) {
	root := map[string]any{}
	if exists && len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(stripJSONComments(data), &root); err != nil {
			return nil, false, fmt.Errorf("parse Copilot MCP config: %w", err)
		}
	}
	servers, ok := root["servers"].(map[string]any)
	if !ok {
		if value, present := root["servers"]; present && value != nil {
			return nil, false, fmt.Errorf("copilot MCP config servers must be an object")
		}
		servers = map[string]any{}
		root["servers"] = servers
	}
	current, present := servers[serverName]
	if present {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil, false, fmt.Errorf("copilot registration %q is not an object: %w", serverName, ErrConflict)
		}
		if copilotMatches(currentMap, req) {
			return data, true, nil
		}
		if !req.Replace {
			return nil, false, fmt.Errorf("%w: Copilot registration %q differs from requested command/profile", ErrConflict, serverName)
		}
	}
	servers[serverName] = copilotRegistration(req)
	return marshalConfig(root), false, nil
}

func inspectCopilot(data []byte) (string, string, bool, error) {
	root := map[string]any{}
	if err := json.Unmarshal(stripJSONComments(data), &root); err != nil {
		return "", "", false, err
	}
	servers, ok := root["servers"].(map[string]any)
	if !ok {
		return "", "", false, nil
	}
	registration, ok := servers[serverName].(map[string]any)
	if !ok {
		return "", "", false, fmt.Errorf("copilot registration %q is not an object", serverName)
	}
	binary, _ := registration["command"].(string)
	profile := profileFromArgs(registration["args"])
	return binary, profile, binary != "", nil
}

func copilotMatches(current map[string]any, req Request) bool {
	command, _ := current["command"].(string)
	if command != req.Binary {
		return false
	}
	args, ok := current["args"].([]any)
	if !ok {
		if strings, ok := current["args"].([]string); ok {
			return stringsEqual(strings, []string{"mcp", "--profile", req.Profile})
		}
		return false
	}
	expected := []string{"mcp", "--profile", req.Profile}
	if len(args) != len(expected) {
		return false
	}
	for index, value := range args {
		text, ok := value.(string)
		if !ok || text != expected[index] {
			return false
		}
	}
	return true
}

func profileFromArgs(value any) string {
	args, ok := value.([]any)
	if !ok {
		return ""
	}
	for index, item := range args {
		if flag, ok := item.(string); ok && flag == "--profile" && index+1 < len(args) {
			profile, _ := args[index+1].(string)
			return profile
		}
	}
	return ""
}

func removeCopilot(data []byte) ([]byte, bool, error) {
	root := map[string]any{}
	if err := json.Unmarshal(stripJSONComments(data), &root); err != nil {
		return nil, false, fmt.Errorf("parse Copilot MCP config: %w", err)
	}
	servers, ok := root["servers"].(map[string]any)
	if !ok {
		return data, false, nil
	}
	if _, ok := servers[serverName]; !ok {
		return data, false, nil
	}
	delete(servers, serverName)
	return marshalConfig(root), true, nil
}

func marshalConfig(root map[string]any) []byte {
	data, _ := json.MarshalIndent(root, "", "  ")
	return append(data, '\n')
}

func stringsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

// stripJSONComments accepts VS Code's JSONC input while keeping strings intact.
func stripJSONComments(data []byte) []byte {
	var out bytes.Buffer
	inString, escaped, lineComment, blockComment := false, false, false, false
	for index := 0; index < len(data); index++ {
		char := data[index]
		if lineComment {
			if char == '\n' {
				lineComment = false
				out.WriteByte(char)
			}
			continue
		}
		if blockComment {
			if char == '*' && index+1 < len(data) && data[index+1] == '/' {
				blockComment = false
				index++
			}
			continue
		}
		if inString {
			out.WriteByte(char)
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
		if char == '"' {
			inString = true
			out.WriteByte(char)
			continue
		}
		if char == '/' && index+1 < len(data) {
			switch data[index+1] {
			case '/':
				lineComment = true
				index++
				continue
			case '*':
				blockComment = true
				index++
				continue
			}
		}
		out.WriteByte(char)
	}
	return out.Bytes()
}
