// Package telemetry verifies request timing aggregation independently of MCP transport.
package telemetry

import (
	"context"
	"testing"
	"time"
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
