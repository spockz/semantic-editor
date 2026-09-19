package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func evaluateDir(fixturePath, targetDir string, modifiedFiles []string) error {
	data, err := os.ReadFile(filepath.Clean(fixturePath))
	if err != nil {
		return fmt.Errorf("read fixture: %w", err)
	}
	task, err := ParseTask(data)
	if err != nil {
		return fmt.Errorf("parse task: %w", err)
	}
	res, err := Evaluate(context.Background(), task, targetDir, modifiedFiles)
	if err != nil {
		return fmt.Errorf("evaluate: %w", err)
	}
	if !res.Passed {
		return fmt.Errorf("oracle failure at stage %s: %s", res.FailureStage, res.ErrorMessage)
	}
	fmt.Printf("✅ Oracle passed for %s in %s (Policy: OK | AST: OK | Build: OK | Test: OK)\n", task.Metadata.TaskID, targetDir)
	return nil
}
