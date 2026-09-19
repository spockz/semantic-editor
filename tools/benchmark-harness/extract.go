package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func extractFixture(fixturePath, targetDir string) error {
	data, err := os.ReadFile(filepath.Clean(fixturePath))
	if err != nil {
		return fmt.Errorf("read fixture: %w", err)
	}
	task, err := ParseTask(data)
	if err != nil {
		return fmt.Errorf("parse task: %w", err)
	}
	if err := os.MkdirAll(targetDir, 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", targetDir, err)
	}
	if err := task.ExtractTo(targetDir); err != nil {
		return fmt.Errorf("extract to %s: %w", targetDir, err)
	}
	fmt.Printf("Extracted task %s to %s (%d files)\n", task.Metadata.TaskID, targetDir, len(task.Archive.Files))
	return nil
}
