// This file renders tracked development config templates to machine-local absolute executable paths.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spockz/semantic-editor/internal/atomicfile"
)

const (
	semeditPlaceholder     = "{{SEMEDIT_BINARY}}"
	semeditNextPlaceholder = "{{SEMEDIT_NEXT_BINARY}}"
)

type configFile struct {
	template string
	output   string
	format   string
}

var configFiles = []configFile{
	{template: ".codex/config.toml.example", output: ".codex/config.toml", format: "toml"},
	{template: ".agents/plugins/workspace-servers/mcp_config.json.example", output: ".agents/plugins/workspace-servers/mcp_config.json", format: "json"},
}

func install(root string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	semeditPath := filepath.Join(root, "bin", "semedit")
	semeditNextPath := filepath.Join(root, "bin", "semedit-next")
	if binaryErr := validateBinary(semeditPath); binaryErr != nil {
		return binaryErr
	}
	if binaryErr := validateBinary(semeditNextPath); binaryErr != nil {
		return binaryErr
	}
	outputs := make([]*atomicfile.StagedFile, 0, len(configFiles))
	defer func() {
		for _, output := range outputs {
			output.Discard()
		}
	}()
	for _, file := range configFiles {
		templatePath := filepath.Join(root, filepath.FromSlash(file.template))
		// #nosec G304 -- file.template is a fixed repo-relative name under the explicit workspace root.
		data, readErr := os.ReadFile(templatePath)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", file.template, readErr)
		}
		data, renderErr := render(file.format, data, semeditPath, semeditNextPath)
		if renderErr != nil {
			return fmt.Errorf("render %s: %w", file.template, renderErr)
		}
		outputPath := filepath.Join(root, filepath.FromSlash(file.output))
		if directoryErr := os.MkdirAll(filepath.Dir(outputPath), 0o750); directoryErr != nil {
			return fmt.Errorf("create config directory for %s: %w", file.output, directoryErr)
		}
		staged, stageErr := atomicfile.Stage(outputPath, data)
		if stageErr != nil {
			return fmt.Errorf("stage %s: %w", file.output, stageErr)
		}
		outputs = append(outputs, staged)
	}
	for i, output := range outputs {
		if publishErr := output.Publish(); publishErr != nil {
			return fmt.Errorf("publish %s: %w", configFiles[i].output, publishErr)
		}
	}
	return nil
}

func validateBinary(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required executable %s: %w", path, err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("required executable %s is not a regular executable file", path)
	}
	return nil
}

func render(format string, source []byte, semeditPath, semeditNextPath string) ([]byte, error) {
	switch format {
	case "toml":
		return renderTOML(source, semeditPath, semeditNextPath)
	case "json":
		return renderJSON(source, semeditPath, semeditNextPath)
	default:
		return nil, fmt.Errorf("unsupported config format %q", format)
	}
}

func renderTOML(source []byte, semeditPath, semeditNextPath string) ([]byte, error) {
	var config map[string]any
	if err := toml.Unmarshal(source, &config); err != nil {
		return nil, fmt.Errorf("parse TOML: %w", err)
	}
	servers, ok := config["mcp_servers"].(map[string]any)
	if !ok {
		return nil, errors.New("TOML config has no mcp_servers table")
	}
	for name, value := range servers {
		server, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("TOML server %q is not a table", name)
		}
		command, ok := server["command"].(string)
		if !ok || command == "" {
			return nil, fmt.Errorf("TOML server %q has no string command", name)
		}
		resolved, err := resolveCommand(command, semeditPath, semeditNextPath)
		if err != nil {
			return nil, fmt.Errorf("TOML server %q: %w", name, err)
		}
		server["command"] = resolved
	}
	var output bytes.Buffer
	if err := toml.NewEncoder(&output).Encode(config); err != nil {
		return nil, fmt.Errorf("encode TOML: %w", err)
	}
	return output.Bytes(), nil
}

func renderJSON(source []byte, semeditPath, semeditNextPath string) ([]byte, error) {
	var config map[string]json.RawMessage
	if err := json.Unmarshal(source, &config); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(config["mcpServers"], &servers); err != nil {
		return nil, fmt.Errorf("parse mcpServers: %w", err)
	}
	if len(servers) == 0 {
		return nil, errors.New("JSON config has no mcpServers entries")
	}
	for name, raw := range servers {
		if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return nil, fmt.Errorf("server %q is not an object", name)
		}
		var server map[string]json.RawMessage
		if err := json.Unmarshal(raw, &server); err != nil {
			return nil, fmt.Errorf("parse server %q: %w", name, err)
		}
		var command string
		commandData := bytes.TrimSpace(server["command"])
		if len(commandData) == 0 || bytes.Equal(commandData, []byte("null")) {
			return nil, fmt.Errorf("server %q has no string command", name)
		}
		if err := json.Unmarshal(commandData, &command); err != nil {
			return nil, fmt.Errorf("server %q has no string command: %w", name, err)
		}
		if command == "" {
			return nil, fmt.Errorf("server %q has an empty command", name)
		}
		command, err := resolveCommand(command, semeditPath, semeditNextPath)
		if err != nil {
			return nil, fmt.Errorf("server %q: %w", name, err)
		}
		updatedCommand, err := json.Marshal(command)
		if err != nil {
			return nil, fmt.Errorf("encode server %q command: %w", name, err)
		}
		server["command"] = updatedCommand
		updatedServer, err := json.Marshal(server)
		if err != nil {
			return nil, fmt.Errorf("encode server %q: %w", name, err)
		}
		servers[name] = updatedServer
	}
	updatedServers, err := json.Marshal(servers)
	if err != nil {
		return nil, fmt.Errorf("encode mcpServers: %w", err)
	}
	config["mcpServers"] = updatedServers
	output, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode JSON: %w", err)
	}
	return append(output, '\n'), nil
}

func resolveCommand(command, semeditPath, semeditNextPath string) (string, error) {
	switch command {
	case semeditPlaceholder:
		return semeditPath, nil
	case semeditNextPlaceholder:
		return semeditNextPath, nil
	default:
		if hasPlaceholder(command) {
			return "", errors.New("unresolved command placeholder")
		}
		return command, nil
	}
}

func hasPlaceholder(command string) bool {
	return strings.Contains(command, "{{") || strings.Contains(command, "}}")
}
