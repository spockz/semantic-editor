// Package cli tests that registered source payload flags expand from Cobra stdin before dispatch.

package cli

import (
	"context"
	"errors"
	"io"
	"semedit/internal/backend"
	"semedit/internal/operation"
	"strings"
	"testing"
)

func (r *trackingReader) Read(buffer []byte) (int, error) {
	r.reads++
	if r.input != "" {
		n := copy(buffer, r.input)
		r.input = r.input[n:]
		return n, nil
	}
	return 0, r.err
}

func TestBuildCommandReadsSourcePayloadAndLeavesOtherDashLiteral(t *testing.T) {
	registry, entry, state := newStdinProbe(t)
	payload := `const Quoted = "quote \\ path"` + "\n"
	reader := &trackingReader{input: payload, err: io.EOF}
	cmd := buildCommand(t.TempDir(), registry, entry)
	cmd.SetArgs([]string{"--source", "-", "--body", "body literal", "--case", "case literal", "--symbol", "-"})
	cmd.SetIn(reader)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute probe: %v", err)
	}
	if !state.dispatched || !state.mutated {
		t.Fatalf("dispatch state = %#v, want called handler", state)
	}
	if state.source != payload {
		t.Fatalf("source payload = %q, want exact stdin bytes %q", state.source, payload)
	}
	if state.body != "body literal" || state.casePayload != "case literal" || state.symbol != "-" {
		t.Fatalf("literal flags changed: body=%q case=%q symbol=%q", state.body, state.casePayload, state.symbol)
	}
}

func TestBuildCommandReadErrorStopsBeforeDispatch(t *testing.T) {
	registry, entry, state := newStdinProbe(t)
	readFailure := errors.New("stdin read failed")
	reader := &trackingReader{input: `const Captured = "valid partial input"` + "\n", err: readFailure}
	cmd := buildCommand(t.TempDir(), registry, entry)
	cmd.SetArgs([]string{"--source", "-", "--body", "literal", "--case", "literal"})
	cmd.SetIn(reader)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	if !errors.Is(err, readFailure) || !strings.Contains(err.Error(), "read --source from stdin") {
		t.Fatalf("execute error = %v, want contextual stdin read error", err)
	}
	if state.parsed || state.dispatched || state.mutated {
		t.Fatalf("dispatch state = %#v, want no parse, dispatch, or mutation", state)
	}
	if reader.reads < 2 {
		t.Fatalf("stdin read calls = %d, want payload followed by read error", reader.reads)
	}
}

func TestBuildCommandConflictingPayloadFlagsDoNotReadOrDispatch(t *testing.T) {
	registry, entry, state := newStdinProbe(t)
	reader := &trackingReader{input: "must remain unread", err: io.EOF}
	cmd := buildCommand(t.TempDir(), registry, entry)
	cmd.SetArgs([]string{"--source", "-", "--body", "-", "--case", "-"})
	cmd.SetIn(reader)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "multiple stdin payload flags --source and --body and --case") {
		t.Fatalf("execute error = %v, want all conflicting flag names", err)
	}
	if reader.reads != 0 || state.parsed || state.dispatched || state.mutated {
		t.Fatalf("reader/dispatch state = reads:%d %#v, want no read, parse, or dispatch", reader.reads, state)
	}
}

func TestBuildCommandReadsCasePayload(t *testing.T) {
	registry, entry, state := newStdinProbe(t)
	payload := `case "quoted" // \\` + "\n"
	reader := &trackingReader{input: payload, err: io.EOF}
	cmd := buildCommand(t.TempDir(), registry, entry)
	cmd.SetArgs([]string{"--source", "source literal", "--body", "body literal", "--case", "-", "--symbol", "-"})
	cmd.SetIn(reader)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute probe: %v", err)
	}
	if state.casePayload != payload {
		t.Fatalf("case payload = %q, want exact stdin bytes %q", state.casePayload, payload)
	}
	if !state.parsed || !state.dispatched || !state.mutated || state.symbol != "-" {
		t.Fatalf("dispatch state = %#v, want one literal selector and called handler", state)
	}
}

func TestBuildCommandDoesNotReadStdinWithoutPayloadMarker(t *testing.T) {
	registry, entry, state := newStdinProbe(t)
	reader := &trackingReader{input: "leave this unread", err: io.EOF}
	cmd := buildCommand(t.TempDir(), registry, entry)
	cmd.SetArgs([]string{"--source", "source literal", "--body", "body literal", "--case", "case literal", "--symbol", "-"})
	cmd.SetIn(reader)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute probe: %v", err)
	}
	if reader.reads != 0 || !state.parsed || !state.dispatched || !state.mutated || state.symbol != "-" {
		t.Fatalf("reader/dispatch state = reads:%d %#v, want no read and literal symbol", reader.reads, state)
	}
}

func TestBuildCommandPreservesEmptyStdinPayload(t *testing.T) {
	registry, entry, state := newStdinProbe(t)
	reader := &trackingReader{err: io.EOF}
	cmd := buildCommand(t.TempDir(), registry, entry)
	cmd.SetArgs([]string{"--source", "-", "--body", "body literal", "--case", "case literal"})
	cmd.SetIn(reader)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute probe: %v", err)
	}
	if !state.sourcePresent || state.source != "" || !state.parsed || !state.dispatched || reader.reads != 1 {
		t.Fatalf("empty input state = reads:%d %#v, want explicit empty source", reader.reads, state)
	}
}

type stdinProbeState struct {
	dispatched    bool
	mutated       bool
	parsed        bool
	sourcePresent bool
	source        string
	body          string
	casePayload   string
	symbol        string
}

type trackingReader struct {
	input string
	err   error
	reads int
}

func newStdinProbe(t *testing.T) (*operation.Registry, operation.Entry, *stdinProbeState) {
	t.Helper()
	state := &stdinProbeState{}
	params := []operation.ParameterContract{
		{Name: "source", JSONName: "source", CLIName: "source", Type: operation.ParamString},
		{Name: "body", JSONName: "body", CLIName: "body", Type: operation.ParamString, Required: true},
		{Name: "case", JSONName: "case", CLIName: "case", Type: operation.ParamString, Required: true},
		{Name: "symbol", JSONName: "symbol", CLIName: "symbol", Type: operation.ParamString},
	}
	registry := operation.NewRegistry()
	if err := operation.Register(registry, operation.Def[map[string]any, string]{
		Key:     "stdin_probe",
		CLIName: "stdin-probe",
		Params:  params,
		Parse: func(raw map[string]any) (map[string]any, error) {
			state.parsed = true
			_, state.sourcePresent = raw["source"]
			return raw, nil
		},
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, map[string]any) (string, error){
			backend.LanguageAuto: func(_ context.Context, _ operation.CallContext, raw map[string]any) (string, error) {
				state.dispatched = true
				state.mutated = true
				state.source, _ = raw["source"].(string)
				state.body, _ = raw["body"].(string)
				state.casePayload, _ = raw["case"].(string)
				state.symbol, _ = raw["symbol"].(string)
				return "dispatched", nil
			},
		},
	}); err != nil {
		t.Fatalf("register probe: %v", err)
	}
	entry, ok := registry.LookupCLI("stdin-probe")
	if !ok {
		t.Fatal("probe command was not registered")
	}
	return registry, entry, state
}
