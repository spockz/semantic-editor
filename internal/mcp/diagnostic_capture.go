// Package mcp captures completed diagnostic evidence for later retrieval.
// WHY: Paging diagnostic snapshots must reuse operation output instead of invoking verification again.
package mcp

import (
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/operation"
)

func moveFileResponseText(result operation.FileEditRes, reportValue any) string {
	parts := []string{"Successfully moved file declarations."}
	if detail := strings.TrimSpace(result.Detail); detail != "" && detail != parts[0] {
		parts = append(parts, detail)
	}
	if state := strings.TrimSpace(result.ApplicationState); state != "" {
		parts = append(parts, "Publication state: "+state+".")
	}
	if len(result.ChangedFiles) > 0 {
		visible := result.ChangedFiles
		if len(visible) > moveFileVisibleChangedPathLimit {
			visible = visible[:moveFileVisibleChangedPathLimit]
			parts = append(parts, fmt.Sprintf("Changed paths (%d): %s, plus %d more in structuredContent.result.changed_files.", len(result.ChangedFiles), strings.Join(visible, ", "), len(result.ChangedFiles)-len(visible)))
		} else {
			parts = append(parts, "Changed paths: "+strings.Join(visible, ", ")+".")
		}
	}
	if report, ok := reportValue.(map[string]any); ok {
		status, _ := report["status"].(string)
		switch status {
		case "available", "partial":
			parts = append(parts, fmt.Sprintf("Diagnostics captured (%s): active %s, introduced %s, resolved %s, suggestions %s. Page details with semantic_diagnostics using report_id %v and category active.", status, diagnosticCountText(report["remaining_count"]), diagnosticCountText(report["introduced_count"]), diagnosticCountText(report["resolved_count"]), diagnosticCountText(report["suggestion_count"]), report["report_id"]))
		case "deferred", diagnosticReportStatusUnavailable:
			reason, _ := report["reason"].(string)
			if reason != "" {
				parts = append(parts, "Diagnostics "+status+": "+reason)
			} else {
				parts = append(parts, "Diagnostics "+status+".")
			}
		}
	}
	return strings.Join(parts, " ")
}

const moveFileVisibleChangedPathLimit = 10

func moveFilePresentationFields(structured map[string]any) map[string]any {
	fields := maps.Clone(structured)
	delete(fields, diagnosticDeltaField)
	result, ok := structured["result"].(map[string]any)
	if !ok {
		return fields
	}
	result = maps.Clone(result)
	if changedFiles, ok := result["changed_files"].([]string); ok {
		result["changed_file_count"] = len(changedFiles)
		delete(result, "changed_files")
	} else if changedFiles, ok := result["changed_files"].([]any); ok {
		result["changed_file_count"] = len(changedFiles)
		delete(result, "changed_files")
	}
	for _, key := range []string{"edit_receipt", "receipt"} {
		receipt, ok := result[key].(map[string]any)
		if !ok {
			continue
		}
		receipt = maps.Clone(receipt)
		if changes, exists := receipt["changes"]; exists {
			receipt["change_count"] = sliceLength(changes)
			delete(receipt, "changes")
		}
		if readSet, exists := receipt["read_set"]; exists {
			receipt["read_set_count"] = sliceLength(readSet)
			delete(receipt, "read_set")
		}
		result[key] = receipt
	}
	fields["result"] = result
	return fields
}

func sliceLength(value any) int {
	switch items := value.(type) {
	case []any:
		return len(items)
	case []string:
		return len(items)
	case []operation.FileRevision:
		return len(items)
	default:
		return 0
	}
}

func diagnosticCountText(value any) string {
	if value == nil {
		return "not captured"
	}
	return fmt.Sprint(value)
}

func (s *Server) attachDiagnosticReport(fields map[string]any, root string, native any) {
	if report := s.diagnosticReportForResult(root, native); report != nil {
		fields["diagnostic_report"] = report
	}
}

func (s *Server) diagnosticReportForResult(root string, native any) map[string]any {
	if edit, ok := operation.DiagnosticFileEditResult(native); ok {
		return s.fileEditDiagnosticReport(root, edit)
	}
	switch result := native.(type) {
	case *backend.RenameResult:
		if result == nil {
			return nil
		}
		if result.ApplicationState == string(operation.EditAppliedWorking) {
			return unavailableDiagnosticReport("deferred", "Diagnostics are deferred until publication.")
		}
		return s.publishDiagnosticReport(messageDiagnosticReport(root, result.Active, result.Diagnostics.Introduced, result.Diagnostics.Resolved, result.Diagnostics.Suggestions))
	case operation.VerifyRes:
		report, ok := verificationDiagnosticReport(root, &result)
		if !ok {
			return unavailableDiagnosticReport(diagnosticReportStatusUnavailable, "Language-server diagnostic collection did not complete.")
		}
		return s.publishDiagnosticReport(report)
	case *BatchResponse:
		return s.batchDiagnosticReport(root, result)
	case operation.StageCommitResult:
		return s.stageDiagnosticReport(root, result)
	case operation.UndoRes:
		return s.undoDiagnosticReport(root, result)
	default:
		return nil
	}
}

func (s *Server) stageDiagnosticReport(root string, result operation.StageCommitResult) map[string]any {
	report, ok := verificationDiagnosticReport(root, result.Verification)
	if !ok {
		return unavailableDiagnosticReport(diagnosticReportStatusUnavailable, "Staged publication did not capture a completed diagnostic inventory.")
	}
	if result.DiagnosticDelta != nil {
		report.Introduced = diagnosticMessages(result.DiagnosticDelta.Introduced)
		report.Resolved = diagnosticMessages(result.DiagnosticDelta.Resolved)
	}
	return s.publishDiagnosticReport(report)
}

func (s *Server) diagnosticReportForError(operationErr error) map[string]any {
	if renameErr, ok := errors.AsType[*backend.RenameDiagnosticsError](operationErr); ok && renameErr.Result != nil {
		return s.diagnosticReportForResult(renameErr.Result.ProjectRoot, renameErr.Result)
	}
	if stageErr, ok := errors.AsType[*operation.StageCommitError](operationErr); ok && stageErr.Result.Verification != nil {
		root := stageErr.Result.Verification.CheckReport.Root
		if root == "" {
			root = stageErr.Result.Verification.NormalizeReport.Root
		}
		if root == "" {
			return unavailableDiagnosticReport(diagnosticReportStatusUnavailable, "The captured workspace root is unavailable.")
		}
		return s.stageDiagnosticReport(root, stageErr.Result)
	}
	return nil
}

func (s *Server) fileEditDiagnosticReport(root string, result operation.FileEditRes) map[string]any {
	if result.DiagnosticStatus == "deferred" || result.WorkingView {
		return unavailableDiagnosticReport("deferred", "Diagnostics are deferred until publication.")
	}
	if !result.HasDelta || (result.DiagnosticStatus != "changed" && result.DiagnosticStatus != "unchanged") {
		return unavailableDiagnosticReport(diagnosticReportStatusUnavailable, "No completed diagnostic inventory was captured.")
	}
	return s.publishDiagnosticReport(messageDiagnosticReport(root, result.Delta.After, result.Delta.Introduced, result.Delta.Resolved, result.Delta.Suggestions))
}

func (s *Server) batchDiagnosticReport(root string, result *BatchResponse) map[string]any {
	if result == nil {
		return nil
	}
	report, ok := verificationDiagnosticReport(root, result.Verification)
	if !ok {
		return unavailableDiagnosticReport(diagnosticReportStatusUnavailable, "No completed final batch diagnostic inventory was captured.")
	}
	if result.DiagnosticDelta != nil {
		report.Introduced = diagnosticMessages(result.DiagnosticDelta.Introduced)
		report.Resolved = diagnosticMessages(result.DiagnosticDelta.Resolved)
		report.Suggestions = diagnosticMessages(result.DiagnosticDelta.Suggestions)
	}
	return s.publishDiagnosticReport(report)
}

func (s *Server) undoDiagnosticReport(root string, result operation.UndoRes) map[string]any {
	if result.Result == nil {
		return nil
	}
	if result.Result.DiagnosticStatus != "available" {
		return unavailableDiagnosticReport(diagnosticReportStatusUnavailable, "Restoration did not capture a completed diagnostic inventory.")
	}
	report := messageDiagnosticReport(root, result.Result.Diagnostics, nil, nil, nil)
	if result.Result.Delta != nil {
		report.Introduced = diagnosticMessages(result.Result.Delta.Introduced)
		report.Resolved = diagnosticMessages(result.Result.Delta.Resolved)
		report.Suggestions = diagnosticMessages(result.Result.Delta.Suggestions)
	}
	return s.publishDiagnosticReport(report)
}

func diagnosticMessages(messages []string) []backend.Diagnostic {
	result := make([]backend.Diagnostic, len(messages))
	for i, message := range messages {
		result[i] = backend.Diagnostic{Message: message}
	}
	return result
}

func messageDiagnosticReport(root string, active, introduced, resolved, suggestions []string) diagnosticReport {
	return diagnosticReport{WorkspaceRoot: root, Status: "available", Active: diagnosticMessages(active), Introduced: diagnosticMessages(introduced), Resolved: diagnosticMessages(resolved), Suggestions: diagnosticMessages(suggestions)}
}

func verificationDiagnosticReport(root string, result *operation.VerifyRes) (diagnosticReport, bool) {
	if result == nil {
		return diagnosticReport{}, false
	}
	verified := false
	partial := false
	for _, coverage := range result.Coverage {
		if coverage.Status == "verified" {
			verified = true
		} else {
			partial = true
		}
	}
	if !verified {
		return diagnosticReport{}, false
	}
	status := "available"
	if partial {
		status = "partial"
	}
	return diagnosticReport{WorkspaceRoot: root, Status: status, Coverage: result.Coverage, Active: result.Diagnostics}, true
}

func unavailableDiagnosticReport(status, reason string) map[string]any {
	return map[string]any{"status": status, "reason": reason}
}
