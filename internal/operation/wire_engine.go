// Package operation wires host-engine editing operations into the central registry.
package operation

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"semedit/internal/pipeline"
	"semedit/internal/telemetry"
)

var (
	placementEnum  = []string{"file_start", "file_end", "public_start", "public_end", "private_start", "private_end", "before_symbol", "after_symbol"}
	accessEnum     = []string{"infer", "public", "private", "protected", "package-private"}
	visibilityEnum = []string{"public", "private"}
	groupEnum      = []string{"append", "standalone"}
	caseEnum       = []string{"first", "last", "before_default", "before", "after"}
)

// FileOutcome is implemented by results that record a written file for batch post-processing.
type FileOutcome interface {
	WrittenFile() string
}

// AppendDiagnosticDelta appends a compiler diagnostic delta to a response base string.
// It mirrors the MCP server helper retired in the ingress migration step.
func AppendDiagnosticDelta(base string, delta pipeline.DiagnosticDelta) string {
	if len(delta.Introduced) == 0 && len(delta.Resolved) == 0 && len(delta.Suggestions) == 0 {
		return base
	}
	var sb strings.Builder
	sb.WriteString(base)
	if len(delta.Introduced) > 0 || len(delta.Resolved) > 0 {
		fmt.Fprintf(&sb, "\nDiagnostics delta (net %d):\n", delta.NetDelta)
		if len(delta.Resolved) > 0 {
			fmt.Fprintf(&sb, "Resolved:\n- %s\n", strings.Join(delta.Resolved, "\n- "))
		}
		if len(delta.Introduced) > 0 {
			fmt.Fprintf(&sb, "Introduced:\n- %s\n", strings.Join(delta.Introduced, "\n- "))
		}
	}
	if len(delta.Suggestions) > 0 {
		fmt.Fprintf(&sb, "\nActionable suggestions:\n- %s\n", strings.Join(delta.Suggestions, "\n- "))
	}
	return sb.String()
}

// resolveWorkPath mirrors the MCP server path resolution: relative paths join the workspace root.
func resolveWorkPath(workDir, file string) string {
	if file == "" {
		return workDir
	}
	if workDir != "" && !filepath.IsAbs(file) {
		return filepath.Join(workDir, file)
	}
	return file
}

// effectiveAutoOrganize applies the batch deferral flag over the parsed default.
func effectiveAutoOrganize(cc CallContext, parsed bool) bool {
	if cc.DeferImports {
		return false
	}
	return parsed
}

const automaticVerificationGuidance = " This standalone operation automatically verifies diagnostics and has no verification opt-out; do not call semantic_verify separately."

func surroundingDelta(ctx context.Context, workDir string, deferVerification bool) func() pipeline.DiagnosticDelta {
	if deferVerification {
		return func() pipeline.DiagnosticDelta { return pipeline.DiagnosticDelta{} }
	}
	before, _ := pipeline.CheckDiagnostics(telemetry.WithPhase(ctx, telemetry.PhaseVerificationBefore), workDir)
	return func() pipeline.DiagnosticDelta {
		after, _ := pipeline.CheckDiagnostics(telemetry.WithPhase(ctx, telemetry.PhaseVerificationAfter), workDir)
		return pipeline.ComputeDelta(before, after)
	}
}

// FileEditRes reports a single-file mutation for response rendering and batch post-processing.
type FileEditRes struct {
	File     string
	Display  string
	Symbol   string
	Diff     string
	Detail   string
	Delta    pipeline.DiagnosticDelta
	HasDelta bool
}

// WrittenFile returns the resolved mutated path for batch post-processing.
func (r FileEditRes) WrittenFile() string { return r.File }

func formatFileEdit(res FileEditRes) (string, error) {
	text := fmt.Sprintf(res.Detail, res.Display)
	if res.Diff != "" {
		text += "\n" + res.Diff
	}
	if res.HasDelta {
		text = AppendDiagnosticDelta(text, res.Delta)
	}
	return text, nil
}

// registerEngineOps adds the host-engine file and build operations to the registry.
func registerEngineOps(registry *Registry) error {
	if err := Register(registry, assertionModeDef()); err != nil {
		return err
	}
	if err := Register(registry, insertDeclarationDef()); err != nil {
		return err
	}
	if err := Register(registry, insertFunctionDef()); err != nil {
		return err
	}
	if err := Register(registry, insertTypeDef()); err != nil {
		return err
	}
	if err := Register(registry, insertDeclDef()); err != nil {
		return err
	}
	if err := Register(registry, organizeImportsDef()); err != nil {
		return err
	}
	if err := Register(registry, addBuildDependencyDef()); err != nil {
		return err
	}
	if err := Register(registry, replaceBodyDef()); err != nil {
		return err
	}
	if err := Register(registry, scaffoldFileDef()); err != nil {
		return err
	}
	return Register(registry, insertCaseDef())
}
