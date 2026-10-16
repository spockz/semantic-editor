// Package telemetry verifies request timing aggregation independently of MCP transport.
package telemetry

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"runtime/trace"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spockz/semantic-editor/internal/atomicfile"
)

func TestMetricsRecordsScopedAndFallbackPhases(t *testing.T) {
	t.Parallel()

	metrics := NewMetrics()
	ctx := WithMetrics(context.Background(), metrics)
	metrics.Record(PhaseFormattingGofmt, 7*time.Millisecond)
	finish := Start(WithPhase(ctx, PhaseVerificationBefore), PhaseVerificationDiagnostics)
	time.Sleep(time.Millisecond)
	finish()

	snapshot := metrics.Snapshot(12 * time.Millisecond)
	if snapshot.SchemaVersion != 1 || snapshot.TotalMS != 12 {
		t.Fatalf("snapshot header = %#v, want schema 1 and total 12ms", snapshot)
	}
	if got := snapshot.Phases[string(PhaseFormattingGofmt)]; got.Count != 1 || got.DurationMS != 7 {
		t.Errorf("gofmt metric = %#v, want one 7ms record", got)
	}
	if got := snapshot.Phases[string(PhaseVerificationBefore)]; got.Count != 1 {
		t.Errorf("scoped metric = %#v, want one before-verification record", got)
	}
}

func TestStartUsesScopedPhaseWhenFallbackIsEmpty(t *testing.T) {
	t.Parallel()

	metrics := NewMetrics()
	ctx := WithPhase(WithMetrics(context.Background(), metrics), PhaseVerificationBefore)
	Start(ctx, "")()
	if got := metrics.Snapshot(0).Phases[string(PhaseVerificationBefore)]; got.Count != 1 {
		t.Fatalf("scoped metric = %#v, want one record", got)
	}
}

func TestStartPhaseIgnoresScopedPhaseAndAggregatesSubMillisecondDurations(t *testing.T) {
	t.Parallel()

	metrics := NewMetrics()
	ctx := WithPhase(WithMetrics(context.Background(), metrics), PhaseVerificationBefore)
	for range 5 {
		metrics.Record("gopls.request.textDocument/diagnostic", 250*time.Microsecond)
	}
	StartPhase(ctx, "gopls.notify.initialized")()
	snapshot := metrics.Snapshot(time.Second)
	if got := snapshot.Phases["gopls.request.textDocument/diagnostic"]; got.Count != 5 || got.DurationMS != 1 {
		t.Fatalf("sub-millisecond aggregate = %#v, want five calls totaling 1ms", got)
	}
	if got := snapshot.Phases["gopls.notify.initialized"]; got.Count != 1 {
		t.Fatalf("exact phase = %#v, want one exact record", got)
	}
	if _, ok := snapshot.Phases[string(PhaseVerificationBefore)]; ok {
		t.Fatal("exact phase was attributed to the scoped verification phase")
	}
}

func TestMetricsConcurrentRecord(t *testing.T) {
	t.Parallel()

	metrics := NewMetrics()
	const workers = 32
	const records = 100
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for range records {
				metrics.Record("gopls.request.concurrent", 10*time.Microsecond)
			}
		})
	}
	wg.Wait()
	got := metrics.Snapshot(0).Phases["gopls.request.concurrent"]
	if got.Count != workers*records || got.DurationMS != 32 {
		t.Fatalf("concurrent aggregate = %#v, want %d records totaling 32ms", got, workers*records)
	}
}

func TestStartEmitsResolvedTraceRegionsWithoutMetrics(t *testing.T) {
	if trace.IsEnabled() {
		t.Skip("runtime tracing is already enabled")
	}
	var traceData bytes.Buffer
	if err := trace.Start(&traceData); err != nil {
		t.Fatalf("start runtime trace: %v", err)
	}
	StartPhase(context.Background(), "go.session.snapshot")()
	Start(WithPhase(context.Background(), "request.dispatch"), "request.fallback")()
	trace.Stop()

	tracePath := filepath.Join(t.TempDir(), "regions.trace")
	if err := atomicfile.WriteAtomic(tracePath, traceData.Bytes()); err != nil {
		t.Fatalf("write runtime trace: %v", err)
	}
	output, err := exec.CommandContext(t.Context(), "go", "tool", "trace", "-d=parsed", tracePath).CombinedOutput()
	if err != nil {
		t.Fatalf("decode runtime trace: %v\n%s", err, output)
	}
	for _, name := range []string{"go.session.snapshot", "request.dispatch"} {
		if !strings.Contains(string(output), name) {
			t.Errorf("runtime trace does not contain region %q", name)
		}
	}
	if strings.Contains(string(output), "request.fallback") {
		t.Error("runtime trace used fallback despite an explicit scoped phase")
	}
}
