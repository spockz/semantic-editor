// Package integration validates and edits Copilot CLI JSON configuration.
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func copilotRegistration(req Request) map[string]any {
	return map[string]any{
		"type":    "local",
		"command": req.Binary,
		"args":    []string{"mcp", "--profile", req.Profile},
		"tools":   []string{"*"},
	}
}

func currentCopilotRegistration(data []byte) ([]byte, error) {
	root := map[string]any{}
	if hasJSONComments(data) {
		return nil, fmt.Errorf("copilot MCP config contains comments")
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	servers, ok := root["mcpServers"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("mcpServers is not an object")
	}
	value, present := servers[serverName]
	if !present {
		return nil, fmt.Errorf("semedit registration not found")
	}
	registration, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("semedit registration is not an object")
	}
	return json.Marshal(registration)
}

func mergeCopilot(data []byte, req Request, exists bool) ([]byte, bool, error) {
	root := map[string]any{}
	if exists && len(bytes.TrimSpace(data)) > 0 {
		if hasJSONComments(data) {
			return nil, false, fmt.Errorf("copilot MCP config contains comments; JSON comments are not supported safely")
		}
		if err := json.Unmarshal(data, &root); err != nil {
			return nil, false, fmt.Errorf("parse Copilot MCP config: %w", err)
		}
	}
	servers, ok := root["mcpServers"].(map[string]any)
	if !ok {
		if value, present := root["mcpServers"]; present && value != nil {
			return nil, false, fmt.Errorf("copilot MCP config servers must be an object")
		}
		servers = map[string]any{}
		root["mcpServers"] = servers
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
	if hasJSONComments(data) {
		return "", "", false, fmt.Errorf("copilot MCP config contains comments; JSON comments are not supported safely")
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return "", "", false, err
	}
	servers, ok := root["mcpServers"].(map[string]any)
	if !ok {
		return "", "", false, nil
	}
	value, present := servers[serverName]
	if !present {
		return "", "", false, nil
	}
	registration, ok := value.(map[string]any)
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
	if typ, _ := current["type"].(string); typ != "local" {
		return false
	}
	tools, ok := current["tools"].([]any)
	if !ok || len(tools) != 1 || tools[0] != "*" {
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
	if hasJSONComments(data) {
		return nil, false, fmt.Errorf("copilot MCP config contains comments; JSON comments are not supported safely")
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, false, fmt.Errorf("parse Copilot MCP config: %w", err)
	}
	servers, ok := root["mcpServers"].(map[string]any)
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

func hasJSONComments(data []byte) bool {
	inString, escaped := false, false
	for index := 0; index+1 < len(data); index++ {
		char := data[index]
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
		if char == '"' {
			inString = true
			continue
		}
		if char == '/' && (data[index+1] == '/' || data[index+1] == '*') {
			return true
		}
	}
	return false
}
