// Package suggestions retrieves bounded candidate declarations for native validation.
package suggestions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	defaultCodannaTimeout = 2 * time.Second
	maximumCodannaTimeout = 30 * time.Second
	defaultOutputLimit    = 2 << 20
	maximumOutputLimit    = 16 << 20
)

var errOutputLimit = errors.New("Codanna output exceeded configured limit")

// Codanna queries an already installed CLI. It never invokes indexing commands.
type Codanna struct {
	Binary         string
	Args           []string
	Dir            string
	Timeout        time.Duration
	MaxOutputBytes int
}

// Retrieve searches the configured CLI within time and output bounds, leaving applicability to native validation.
func (provider Codanna) Retrieve(ctx context.Context, query Query) ([]Candidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(query.Selector) == "" {
		return []Candidate{}, nil
	}
	binary := provider.Binary
	if binary == "" {
		binary = "codanna"
	}
	timeout := provider.Timeout
	if timeout <= 0 || timeout > maximumCodannaTimeout {
		timeout = defaultCodannaTimeout
	}
	outputLimit := provider.MaxOutputBytes
	if outputLimit <= 0 {
		outputLimit = defaultOutputLimit
	} else if outputLimit > maximumOutputLimit {
		outputLimit = maximumOutputLimit
	}
	limit := query.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maximumLimit {
		limit = maximumLimit
	}
	searchLimit := min(maximumLimit, max(limit*4, limit))
	args := append([]string(nil), provider.Args...)
	args = append(args, "mcp", "search_symbols", "query:"+query.Selector, fmt.Sprintf("limit:%d", searchLimit), "--json")
	commandContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	// #nosec G204 -- binary is configured or PATH-resolved tooling; arguments are passed without a shell.
	command := exec.CommandContext(commandContext, binary, args...)
	command.Dir = provider.Dir
	command.WaitDelay = 50 * time.Millisecond
	stdout := &limitedBuffer{limit: outputLimit}
	stderr := &limitedBuffer{limit: min(outputLimit, 64<<10)}
	command.Stdout, command.Stderr = stdout, stderr
	err := command.Run()
	if stdout.err != nil || stderr.err != nil {
		return nil, errOutputLimit
	}
	if err != nil {
		if commandContext.Err() != nil {
			return nil, commandContext.Err()
		}
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return nil, fmt.Errorf("Codanna search_symbols: %w: %s", err, message)
		}
		return nil, fmt.Errorf("Codanna search_symbols: %w", err)
	}
	var payload any
	if decodeErr := json.Unmarshal(stdout.Bytes(), &payload); decodeErr != nil {
		return nil, fmt.Errorf("decode Codanna search_symbols output: %w", decodeErr)
	}
	candidates, err := decodeCodannaCandidates(payload)
	if err != nil {
		return nil, err
	}
	query.Candidates = candidates
	return (Matcher{}).Retrieve(ctx, query)
}

type limitedBuffer struct {
	data  []byte
	limit int
	err   error
}

func (buffer *limitedBuffer) Bytes() []byte  { return buffer.data }
func (buffer *limitedBuffer) String() string { return string(buffer.data) }
func (buffer *limitedBuffer) Len() int       { return len(buffer.data) }

func (buffer *limitedBuffer) Write(data []byte) (int, error) {
	original := len(data)
	remaining := buffer.limit - len(buffer.data)
	if remaining <= 0 {
		buffer.err = errOutputLimit
		return 0, errOutputLimit
	}
	if len(data) > remaining {
		buffer.data = append(buffer.data, data[:remaining]...)
		buffer.err = errOutputLimit
		return 0, errOutputLimit
	}
	buffer.data = append(buffer.data, data...)
	return original, nil
}

func decodeCodannaCandidates(payload any) ([]Candidate, error) {
	rows := collectCodannaCandidates(payload)
	if rows == nil {
		return nil, errors.New("Codanna search_symbols returned malformed candidates")
	}
	result := make([]Candidate, 0, len(rows))
	for _, row := range rows {
		candidate := codannaCandidate(row)
		if candidate.Name != "" {
			result = append(result, candidate)
		}
	}
	return result, nil
}

func collectCodannaCandidates(value any) []any {
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	for _, key := range []string{"results", "symbols", "matches", "data"} {
		if rows, ok := object[key].([]any); ok {
			return rows
		}
	}
	for _, key := range []string{"result", "response"} {
		if nested, ok := object[key]; ok {
			if rows := collectCodannaCandidates(nested); rows != nil {
				return rows
			}
		}
	}
	if rows, ok := value.([]any); ok {
		return rows
	}
	return nil
}

func codannaCandidate(value any) Candidate {
	row, ok := value.(map[string]any)
	if !ok {
		return Candidate{}
	}
	if nested, ok := row["symbol"]; ok {
		if parsed := codannaCandidate(nested); parsed.Name != "" {
			return parsed
		}
	}
	file := firstStringField(row, "file", "file_path", "path", "source_file")
	name := firstStringField(row, "name", "symbol_name", "identifier")
	selector := firstStringField(row, "selector", "qualified_name", "full_name")
	if selector == "" {
		selector = name
	}
	return Candidate{Selector: selector, File: file, Name: name, Kind: firstStringField(row, "kind", "symbol_kind", "type"), Signature: firstStringField(row, "signature", "display_signature"), Identity: scalarField(row, "id", "symbol_id"), Language: firstStringField(row, "language")}
}

func firstStringField(row map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := row[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}

func scalarField(row map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := row[key]; ok {
			switch typed := value.(type) {
			case string:
				return typed
			case float64:
				return fmt.Sprintf("%g", typed)
			}
		}
	}
	return ""
}
