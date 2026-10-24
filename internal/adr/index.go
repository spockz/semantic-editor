// Package adr owns deterministic rendering of the repository's authored ADR register.
package adr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spockz/semantic-editor/internal/atomicfile"
)

const (
	registerStart = "<!-- semedit-adr-register:start -->"
	registerEnd   = "<!-- semedit-adr-register:end -->"
)

// SyncIndex replaces only the marked decision register in the ADR README.
func SyncIndex(root string) error {
	records, err := ReadAll(root)
	if err != nil {
		return fmt.Errorf("validate ADR records before indexing: %w", err)
	}
	dir, err := Directory(root)
	if err != nil {
		return err
	}
	indexPath := filepath.Join(dir, "README.md")
	indexInfo, err := os.Lstat(indexPath)
	if err != nil {
		return fmt.Errorf("inspect ADR index: %w", err)
	}
	if !indexInfo.Mode().IsRegular() {
		return fmt.Errorf("ADR index must be a regular file")
	}
	index, err := os.ReadFile(indexPath) // #nosec G304 -- indexPath is fixed under the validated ADR directory
	if err != nil {
		return fmt.Errorf("read ADR index: %w", err)
	}
	updated, err := replaceRegister(string(index), records)
	if err != nil {
		return err
	}
	if updated == string(index) {
		return nil
	}
	if err := atomicfile.WriteAtomic(indexPath, []byte(updated)); err != nil {
		return fmt.Errorf("write ADR index: %w", err)
	}
	return nil
}

func replaceRegister(index string, records []Record) (string, error) {
	start := strings.Index(index, registerStart)
	end := strings.Index(index, registerEnd)
	if start < 0 || end < 0 || start >= end || strings.Count(index, registerStart) != 1 || strings.Count(index, registerEnd) != 1 {
		return "", fmt.Errorf("ADR index must contain exactly one ordered generated-register boundary pair")
	}
	startEnd := start + len(registerStart)
	rows := strings.Builder{}
	rows.WriteString("\n| ID | Title | Status | Date | Summary | Description | Role | Scope | Area | Languages |\n")
	rows.WriteString("| :--- | :--- | :---: | :---: | :--- | :--- | :--- | :--- | :--- | :--- |\n")
	for _, record := range records {
		fmt.Fprintf(&rows, "| [ADR-%s](%s) | %s | **%s** | %s | %s | %s | %s | %s | %s | %s |\n",
			record.ID, escapeCell(filepath.Base(record.Path)), escapeCell(record.Title), escapeCell(record.Status), record.Date,
			escapeCell(record.Summary), escapeCell(record.Description), record.Role, record.Scope, record.Area,
			escapeCell(strings.Join(record.Languages, ", ")))
	}
	return index[:startEnd] + rows.String() + index[end:], nil
}

func escapeCell(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "|", "\\|"), "\n", " ")
}
