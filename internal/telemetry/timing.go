// Package telemetry carries request-scoped timing data without coupling semantic operations to MCP transport.
package telemetry

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"
)

// Phase identifies a portion of a semantic tool request whose latency is useful to compare.
type Phase string

const (
	// PhaseArgumentParsing measures decoding the tool arguments.
	PhaseArgumentParsing Phase = "request.argument_parsing"
	// PhaseDispatch measures operation dispatch and semantic mutation work outside instrumented substeps.
	PhaseDispatch Phase = "request.dispatch"
	// PhaseResponseFormatting measures construction of the human-readable tool response.
	PhaseResponseFormatting Phase = "request.response_formatting"
	// PhaseVerificationBefore measures diagnostics captured before an automatic mutation.
	PhaseVerificationBefore Phase = "verification.before_diagnostics"
	// PhaseVerificationAfter measures diagnostics captured after an automatic mutation.
	PhaseVerificationAfter Phase = "verification.after_diagnostics"
	// PhaseVerificationDiagnostics measures diagnostics not associated with a before/after edit pair.
	PhaseVerificationDiagnostics Phase = "verification.diagnostics"
	// PhaseFormattingGofmt measures gofmt invoked by semantic operations.
	PhaseFormattingGofmt Phase = "formatting.gofmt"
	// PhaseFormattingImports measures import organization and its formatting work.
	PhaseFormattingImports Phase = "formatting.organize_imports"
	// PhaseFormattingAST measures in-process Go source formatting after an AST transformation.
	PhaseFormattingAST Phase = "formatting.ast"
)

// PhaseMetric aggregates repeated occurrences of a timing phase in one tool request.
type PhaseMetric struct {
	Count      int   `json:"count"`
	DurationMS int64 `json:"duration_ms"`
}

// Snapshot is the stable, transport-safe representation of request timing data.
type Snapshot struct {
	SchemaVersion int                    `json:"schema_version"`
	TotalMS       int64                  `json:"total_ms"`
	Phases        map[string]PhaseMetric `json:"phases,omitempty"`
}

// Metrics accumulates phases for one request and is safe for operation implementations to share.
type Metrics struct {
	mu     sync.Mutex
	phases map[Phase]PhaseMetric
}

// NewMetrics creates an empty request recorder.
func NewMetrics() *Metrics {
	return &Metrics{phases: make(map[Phase]PhaseMetric)}
}

// Record adds one completed phase.
func (m *Metrics) Record(phase Phase, elapsed time.Duration) {
	if m == nil || phase == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	metric := m.phases[phase]
	metric.Count++
	metric.DurationMS += elapsed.Milliseconds()
	m.phases[phase] = metric
}

// Snapshot returns a copy suitable for including in an MCP structured response.
func (m *Metrics) Snapshot(total time.Duration) Snapshot {
	snapshot := Snapshot{SchemaVersion: 1, TotalMS: total.Milliseconds()}
	if m == nil {
		return snapshot
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.phases) == 0 {
		return snapshot
	}
	keys := slices.Sorted(maps.Keys(m.phases))
	snapshot.Phases = make(map[string]PhaseMetric, len(keys))
	for _, key := range keys {
		snapshot.Phases[string(key)] = m.phases[key]
	}
	return snapshot
}

type metricsContextKey struct{}
type phaseContextKey struct{}

// WithMetrics attaches a recorder so lower layers can contribute timing without depending on MCP.
func WithMetrics(ctx context.Context, metrics *Metrics) context.Context {
	return context.WithValue(ctx, metricsContextKey{}, metrics)
}

// WithPhase selects the name recorded by the next lower-layer timing call.
func WithPhase(ctx context.Context, phase Phase) context.Context {
	return context.WithValue(ctx, phaseContextKey{}, phase)
}

// Start records elapsed time under the scoped phase, or fallback when no scoped phase exists.
func Start(ctx context.Context, fallback Phase) func() {
	metrics, _ := ctx.Value(metricsContextKey{}).(*Metrics)
	phase := fallback
	if scoped, ok := ctx.Value(phaseContextKey{}).(Phase); ok && scoped != "" {
		phase = scoped
	}
	started := time.Now()
	return func() { metrics.Record(phase, time.Since(started)) }
}
