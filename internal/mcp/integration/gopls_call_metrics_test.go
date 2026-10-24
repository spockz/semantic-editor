// Package mcp_test checks gopls timings through MCP and retains per-call benchmark attribution.

package mcp_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/operation"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/telemetry"
)

func assertUnpublishedGoSource(t *testing.T, path string, original []byte) {
	t.Helper()
	disk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(disk, original) {
		t.Fatalf("staged bytes published before commit: %s", disk)
	}
}

func assertDeferredGoNotifications(t *testing.T, client *warmMCPCostClient, tool string) {
	t.Helper()
	metrics := latestGoplsCallMetrics(t, client, tool)
	for phase, measurement := range metrics.Phases {
		if phase == "go.session.use" || phase == "go.session.sync_documents" || strings.HasPrefix(phase, "gopls.notify.") || strings.HasPrefix(phase, "gopls.request.") {
			if measurement.Count != 0 {
				t.Errorf("buffered AST tool %s ran %s %d times", tool, phase, measurement.Count)
			}
		}
	}
}

func assertCleanGoNotificationVerification(t *testing.T, client *warmMCPCostClient) {
	t.Helper()
	raw := client.toolResult(t, "semantic_verify", map[string]any{"check_only": true})
	var envelope struct {
		Report struct {
			Status    string `json:"status"`
			Remaining *int   `json:"remaining_count"`
		} `json:"diagnostic_report"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Report.Status != "available" || envelope.Report.Remaining == nil || *envelope.Report.Remaining != 0 {
		t.Fatalf("verification did not report clean diagnostics: %s", raw)
	}
	metrics := latestGoplsCallMetrics(t, client, "semantic_verify")
	if got := metrics.Phases["gopls.request.initialize"].Count; got != 0 {
		t.Fatalf("metadata reload restarted gopls %d times", got)
	}
}

func checkGoNotificationPublication(t *testing.T, fixture managedMCPFixture, client *warmMCPCostClient, mode string) {
	t.Helper()
	client.toolResult(t, "refactor_rename", map[string]any{"file": "value.go", "symbol": "Value", "to": "Renamed"})
	value, err := os.ReadFile(filepath.Join(fixture.root, "value.go"))
	if err != nil {
		t.Fatal(err)
	}
	want := "return 2"
	if mode == "abort" {
		want = "return 1"
	}
	if !strings.Contains(string(value), want) || !strings.Contains(string(value), "func Renamed()") {
		t.Fatalf("published source = %s", value)
	}
	read, err := os.ReadFile(filepath.Join(fixture.root, "reader.go"))
	if err != nil || !strings.Contains(string(read), "return Renamed()") {
		t.Fatalf("reader source = %s, err = %v", read, err)
	}
	generated := filepath.Join(fixture.root, "generated.go")
	if mode == "abort" {
		if _, statErr := os.Stat(generated); !os.IsNotExist(statErr) {
			t.Fatalf("aborted scaffold: %v", statErr)
		}
	} else {
		client.toolResult(t, "refactor_rename", map[string]any{"file": "generated.go", "symbol": "Generated", "to": "Published"})
		contents, readErr := os.ReadFile(generated)
		if readErr != nil || !strings.Contains(string(contents), "func Published()") || !strings.Contains(string(contents), "return Renamed()") {
			t.Fatalf("generated source = %s, err = %v", contents, readErr)
		}
	}
	if writeErr := pipeline.WriteAtomic(filepath.Join(fixture.root, "reload.go"), []byte("package sample\n\nimport \"strconv\"\n\nfunc Reload() string { return strconv.Itoa(Renamed()) }\n")); writeErr != nil {
		t.Fatal(writeErr)
	}
	assertCleanGoNotificationVerification(t, client)
	assertSingleSessionStart(t, fixture.counter)
}

func publishGoNotificationChanges(t *testing.T, client *warmMCPCostClient, mode string) string {
	t.Helper()
	edits := []map[string]any{
		{"tool": "semantic_replace_body", "params": map[string]any{"file": "value.go", "symbol": "Value", "body": "return 2"}},
		{"tool": "semantic_scaffold_file", "params": map[string]any{"file": "generated.go", "package": "sample"}},
		{"tool": "semantic_insert_function", "params": map[string]any{"file": "generated.go", "source": "func Generated() int { return Value() }"}},
	}
	if mode == "batch" {
		client.toolResult(t, "semantic_batch", map[string]any{"edits": edits})
		return "semantic_batch"
	}
	client.toolResult(t, "semantic_stage", map[string]any{})
	for _, edit := range edits {
		tool := edit["tool"].(string)
		client.toolResult(t, tool, edit["params"].(map[string]any))
		assertDeferredGoNotifications(t, client, tool)
	}
	publication := "semantic_commit"
	if mode == "abort" {
		publication = "semantic_stage_abort"
	}
	client.toolResult(t, publication, map[string]any{})
	return publication
}

func latestGoplsCallMetrics(tb testing.TB, client *warmMCPCostClient, tool string) telemetry.Snapshot {
	tb.Helper()
	raw := client.toolResult(tb, "semantic_metrics", map[string]any{"limit": 1})
	var envelope struct {
		Result struct {
			Records []struct {
				Tool    string             `json:"tool"`
				Metrics telemetry.Snapshot `json:"metrics"`
			} `json:"records"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		tb.Fatal(err)
	}
	if len(envelope.Result.Records) != 1 || envelope.Result.Records[0].Tool != tool {
		tb.Fatalf("latest metrics do not describe %s: %s", tool, raw)
	}
	return envelope.Result.Records[0].Metrics
}

func TestMCPStagedRenameRepairsLazyAndDroppedViews(t *testing.T) {
	fixture := newWarmMCPCostFixture(t, 1)
	path := filepath.Join(fixture.root, "value.go")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	client := newWarmMCPCostClient(t, fixture.root)
	client.call(t, "initialize", map[string]any{})
	client.toolResult(t, "semantic_verify", map[string]any{"check_only": true})
	client.toolResult(t, "semantic_stage", map[string]any{})
	client.toolResult(t, "semantic_insert_function", map[string]any{"file": "value.go", "source": "func Added() int { return Value() + 3 }"})
	assertDeferredGoNotifications(t, client, "semantic_insert_function")
	assertUnpublishedGoSource(t, path, original)
	renamed := client.toolResult(t, "refactor_rename", map[string]any{"file": "value.go", "symbol": "Added", "to": "Ready"})
	metrics := latestGoplsCallMetrics(t, client, "refactor_rename")
	if metrics.Phases["gopls.request.textDocument/prepareRename"].Count != 1 || metrics.Phases["gopls.notify.textDocument/didChange"].Count == 0 {
		t.Fatalf("staged rename did not synchronize/query its working declaration: %+v", metrics)
	}
	assertUnpublishedGoSource(t, path, original)
	var envelope struct {
		Receipt struct {
			StepID string `json:"step_id"`
		} `json:"stage_receipt"`
	}
	if decodeErr := json.Unmarshal(renamed, &envelope); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if envelope.Receipt.StepID == "" {
		t.Fatalf("rename omitted staged step: %s", renamed)
	}
	client.toolResult(t, "semantic_stage_drop", map[string]any{"step_id": envelope.Receipt.StepID})
	assertDeferredGoNotifications(t, client, "semantic_stage_drop")
	client.toolResult(t, "refactor_rename", map[string]any{"file": "value.go", "symbol": "Added", "to": "Final"})
	assertUnpublishedGoSource(t, path, original)
	client.toolResult(t, "semantic_commit", map[string]any{})
	published, readErr := os.ReadFile(path)
	if readErr != nil || !strings.Contains(string(published), "func Final()") || strings.Contains(string(published), "func Ready()") || strings.Contains(string(published), "func Added()") {
		t.Fatalf("published replay=%s err=%v", published, readErr)
	}
	assertCleanGoNotificationVerification(t, client)
	assertSingleSessionStart(t, fixture.counter)
}

func TestMCPStageAbortRestoresDiskDocumentsWithoutRestart(t *testing.T) {
	fixture := newWarmMCPCostFixture(t, 1)
	value := filepath.Join(fixture.root, "value.go")
	original, err := os.ReadFile(value)
	if err != nil {
		t.Fatal(err)
	}
	client := newWarmMCPCostClient(t, fixture.root)
	client.call(t, "initialize", map[string]any{})
	client.toolResult(t, "semantic_verify", map[string]any{"check_only": true})
	client.toolResult(t, "semantic_stage", map[string]any{})
	client.toolResult(t, "semantic_scaffold_file", map[string]any{"file": "staged.go", "package": "sample"})
	client.toolResult(t, "semantic_insert_function", map[string]any{"file": "staged.go", "source": "func Staged() int { return Value() }"})
	client.toolResult(t, "refactor_rename", map[string]any{"file": "value.go", "symbol": "Value", "to": "Renamed"})
	client.toolResult(t, "semantic_stage_abort", map[string]any{})
	metrics := latestGoplsCallMetrics(t, client, "semantic_stage_abort")
	if got := metrics.Phases["gopls.notify.textDocument/didClose"].Count; got == 0 {
		t.Fatalf("abort did not close staged-only Go document: %+v", metrics)
	}
	if got := metrics.Phases["gopls.request.initialize"].Count; got != 0 {
		t.Fatalf("abort restarted gopls: %+v", metrics)
	}
	assertUnpublishedGoSource(t, value, original)
	if _, statErr := os.Stat(filepath.Join(fixture.root, "staged.go")); !os.IsNotExist(statErr) {
		t.Fatalf("abort retained staged file: %v", statErr)
	}
	assertCleanGoNotificationVerification(t, client)
	assertSingleSessionStart(t, fixture.counter)
}

func TestMCPStageAbortSynchronizesConcurrentDiskChange(t *testing.T) {
	fixture := newWarmMCPCostFixture(t, 1)
	value := filepath.Join(fixture.root, "value.go")
	client := newWarmMCPCostClient(t, fixture.root)
	client.call(t, "initialize", map[string]any{})
	client.toolResult(t, "semantic_verify", map[string]any{"check_only": true})
	client.toolResult(t, "semantic_stage", map[string]any{})
	client.toolResult(t, "semantic_replace_body", map[string]any{"file": "value.go", "symbol": "Value", "body": "return 2"})
	if err := pipeline.WriteAtomic(value, []byte("package sample\n\nfunc Value() int { return 3 }\n")); err != nil {
		t.Fatal(err)
	}
	client.toolResult(t, "semantic_stage_abort", map[string]any{})
	assertCleanGoNotificationVerification(t, client)
	contents, err := os.ReadFile(value)
	if err != nil || !strings.Contains(string(contents), "return 3") {
		t.Fatalf("abort did not retain concurrent disk source=%q err=%v", contents, err)
	}
	assertSingleSessionStart(t, fixture.counter)
}

func TestMCPASTOnlyBatchSynchronizesAfterFlush(t *testing.T) {
	fixture := newWarmMCPCostFixture(t, 1)
	client := newWarmMCPCostClient(t, fixture.root)
	client.call(t, "initialize", map[string]any{})
	client.toolResult(t, "semantic_verify", map[string]any{"check_only": true})
	edits := make([]map[string]any, 0, 3)
	for _, body := range []string{"return 2", "return 3", "return 4"} {
		edits = append(edits, map[string]any{"tool": "semantic_replace_body", "params": map[string]any{"file": "value.go", "symbol": "Value", "body": body}})
	}
	client.toolResult(t, "semantic_batch", map[string]any{"edits": edits})
	metrics := latestGoplsCallMetrics(t, client, "semantic_batch")
	for phase, want := range map[string]int{"gopls.notify.textDocument/didChange": 1, "gopls.notify.textDocument/didSave": 0, "gopls.notify.textDocument/didOpen": 0} {
		if got := metrics.Phases[phase].Count; got != want {
			t.Errorf("%s=%d want %d", phase, got, want)
		}
	}
	contents, err := os.ReadFile(filepath.Join(fixture.root, "value.go"))
	if err != nil || !strings.Contains(string(contents), "return 4") {
		t.Fatalf("published body=%s err=%v", contents, err)
	}
	assertCleanGoNotificationVerification(t, client)
	assertSingleSessionStart(t, fixture.counter)
}

func TestMCPEncapsulateFieldBatchReusesGoplsSession(t *testing.T) {
	fixture := newWarmMCPCostFixture(t, 1)
	configSource := "package sample\n\ntype config struct { value int }\n\nfunc (c *config) read() int { return c.value }\n"
	if err := pipeline.WriteAtomic(filepath.Join(fixture.root, "config.go"), []byte(configSource)); err != nil {
		t.Fatal(err)
	}
	client := newWarmMCPCostClient(t, fixture.root)
	client.call(t, "initialize", map[string]any{})
	client.toolResult(t, "semantic_verify", map[string]any{"check_only": true})
	edits := []map[string]any{
		{"tool": "refactor_encapsulate_field", "params": map[string]any{"file": "config.go", "symbol": "config.value", "getter": "readValue", "setter": "setValue"}},
		{"tool": "refactor_rename", "params": map[string]any{"file": "config.go", "symbol": "(*config).readValue", "to": "readStoredValue"}},
		{"tool": "semantic_replace_body", "params": map[string]any{"file": "config.go", "symbol": "(*config).setValue", "body": "receiver.value = value + 1"}},
	}
	client.toolResult(t, "semantic_batch", map[string]any{"edits": edits})
	metrics := latestGoplsCallMetrics(t, client, "semantic_batch")
	if got := metrics.Phases["gopls.request.textDocument/rename"].Count; got == 0 {
		t.Fatalf("encapsulate-field batch did not rename its generated accessor through gopls: %+v", metrics)
	}
	assertSingleSessionStart(t, fixture.counter)
	contents, err := os.ReadFile(filepath.Join(fixture.root, "config.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"c.readStoredValue()", "func (receiver *config) setValue", "receiver.value = value + 1"} {
		if !strings.Contains(string(contents), fragment) {
			t.Errorf("published encapsulation batch source lacks %q: %s", fragment, contents)
		}
	}
}

func TestMCPEncapsulateFieldReceiptBindsLiveAccessors(t *testing.T) {
	fixture := newWarmMCPCostFixture(t, 1)
	accessorsPath := filepath.Join(fixture.root, "accessors.go")
	if err := pipeline.WriteAtomic(filepath.Join(fixture.root, "config.go"), []byte("package sample\n\ntype config struct { value int }\n\nfunc (c *config) read() int { return c.value }\n")); err != nil {
		t.Fatal(err)
	}
	if err := pipeline.WriteAtomic(accessorsPath, []byte("package sample\n")); err != nil {
		t.Fatal(err)
	}
	client := newWarmMCPCostClient(t, fixture.root)
	client.call(t, "initialize", map[string]any{})
	client.toolResult(t, "semantic_verify", map[string]any{"check_only": true})
	raw := client.toolResult(t, "refactor_encapsulate_field", map[string]any{
		"file": "config.go", "destination_file": "accessors.go", "symbol": "config.value", "getter": "readValue", "setter": "setValue",
	})
	var response struct {
		Result struct {
			Receipt operation.EditReceipt `json:"edit_receipt"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatal(err)
	}
	if response.Result.Receipt.Status != operation.EditPersistedWithDiagnostics && response.Result.Receipt.Status != operation.EditPersisted {
		t.Fatalf("encapsulation receipt status = %q: %s", response.Result.Receipt.Status, raw)
	}
	if len(response.Result.Receipt.Continuations) != 2 {
		t.Fatalf("encapsulation continuations = %+v", response.Result.Receipt.Continuations)
	}
	accessors, err := os.ReadFile(accessorsPath)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(accessors)
	wantRevision := hex.EncodeToString(digest[:])
	for index, wantSymbol := range []string{"(*config).readValue", "(*config).setValue"} {
		continuation := response.Result.Receipt.Continuations[index]
		if continuation.File != "accessors.go" || continuation.Symbol != wantSymbol || continuation.ExpectedRevision == nil || *continuation.ExpectedRevision != wantRevision {
			t.Fatalf("continuation %d = %+v, want accessors.go %s at %s", index, continuation, wantSymbol, wantRevision)
		}
	}
	getter, setter := response.Result.Receipt.Continuations[0], response.Result.Receipt.Continuations[1]
	client.toolResult(t, "refactor_rename", map[string]any{"file": getter.File, "symbol": getter.Symbol, "to": "readStoredValue"})
	client.toolResult(t, "semantic_replace_body", map[string]any{"file": setter.File, "symbol": setter.Symbol, "body": "receiver.value = value + 1"})
	contents, err := os.ReadFile(accessorsPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"func (receiver *config) readStoredValue()", "func (receiver *config) setValue", "receiver.value = value + 1"} {
		if !bytes.Contains(contents, []byte(fragment)) {
			t.Errorf("follow-up using receipt target lacks %q: %s", fragment, contents)
		}
	}
	assertSingleSessionStart(t, fixture.counter)
}

func TestMCPEncapsulateFieldStageKeepsWorkingEditsUnpublished(t *testing.T) {
	fixture := newWarmMCPCostFixture(t, 1)
	path := filepath.Join(fixture.root, "config.go")
	original := []byte("package sample\n\ntype config struct { value int }\n\nfunc (c *config) read() int { return c.value }\n")
	if err := pipeline.WriteAtomic(path, original); err != nil {
		t.Fatal(err)
	}
	client := newWarmMCPCostClient(t, fixture.root)
	client.call(t, "initialize", map[string]any{})
	client.toolResult(t, "semantic_verify", map[string]any{"check_only": true})
	client.toolResult(t, "semantic_stage", map[string]any{})

	encapsulated := client.toolResult(t, "refactor_encapsulate_field", map[string]any{
		"file": "config.go", "symbol": "config.value", "getter": "readValue", "setter": "setValue",
	})
	if !bytes.Contains(encapsulated, []byte(`"status":"applied_working"`)) || !bytes.Contains(encapsulated, []byte(`"operation":"refactor_encapsulate_field"`)) {
		t.Fatalf("encapsulation did not report its working receipt: %s", encapsulated)
	}
	assertUnpublishedGoSource(t, path, original)

	client.toolResult(t, "refactor_rename", map[string]any{"file": "config.go", "symbol": "(*config).readValue", "to": "readStoredValue"})
	metrics := latestGoplsCallMetrics(t, client, "refactor_rename")
	if metrics.Phases["gopls.request.textDocument/rename"].Count == 0 || metrics.Phases["gopls.notify.textDocument/didChange"].Count == 0 {
		t.Fatalf("staged rename did not synchronize the working accessor: %+v", metrics)
	}
	client.toolResult(t, "semantic_replace_body", map[string]any{"file": "config.go", "symbol": "(*config).setValue", "body": "receiver.value = value + 1"})
	assertUnpublishedGoSource(t, path, original)
	assertSingleSessionStart(t, fixture.counter)

	client.toolResult(t, "semantic_commit", map[string]any{})
	published, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"c.readStoredValue()", "func (receiver *config) setValue", "receiver.value = value + 1"} {
		if !bytes.Contains(published, []byte(fragment)) {
			t.Errorf("published encapsulation stage lacks %q: %s", fragment, published)
		}
	}
}

func TestMCPGoPublicationNotificationPolicy(t *testing.T) {
	for _, mode := range []string{"batch", "stage", "abort"} {
		t.Run(mode, func(t *testing.T) {
			fixture := newWarmMCPCostFixture(t, 1)
			if err := pipeline.WriteAtomic(filepath.Join(fixture.root, "reader.go"), []byte("package sample\n\nfunc Read() int { return Value() }\n")); err != nil {
				t.Fatal(err)
			}
			client := newWarmMCPCostClient(t, fixture.root)
			client.call(t, "initialize", map[string]any{})
			client.toolResult(t, "semantic_verify", map[string]any{"check_only": true})
			publication := publishGoNotificationChanges(t, client, mode)
			metrics := latestGoplsCallMetrics(t, client, publication)
			if mode != "abort" {
				if metrics.Phases["gopls.notify.textDocument/didChange"].Count == 0 {
					t.Fatal("publication did not refresh gopls content after buffer flush")
				}
				if got := metrics.Phases["gopls.notify.textDocument/didSave"].Count; got != 0 {
					t.Errorf("atomic publication sent %d save notifications", got)
				}
			}
			checkGoNotificationPublication(t, fixture, client, mode)
		})
	}
}

func TestGoplsCallMetricsThroughMCP(t *testing.T) {
	fixture := newWarmMCPCostFixture(t, 1)
	client := newWarmMCPCostClient(t, fixture.root)
	client.call(t, "initialize", map[string]any{})
	client.toolResult(t, "refactor_rename", map[string]any{"file": "value.go", "symbol": "Value", "to": "Renamed"})
	metrics := latestGoplsCallMetrics(t, client, "refactor_rename")
	for phase, count := range map[string]int{
		"gopls.request.initialize":                 1,
		"gopls.request.textDocument/prepareRename": 1,
		"gopls.request.textDocument/rename":        1,
		"gopls.request.textDocument/diagnostic":    2,
		"verification.before_diagnostics":          1,
		"verification.after_diagnostics":           1,
	} {
		if got := metrics.Phases[phase].Count; got != count {
			t.Errorf("%s count = %d, want %d; metrics: %+v", phase, got, count, metrics)
		}
	}
	if metrics.Phases["gopls.notify.textDocument/didChange"].Count == 0 {
		t.Errorf("rename did not record source synchronization: %+v", metrics)
	}
	content, err := os.ReadFile(filepath.Join(fixture.root, "value.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "package sample\n\nfunc Renamed() int { return 1 }\n" {
		t.Fatalf("rename source = %q", content)
	}
}

func BenchmarkMCPGoplsCallAttribution(b *testing.B) {
	cases := []struct {
		name string
		kind string
	}{
		{name: "dirty_import_root", kind: "dirty_import_root"},
		{name: "dirty_import_root_unsafe", kind: "dirty_import_root_unsafe"},
		{name: "dirty_import_root_errors", kind: "dirty_import_root_errors"},
		{name: "dirty_import_leaf", kind: "dirty_import_leaf"},
		{name: "exported_api_rename", kind: "exported_api_rename"},
		{name: "body_only_edit", kind: "body_only_edit"},
		{name: "repository_telemetry_import", kind: "repository_telemetry_import"},
	}
	for _, scenario := range cases {
		b.Run(scenario.name, func(b *testing.B) { benchmarkGoplsCostScenario(b, scenario.kind) })
	}
}

func TestMCPImportCleanupRefreshesOnlyAffectedPackages(t *testing.T) {
	fixture := newWarmMCPCostFixture(t, 20)
	client := newWarmMCPCostClient(t, fixture.root)
	client.call(t, "initialize", map[string]any{})
	args := map[string]any{"file": "consumer19/consumer.go"}
	client.toolResult(t, "semantic_organize_imports", args)
	client.toolResult(t, "semantic_organize_imports", args)
	reused := latestGoplsCallMetrics(t, client, "semantic_organize_imports")
	if got := reused.Phases["gopls.request.textDocument/diagnostic"].Count; got != 0 {
		t.Fatalf("unchanged automatic capture pulled %d files, want 0", got)
	}
	path := filepath.Join(fixture.root, "consumer19", "consumer.go")
	original, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	dirty := bytes.Replace(original, []byte("import sample "), []byte("import \"strings\"\n\nimport sample "), 1)
	if bytes.Equal(dirty, original) {
		t.Fatal("fixture has no consumer import to extend")
	}
	if writeErr := pipeline.WriteAtomic(path, dirty); writeErr != nil {
		t.Fatal(writeErr)
	}
	raw := client.toolResult(t, "semantic_organize_imports", args)
	metrics := latestGoplsCallMetrics(t, client, "semantic_organize_imports")
	if got := metrics.Phases["gopls.request.textDocument/diagnostic"].Count; got != 2 {
		t.Errorf("import cleanup pulled %d files, want one affected leaf before and after", got)
	}
	if got := metrics.Phases["gopls.request.initialize"].Count; got != 0 {
		t.Errorf("import cleanup restarted gopls: initialize=%d", got)
	}
	var envelope struct {
		Report struct {
			Status    string `json:"status"`
			Remaining *int   `json:"remaining_count"`
			Resolved  *int   `json:"resolved_count"`
		} `json:"diagnostic_report"`
	}
	if decodeErr := json.Unmarshal(raw, &envelope); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if envelope.Report.Status != "available" || envelope.Report.Remaining == nil || *envelope.Report.Remaining != 0 || envelope.Report.Resolved == nil || *envelope.Report.Resolved != 1 {
		t.Fatalf("cleanup diagnostic report did not resolve the unused import: %s", raw)
	}
	published, finalReadErr := os.ReadFile(path)
	if finalReadErr != nil || !bytes.Equal(published, original) {
		t.Fatalf("cleanup source=%s err=%v, want original source", published, finalReadErr)
	}
	client.toolResult(t, "semantic_verify", map[string]any{"check_only": true})
	explicit := latestGoplsCallMetrics(t, client, "semantic_verify")
	if got := explicit.Phases["gopls.request.textDocument/diagnostic"].Count; got != 20 {
		t.Errorf("explicit verification pulled %d files, want all 20", got)
	}
	assertSingleSessionStart(t, fixture.counter)
}

type goplsCostTarget struct {
	fixture managedMCPFixture
	target  string
	tool    string
}

type goplsCostBenchmark struct {
	target     goplsCostTarget
	original   []byte
	client     *warmMCPCostClient
	baseStarts map[string]int
	baseline   []string
}

func benchmarkGoplsCostScenario(b *testing.B, kind string) {
	b.Helper()
	benchmark := setupGoplsCostBenchmark(b, kind)
	observed := newWarmObservations()
	b.ResetTimer()
	for index := range b.N {
		b.StopTimer()
		sample := index + 1
		prepareGoplsCostScenario(b, benchmark.target.fixture.root, kind, benchmark.target.target, benchmark.original, sample)
		b.StartTimer()
		raw := observed.call(b, benchmark.client, "operation", benchmark.target.tool, goplsCostArguments(kind, benchmark.target.target, sample))
		b.StopTimer()
		reportGoplsCostSample(b, observed, benchmark.client, benchmark.target.tool, kind, sample)
		if kind == "repository_telemetry_import" {
			assertGoplsCostReportCaptured(b, raw)
			assertGoplsCostVerificationBaseline(b, benchmark.client, benchmark.baseline)
		} else {
			warmDiagnosticReport(b, raw)
			if isGoplsCostImportScenario(kind) {
				assertGoplsCostImportResolved(b, raw)
			}
		}
		validateGoplsCostScenario(b, benchmark.target.fixture.root, kind, benchmark.target.target, benchmark.original, sample)
		assertGoplsCostContinuity(b, benchmark.target.fixture, benchmark.baseStarts)
		b.StartTimer()
	}
	b.StopTimer()
	if kind != "repository_telemetry_import" {
		warmVerify(b, benchmark.client)
	}
	assertGoplsCostContinuity(b, benchmark.target.fixture, benchmark.baseStarts)
	observed.report(b, map[string]time.Duration{"operation": time.Second}, managedSessionStartCount(benchmark.baseStarts))
}

func setupGoplsCostBenchmark(b *testing.B, kind string) goplsCostBenchmark {
	target := newGoplsCostTarget(b, kind)
	original, err := os.ReadFile(filepath.Join(target.fixture.root, target.target))
	if err != nil {
		b.Fatal(err)
	}
	client := newWarmMCPCostClient(b, target.fixture.root)
	client.call(b, "initialize", map[string]any{})
	var baseline []string
	if kind == "repository_telemetry_import" {
		baseline = goplsCostVerificationFingerprints(b, client)
		b.Logf("GOPLS_COST_DIAGNOSTIC_BASELINE count=%d", len(baseline))
	} else {
		warmVerify(b, client)
	}
	baseStarts := fileEditSessionStarts(b, target.fixture)
	if len(baseStarts) == 0 || fileEditSessionRestarted(baseStarts) {
		b.Fatalf("setup did not retain one gopls process per module: %v", baseStarts)
	}
	b.Logf("GOPLS_COST_SESSIONS retained_modules=%d", len(baseStarts))
	prepareGoplsCostScenario(b, target.fixture.root, kind, target.target, original, 0)
	calibration := client.toolResult(b, target.tool, goplsCostArguments(kind, target.target, 0))
	if kind == "repository_telemetry_import" {
		assertGoplsCostReportCaptured(b, calibration)
	} else {
		warmDiagnosticReport(b, calibration)
	}
	validateGoplsCostScenario(b, target.fixture.root, kind, target.target, original, 0)
	if kind != "repository_telemetry_import" && isGoplsCostImportScenario(kind) {
		assertGoplsCostImportResolved(b, calibration)
	}
	if kind == "repository_telemetry_import" {
		assertGoplsCostVerificationBaseline(b, client, baseline)
	}
	return goplsCostBenchmark{target: target, original: original, client: client, baseStarts: baseStarts, baseline: baseline}
}

func newGoplsCostTarget(b *testing.B, kind string) goplsCostTarget {
	result := goplsCostTarget{target: "value.go", tool: "semantic_organize_imports"}
	if kind == "repository_telemetry_import" {
		corpus := newRepositoryCostCorpus(b)
		result.fixture = newManagedMCPFixture(b)
		result.fixture.root = corpus.clone(b)
		result.target = filepath.Join("internal", "telemetry", "cost_mcp_diagnostics.go")
		if err := pipeline.WriteAtomic(filepath.Join(result.fixture.root, result.target), []byte("package telemetry\n\nfunc CostMCPDiagnosticsProbe() int { return 1 }\n")); err != nil {
			b.Fatal(err)
		}
		coldHotWrite(b, result.fixture.root, ".semedit.yaml", "version: 1\nverify:\n  check:\n    - id: golangci-lint\n      languages: [go]\n      disabled: true\n")
		b.Logf("GOPLS_COST_CORPUS revision=%s target=%s", corpus.revision, result.target)
	} else {
		result.fixture = newWarmMCPCostFixture(b, 20)
		switch kind {
		case "dirty_import_leaf":
			result.target = filepath.Join("consumer19", "consumer.go")
		case "exported_api_rename":
			result.tool = "refactor_rename"
		case "body_only_edit":
			result.tool = "semantic_replace_body"
		}
	}
	return result
}

func assertGoplsCostContinuity(b *testing.B, fixture managedMCPFixture, baseStarts map[string]int) {
	b.Helper()
	if change := fileEditSessionChange(baseStarts, fileEditSessionStarts(b, fixture)); change != "" {
		b.Fatalf("gopls session continuity changed: %s", change)
	}
}

func reportGoplsCostSample(b *testing.B, observed *warmObservations, client *warmMCPCostClient, tool, kind string, sample int) {
	b.Helper()
	metrics := latestGoplsCallMetrics(b, client, tool)
	before := metrics.Phases["verification.before_diagnostics"]
	after := metrics.Phases["verification.after_diagnostics"]
	var goplsCalls int
	for phase, measurement := range metrics.Phases {
		if strings.HasPrefix(phase, "gopls.request.") {
			goplsCalls += measurement.Count
		}
	}
	latencies := observed.components["operation"].latencies
	latency := latencies[len(latencies)-1]
	report := struct {
		Scenario                string             `json:"scenario"`
		Sample                  int                `json:"sample"`
		EndToEndMS              float64            `json:"end_to_end_ms"`
		VerificationBeforeCount int                `json:"verification_before_count"`
		VerificationBeforeMS    float64            `json:"verification_before_ms"`
		VerificationAfterCount  int                `json:"verification_after_count"`
		VerificationAfterMS     float64            `json:"verification_after_ms"`
		GoplsRequestCalls       int                `json:"gopls_request_calls"`
		Metrics                 telemetry.Snapshot `json:"metrics"`
	}{
		Scenario: kind, Sample: sample,
		EndToEndMS:              float64(latency) / float64(time.Millisecond),
		VerificationBeforeCount: before.Count, VerificationBeforeMS: float64(before.DurationMS),
		VerificationAfterCount: after.Count, VerificationAfterMS: float64(after.DurationMS),
		GoplsRequestCalls: goplsCalls, Metrics: metrics,
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("GOPLS_COST_SAMPLE %s", encoded)
}

func goplsCostVerificationFingerprints(tb testing.TB, client *warmMCPCostClient) []string {
	tb.Helper()
	raw := client.toolResult(tb, "semantic_verify", coldHotVerifyArgs())
	var verification struct {
		Result struct {
			Failure     string               `json:"failure"`
			Diagnostics []backend.Diagnostic `json:"diagnostics"`
			Coverage    []struct {
				Language   string `json:"language"`
				Status     string `json:"status"`
				ModuleRoot string `json:"module_root"`
			} `json:"coverage"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &verification); err != nil {
		tb.Fatalf("decode full repository verification: %v: %s", err, raw)
	}
	if verification.Result.Failure != "" {
		tb.Fatalf("full repository verification failed: %s", raw)
	}
	verifiedModules := make(map[string]bool)
	for _, coverage := range verification.Result.Coverage {
		if coverage.Language == "go" && coverage.Status == "verified" {
			verifiedModules[coverage.ModuleRoot] = true
		}
	}
	if len(verifiedModules) < 3 {
		tb.Fatalf("full repository verification covered %d verified Go modules, want at least 3: %s", len(verifiedModules), raw)
	}
	fingerprints := make([]string, 0, len(verification.Result.Diagnostics))
	for _, diagnostic := range verification.Result.Diagnostics {
		encoded, err := json.Marshal(diagnostic)
		if err != nil {
			tb.Fatalf("encode diagnostic identity: %v", err)
		}
		fingerprints = append(fingerprints, string(encoded))
	}
	slices.Sort(fingerprints)
	return fingerprints
}

func assertGoplsCostVerificationBaseline(tb testing.TB, client *warmMCPCostClient, baseline []string) {
	tb.Helper()
	after := goplsCostVerificationFingerprints(tb, client)
	if !slices.Equal(after, baseline) {
		tb.Fatalf("full-workspace diagnostic set changed from baseline: before=%d after=%d", len(baseline), len(after))
	}
}

func assertGoplsCostReportCaptured(tb testing.TB, raw []byte) {
	tb.Helper()
	var response struct {
		Report struct {
			Status string `json:"status"`
			ID     string `json:"report_id"`
		} `json:"diagnostic_report"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		tb.Fatalf("decode diagnostic report: %v: %s", err, raw)
	}
	if (response.Report.Status != "available" && response.Report.Status != "partial") || response.Report.ID == "" {
		tb.Fatalf("operation did not capture diagnostics: %s", raw)
	}
}

func goplsCostArguments(kind, target string, sample int) map[string]any {
	switch kind {
	case "exported_api_rename":
		from, to := "Value", "Renamed"
		if sample%2 != 0 {
			from, to = to, from
		}
		return map[string]any{"file": target, "symbol": from, "to": to}
	case "body_only_edit":
		body := "return 2"
		if sample%2 != 0 {
			body = "return 1"
		}
		return map[string]any{"file": target, "symbol": "Value", "body": body}
	default:
		return map[string]any{"file": target}
	}
}

func prepareGoplsCostScenario(tb testing.TB, root, kind, target string, original []byte, sample int) {
	tb.Helper()
	switch kind {
	case "dirty_import_root", "dirty_import_root_unsafe", "dirty_import_root_errors", "dirty_import_leaf", "repository_telemetry_import":
		packageLineEnd := bytes.IndexByte(original, '\n')
		if packageLineEnd < 0 {
			tb.Fatalf("source file %s has no package declaration line", target)
		}
		dirty := append([]byte{}, original[:packageLineEnd+1]...)
		dirty = append(dirty, []byte("\nimport \""+goplsCostUnusedImport(kind)+"\"\n")...)
		dirty = append(dirty, original[packageLineEnd+1:]...)
		if err := pipeline.WriteAtomic(filepath.Join(root, target), dirty); err != nil {
			tb.Fatal(err)
		}
	case "body_only_edit":
		if err := pipeline.WriteAtomic(filepath.Join(root, target), original); err != nil {
			tb.Fatal(err)
		}
		if kind == "body_only_edit" && sample%2 != 0 {
			updated := strings.Replace(string(original), "return 1", "return 2", 1)
			if err := pipeline.WriteAtomic(filepath.Join(root, target), []byte(updated)); err != nil {
				tb.Fatal(err)
			}
		}
	}
}

func validateGoplsCostScenario(tb testing.TB, root, kind, target string, original []byte, sample int) {
	tb.Helper()
	if isGoplsCostImportScenario(kind) {
		data, err := os.ReadFile(filepath.Join(root, target))
		if err != nil {
			tb.Fatal(err)
		}
		importPath := goplsCostUnusedImport(kind)
		if strings.Contains(string(data), `"`+importPath+`"`) {
			tb.Fatalf("organize imports retained unused %s import in %s", importPath, target)
		}
		if !bytes.Equal(data, original) {
			tb.Fatalf("organize imports changed source beyond removing unused %s import in %s: got %q, want %q", importPath, target, data, original)
		}
		if kind == "repository_telemetry_import" && string(data) != "package telemetry\n\nfunc CostMCPDiagnosticsProbe() int { return 1 }\n" {
			tb.Fatalf("repository import operation changed probe content: %q", data)
		}
		return
	}
	if kind == "exported_api_rename" {
		name := "Value"
		if sample%2 == 0 {
			name = "Renamed"
		}
		goplsCostSourceContains(tb, root, target, "func "+name+"(")
		goplsCostSourceContains(tb, root, filepath.Join("consumer1", "consumer.go"), "sample."+name+"()")
		return
	}
	value := 1
	if sample%2 == 0 {
		value = 2
	}
	goplsCostSourceContains(tb, root, target, fmt.Sprintf("return %d", value))
}

func isGoplsCostImportScenario(kind string) bool {
	switch kind {
	case "dirty_import_root", "dirty_import_root_unsafe", "dirty_import_root_errors", "dirty_import_leaf", "repository_telemetry_import":
		return true
	default:
		return false
	}
}

func goplsCostUnusedImport(kind string) string {
	switch kind {
	case "dirty_import_root_unsafe":
		return "unsafe"
	case "dirty_import_root_errors":
		return "errors"
	default:
		return "fmt"
	}
}

func assertGoplsCostImportResolved(tb testing.TB, raw []byte) {
	tb.Helper()
	var response struct {
		Result struct {
			DiagnosticDelta struct {
				Resolved []string `json:"resolved"`
			} `json:"diagnostic_delta"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		tb.Fatalf("decode import-resolution diagnostics: %v: %s", err, raw)
	}
	if len(response.Result.DiagnosticDelta.Resolved) == 0 {
		tb.Fatalf("unused import did not appear in resolved pre-edit diagnostics: %s", raw)
	}
}

func goplsCostSourceContains(tb testing.TB, root, file, fragment string) {
	tb.Helper()
	data, err := os.ReadFile(filepath.Join(root, file))
	if err != nil {
		tb.Fatal(err)
	}
	if !strings.Contains(string(data), fragment) {
		tb.Fatalf("%s does not contain %q", file, fragment)
	}
}
