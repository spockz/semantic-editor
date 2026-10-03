// Package mcp_test proves AST and gopls share unsaved revisions while disk readers see the preimage.
package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"semedit/internal/backend"
	"semedit/internal/mcp"
	"semedit/internal/operation"
	"semedit/internal/pipeline"
	"semedit/internal/sourcebuf"
	"strings"
	"testing"
)

func TestMCPBatchSharesUnsavedGoBuffers(t *testing.T) {
	fixture := newManagedMCPFixture(t)
	originals := bufferedGoSources()
	identities := writeBufferedGoSources(t, fixture.root, originals)
	registry := operation.DefaultRegistry()
	probes := 0
	registerUnsavedBufferProbe(t, registry, fixture.root, originals, identities, &probes)
	output := executeUnsavedBufferBatch(t, fixture.root, registry)
	assertSuccessfulMCPResponses(t, output)
	if probes != 3 {
		t.Fatalf("buffer probes=%d", probes)
	}
	assertUnsavedBufferSources(t, fixture.root)
	assertSingleSessionStart(t, fixture.counter)
}

func bufferedGoSources() map[string]string {
	return map[string]string{
		"value.go": "package sample\n\nfunc Value() int { return 1 }\n",
		"uses.go":  "package sample\n",
	}
}

func writeBufferedGoSources(t *testing.T, root string, originals map[string]string) map[string]os.FileInfo {
	t.Helper()
	identities := make(map[string]os.FileInfo, len(originals))
	for file, source := range originals {
		path := filepath.Join(root, file)
		if err := pipeline.WriteAtomic(path, []byte(source)); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		identities[file] = info
	}
	return identities
}

func registerUnsavedBufferProbe(t *testing.T, registry *operation.Registry, root string, originals map[string]string, identities map[string]os.FileInfo, probes *int) {
	t.Helper()
	definition := operation.Def[map[string]any, operation.FileEditRes]{
		Key: "probe_buffers", MCPName: "semantic_probe_buffers", Batchable: true, BufferedBatch: true,
		Params: []operation.ParameterContract{
			{Name: "file", JSONName: "file", CLIName: "file", Type: operation.ParamString, Required: true},
			{Name: "expected", JSONName: "expected", CLIName: "expected", Type: operation.ParamString, Required: true},
		},
		Parse: func(raw map[string]any) (map[string]any, error) { return raw, nil },
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, map[string]any) (operation.FileEditRes, error){
			backend.LanguageGo: func(ctx context.Context, _ operation.CallContext, raw map[string]any) (operation.FileEditRes, error) {
				return observeUnsavedBuffer(ctx, t, root, originals, identities, raw, probes)
			},
		},
		Format: func(operation.FileEditRes) (string, error) { return "observed", nil },
	}
	if err := operation.Register(registry, definition); err != nil {
		t.Fatal(err)
	}
}

func observeUnsavedBuffer(ctx context.Context, t *testing.T, root string, originals map[string]string, identities map[string]os.FileInfo, raw map[string]any, probes *int) (operation.FileEditRes, error) {
	t.Helper()
	if err := assertDiskPreimage(root, originals, identities); err != nil {
		return operation.FileEditRes{}, err
	}
	file, _ := raw["file"].(string)
	expected, _ := raw["expected"].(string)
	visible, err := sourcebuf.ReadFile(ctx, filepath.Join(root, file))
	if err != nil {
		return operation.FileEditRes{}, err
	}
	if !strings.Contains(string(visible), expected) {
		return operation.FileEditRes{}, fmt.Errorf("buffer %s lacks %q: %s", file, expected, visible)
	}
	*probes++
	return operation.FileEditRes{}, nil
}

func assertDiskPreimage(root string, originals map[string]string, identities map[string]os.FileInfo) error {
	for file, original := range originals {
		path := filepath.Join(root, file)
		disk, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if string(disk) != original || !os.SameFile(identities[file], info) || !info.ModTime().Equal(identities[file].ModTime()) {
			return fmt.Errorf("intermediate edit published %s", file)
		}
	}
	return nil
}

func executeUnsavedBufferBatch(t *testing.T, root string, registry *operation.Registry) string {
	t.Helper()
	var input, output bytes.Buffer
	encoder := json.NewEncoder(&input)
	for _, request := range unsavedBufferRequests() {
		if err := encoder.Encode(request); err != nil {
			t.Fatal(err)
		}
	}
	if err := mcp.NewServer("full", root, &output, mcp.WithRegistry(registry)).Serve(t.Context(), &input); err != nil {
		t.Fatal(err)
	}
	return output.String()
}

func unsavedBufferRequests() []map[string]any {
	return []map[string]any{
		{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}},
		{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": "semantic_batch", "arguments": map[string]any{"edits": unsavedBufferEdits()}}},
	}
}

func unsavedBufferEdits() []map[string]any {
	return []map[string]any{
		{"tool": "semantic_insert_function", "params": map[string]any{"file": "uses.go", "source": "func Read() int { return Value() }"}},
		{"tool": "semantic_probe_buffers", "params": map[string]any{"file": "uses.go", "expected": "return Value()"}},
		{"tool": "semantic_rename", "params": map[string]any{"file": "value.go", "symbol": "Value", "to": "Renamed"}},
		{"tool": "semantic_probe_buffers", "params": map[string]any{"file": "uses.go", "expected": "return Renamed()"}},
		{"tool": "semantic_replace_body", "params": map[string]any{"file": "value.go", "symbol": "Renamed", "body": "return 3"}},
		{"tool": "semantic_rename", "params": map[string]any{"file": "value.go", "symbol": "Renamed", "to": "Final"}},
		{"tool": "semantic_probe_buffers", "params": map[string]any{"file": "uses.go", "expected": "return Final()"}},
	}
}

func assertSuccessfulMCPResponses(t *testing.T, output string) {
	t.Helper()
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		var response struct {
			Error  json.RawMessage `json:"error"`
			Result struct {
				IsError bool `json:"isError"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatal(err)
		}
		if response.Error != nil || response.Result.IsError {
			t.Fatalf("response=%s", line)
		}
	}
}

func assertUnsavedBufferSources(t *testing.T, root string) {
	t.Helper()
	for file, fragments := range map[string][]string{
		"uses.go":  {"func Read()", "return Final()"},
		"value.go": {"func Final()", "return 3"},
	} {
		disk, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range fragments {
			if !strings.Contains(string(disk), fragment) {
				t.Fatalf("published %s lacks %q: %s", file, fragment, disk)
			}
		}
	}
}

func assertSingleSessionStart(t *testing.T, counter string) {
	t.Helper()
	starts, err := os.ReadFile(counter)
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(starts), "start\n"); count != 1 {
		t.Fatalf("gopls starts=%d, want 1", count)
	}
}

func TestBatchBufferDiskBarrierOrdering(t *testing.T) {
	fixture := newManagedMCPFixture(t)
	registry := operation.DefaultRegistry()
	barriers := 0
	if err := operation.Register(registry, operation.Def[struct{}, operation.FileEditRes]{Key: "probe_disk", MCPName: "semantic_probe_disk", Batchable: true, Parse: func(map[string]any) (struct{}, error) { return struct{}{}, nil }, Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, struct{}) (operation.FileEditRes, error){backend.LanguageGo: func(ctx context.Context, _ operation.CallContext, _ struct{}) (operation.FileEditRes, error) {
		if sourcebuf.FromContext(ctx) != nil {
			return operation.FileEditRes{}, fmt.Errorf("disk barrier retained source overlay")
		}
		disk, err := os.ReadFile(filepath.Join(fixture.root, "value.go"))
		if err != nil {
			return operation.FileEditRes{}, err
		}
		if !strings.Contains(string(disk), "return 2") {
			return operation.FileEditRes{}, fmt.Errorf("barrier lacks published preceding edit: %s", disk)
		}
		barriers++
		return operation.FileEditRes{}, nil
	}}, Format: func(operation.FileEditRes) (string, error) { return "observed", nil }}); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	srv := mcp.NewServer("full", fixture.root, &output, mcp.WithRegistry(registry))
	response, err := srv.ExecuteBatch(t.Context(), []mcp.BatchEntry{{Tool: "semantic_replace_body", Params: json.RawMessage(`{"file":"value.go","symbol":"Value","body":"return 2"}`)}, {Tool: "semantic_probe_disk", Params: json.RawMessage(`{}`)}, {Tool: "semantic_replace_body", Params: json.RawMessage(`{"file":"value.go","symbol":"Value","body":"return 3"}`)}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != "ok" || barriers != 1 {
		t.Fatalf("response=%+v, barriers=%d", response, barriers)
	}
	disk, err := os.ReadFile(filepath.Join(fixture.root, "value.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(disk), "return 3") {
		t.Fatalf("last segment=%s", disk)
	}
	starts, err := os.ReadFile(fixture.counter)
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(starts), "start\n"); count != 1 {
		t.Fatalf("gopls starts=%d, want 1", count)
	}
}

func TestBatchSubdirectoryRenameAcrossBarrier(t *testing.T) {
	fixture := newManagedMCPFixture(t)
	subdir := filepath.Join(fixture.root, "child")
	if err := os.MkdirAll(subdir, 0o700); err != nil {
		t.Fatal(err)
	}
	// The declaration is in the child package; its caller is outside the MCP workspace.
	if err := pipeline.WriteAtomic(filepath.Join(fixture.root, "go.mod"), []byte("module example.com/subdir\n\ngo 1.24\n")); err != nil {
		t.Fatal(err)
	}
	if err := pipeline.WriteAtomic(filepath.Join(subdir, "value.go"), []byte("package child\n\nfunc Value() int { return 1 }\n")); err != nil {
		t.Fatal(err)
	}
	if err := pipeline.WriteAtomic(filepath.Join(fixture.root, "value.go"), []byte("package sample\nimport \"example.com/subdir/child\"\nfunc Read()int{return child.Value()}\n")); err != nil {
		t.Fatal(err)
	}
	registry := operation.DefaultRegistry()
	if err := operation.Register(registry, operation.Def[struct{}, operation.FileEditRes]{Key: "probe_module_disk", MCPName: "semantic_probe_module_disk", Batchable: true, Parse: func(map[string]any) (struct{}, error) { return struct{}{}, nil }, Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, struct{}) (operation.FileEditRes, error){backend.LanguageGo: func(ctx context.Context, _ operation.CallContext, _ struct{}) (operation.FileEditRes, error) {
		if sourcebuf.FromContext(ctx) != nil {
			return operation.FileEditRes{}, fmt.Errorf("barrier retained buffer")
		}
		data, err := os.ReadFile(filepath.Join(fixture.root, "value.go"))
		if err != nil {
			return operation.FileEditRes{}, err
		}
		if !strings.Contains(string(data), "child.Renamed()") {
			return operation.FileEditRes{}, fmt.Errorf("external rename not published before barrier: %s", data)
		}
		return operation.FileEditRes{}, nil
	}}, Format: func(operation.FileEditRes) (string, error) { return "observed", nil }}); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	response, err := mcp.NewServer("full", subdir, &output, mcp.WithRegistry(registry)).ExecuteBatch(t.Context(), []mcp.BatchEntry{
		{Tool: "semantic_rename", Params: json.RawMessage(`{"file":"value.go","symbol":"Value","to":"Renamed"}`)},
		{Tool: "semantic_probe_module_disk", Params: json.RawMessage(`{}`)},
		{Tool: "semantic_replace_body", Params: json.RawMessage(`{"file":"value.go","symbol":"Renamed","body":"return 3"}`)},
		{Tool: "semantic_rename", Params: json.RawMessage(`{"file":"value.go","symbol":"Renamed","to":"Final"}`)},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != "ok" {
		t.Fatalf("response=%+v", response)
	}
	for _, fragment := range []string{"../value.go", "child.Final()", "return 3"} {
		if !strings.Contains(response.FinalDiff, fragment) {
			t.Fatalf("final diff lacks %q: %s", fragment, response.FinalDiff)
		}
	}
	external, err := os.ReadFile(filepath.Join(fixture.root, "value.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(external) != "package sample\n\nimport \"example.com/subdir/child\"\n\nfunc Read() int { return child.Final() }\n" {
		t.Fatalf("external formatting=%s", external)
	}
	starts, err := os.ReadFile(fixture.counter)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(starts), "start\n") != 1 {
		t.Fatalf("gopls starts=%s", starts)
	}
}

func TestBatchMovesAndRenameShareUnsavedModule(t *testing.T) {
	fixture := newManagedMCPFixture(t)
	originals := map[string]string{
		"legacy/source.go": "package legacy\nimport \"strings\"\nfunc MoveMe(value string) string { return strings.TrimSpace(value) }\nfunc Keep() int { return 1 }\n",
		"legacy/spare.go":  "package legacy\n",
		"api/dest.go":      "package api\nimport \"example.com/mcpreuse/legacy\"\nfunc Existing() string { return legacy.MoveMe(\" existing \") }\n",
		"caller/use.go":    "package caller\nimport \"example.com/mcpreuse/legacy\"\nfunc Call() string { return legacy.MoveMe(\" caller \") }\n",
	}
	for file, data := range originals {
		path := filepath.Join(fixture.root, file)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := pipeline.WriteAtomic(path, []byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	registry := operation.DefaultRegistry()
	probes := 0
	if err := registerMoveBufferProbe(registry, fixture.root, originals, &probes); err != nil {
		t.Fatal(err)
	}
	edits := []map[string]any{
		{"tool": "semantic_insert_function", "params": map[string]any{"file": "caller/use.go", "source": "func Unsaved() string { return legacy.MoveMe(\" unsaved \") }"}},
		{"tool": "semantic_move_construct", "params": map[string]any{"file": "legacy/source.go", "destination_file": "legacy/spare.go", "symbol": "MoveMe"}},
		{"tool": "semantic_probe_move_buffers", "params": map[string]any{}},
		{"tool": "semantic_move_construct", "params": map[string]any{"file": "legacy/spare.go", "destination_file": "api/dest.go", "symbol": "MoveMe"}},
		{"tool": "semantic_probe_move_buffers", "params": map[string]any{}},
		{"tool": "semantic_rename", "params": map[string]any{"file": "api/dest.go", "symbol": "MoveMe", "to": "Final"}},
		{"tool": "semantic_replace_body", "params": map[string]any{"file": "api/dest.go", "symbol": "Final", "body": "return strings.TrimSpace(value) + \"!\""}},
		{"tool": "semantic_probe_move_buffers", "params": map[string]any{}},
	}
	var input, output bytes.Buffer
	encoder := json.NewEncoder(&input)
	requests := []map[string]any{
		{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "semantic_rename", "arguments": map[string]any{"file": "value.go", "symbol": "Value", "to": "Warm"}}},
		{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": "semantic_batch", "arguments": map[string]any{"edits": edits}}},
	}
	for _, request := range requests {
		if err := encoder.Encode(request); err != nil {
			t.Fatal(err)
		}
	}
	if err := mcp.NewServer("full", fixture.root, &output, mcp.WithRegistry(registry)).Serve(t.Context(), &input); err != nil {
		t.Fatal(err)
	}
	for line := range strings.SplitSeq(strings.TrimSpace(output.String()), "\n") {
		var response struct {
			Error  json.RawMessage `json:"error"`
			Result struct {
				IsError bool `json:"isError"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatal(err)
		}
		if response.Error != nil || response.Result.IsError {
			t.Fatalf("response=%s", line)
		}
	}
	if probes != 3 {
		t.Fatalf("buffer probes=%d", probes)
	}
	for file, fragments := range map[string][]string{"api/dest.go": {"func Final(", "return Final(\" existing \")", "+ \"!\""}, "caller/use.go": {"func Unsaved()", "api.Final(\" unsaved \")", "api.Final(\" caller \")"}, "legacy/source.go": {"func Keep()"}} {
		data, err := os.ReadFile(filepath.Join(fixture.root, file))
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range fragments {
			if !strings.Contains(string(data), fragment) {
				t.Fatalf("%s lacks %q: %s", file, fragment, data)
			}
		}
	}
	starts, err := os.ReadFile(fixture.counter)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(starts), "start\n") != 1 {
		t.Fatalf("gopls restarted during content-only moves: %s", starts)
	}
}

func registerMoveBufferProbe(registry *operation.Registry, root string, originals map[string]string, probes *int) error {
	definition := operation.Def[struct{}, operation.FileEditRes]{
		Key: "probe_move_buffers", MCPName: "semantic_probe_move_buffers", Batchable: true, BufferedBatch: true,
		Parse: func(map[string]any) (struct{}, error) { return struct{}{}, nil },
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, struct{}) (operation.FileEditRes, error){
			backend.LanguageGo: func(ctx context.Context, _ operation.CallContext, _ struct{}) (operation.FileEditRes, error) {
				if sourcebuf.FromContext(ctx) == nil {
					return operation.FileEditRes{}, fmt.Errorf("move published at a barrier")
				}
				for file, original := range originals {
					data, err := os.ReadFile(filepath.Join(root, file))
					if err != nil {
						return operation.FileEditRes{}, err
					}
					if string(data) != original {
						return operation.FileEditRes{}, fmt.Errorf("move published %s before batch completion", file)
					}
				}
				*probes++
				return operation.FileEditRes{}, nil
			},
		},
		Format: func(operation.FileEditRes) (string, error) { return "observed", nil },
	}
	return operation.Register(registry, definition)
}

func TestMCPBatchSignatureKeepsDiskPreimageUntilFlush(t *testing.T) {
	fixture := newManagedMCPFixture(t)
	originals := map[string]string{
		"value.go": "package sample\n\nfunc value() int { return 1 }\n",
		"uses.go":  "package sample\n\nfunc Read() int { return value() }\n",
	}
	identities := make(map[string]os.FileInfo, len(originals))
	for file, content := range originals {
		path := filepath.Join(fixture.root, file)
		if err := pipeline.WriteAtomic(path, []byte(content)); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		identities[file] = info
	}
	registry := operation.DefaultRegistry()
	probes := 0
	registerSignatureBufferProbe(t, registry, fixture.root, originals, identities, &probes)
	edits := []map[string]any{
		{"tool": "semantic_change_signature", "params": map[string]any{
			"file": "value.go", "symbol": "value", "parameters": []any{
				map[string]any{"parameter": map[string]any{"name": "mode", "type": "int"}, "argument": "7"},
			},
		}},
		{"tool": "semantic_probe_signature_buffer", "params": map[string]any{}},
	}
	var input, output bytes.Buffer
	encoder := json.NewEncoder(&input)
	requests := []map[string]any{
		{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"initializationOptions": map[string]any{"trust_workspace": true}}},
		{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": "semantic_batch", "arguments": map[string]any{"edits": edits}}},
	}
	for _, request := range requests {
		if err := encoder.Encode(request); err != nil {
			t.Fatal(err)
		}
	}
	if err := mcp.NewServer("full", fixture.root, &output, mcp.WithRegistry(registry)).Serve(t.Context(), &input); err != nil {
		t.Fatal(err)
	}
	if probes != 1 {
		t.Fatalf("buffer probes = %d, want 1; server output: %s", probes, output.String())
	}
	for _, file := range []string{"value.go", "uses.go"} {
		data, err := os.ReadFile(filepath.Join(fixture.root, file))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) == originals[file] {
			t.Fatalf("batch did not publish final signature edit in %s", file)
		}
	}
}

func registerSignatureBufferProbe(t *testing.T, registry *operation.Registry, root string, originals map[string]string, identities map[string]os.FileInfo, probes *int) {
	t.Helper()
	probe := operation.Def[struct{}, operation.FileEditRes]{
		Key: "probe_signature_buffer", MCPName: "semantic_probe_signature_buffer", Batchable: true, BufferedBatch: true,
		Parse: func(map[string]any) (struct{}, error) { return struct{}{}, nil },
		Handlers: map[backend.LanguageID]func(context.Context, operation.CallContext, struct{}) (operation.FileEditRes, error){
			backend.LanguageGo: func(ctx context.Context, _ operation.CallContext, _ struct{}) (operation.FileEditRes, error) {
				return observeSignatureBuffer(ctx, root, originals, identities, probes)
			},
		},
		Format: func(operation.FileEditRes) (string, error) { return "observed", nil },
	}
	if err := operation.Register(registry, probe); err != nil {
		t.Fatal(err)
	}
}

func observeSignatureBuffer(ctx context.Context, root string, originals map[string]string, identities map[string]os.FileInfo, probes *int) (operation.FileEditRes, error) {
	if err := assertDiskPreimage(root, originals, identities); err != nil {
		return operation.FileEditRes{}, fmt.Errorf("signature batch published before flush: %w", err)
	}
	visible, err := sourcebuf.ReadFile(ctx, filepath.Join(root, "uses.go"))
	if err != nil {
		return operation.FileEditRes{}, err
	}
	if !strings.Contains(string(visible), "value(7)") {
		return operation.FileEditRes{}, fmt.Errorf("signature batch overlay lacks updated call: %s", visible)
	}
	*probes++
	return operation.FileEditRes{}, nil
}
