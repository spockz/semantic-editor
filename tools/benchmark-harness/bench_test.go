// Package main verifies the benchmark task parsing, extraction, and oracle behavior.
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAllBenchmarkFixtures(t *testing.T) {
	benchDir := filepath.Join("..", "..", "testdata", "bench")
	entries, err := os.ReadDir(benchDir)
	if err != nil {
		t.Fatalf("read bench dir: %v", err)
	}

	expectedTasks := 10
	foundTasks := 0

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".txtar" {
			continue
		}

		foundTasks++
		cleanPath := filepath.Clean(filepath.Join(benchDir, entry.Name()))
		data, err := os.ReadFile(cleanPath)
		if err != nil {
			t.Fatalf("read fixture %s: %v", entry.Name(), err)
		}

		task, err := ParseTask(data)
		if err != nil {
			t.Fatalf("parse fixture %s: %v", entry.Name(), err)
		}

		if task.Metadata.TaskID == "" {
			t.Errorf("fixture %s missing task_id", entry.Name())
		}
		if task.Metadata.Instruction == "" {
			t.Errorf("fixture %s missing instruction", entry.Name())
		}
		if task.Metadata.Category == "" {
			t.Errorf("fixture %s missing category", entry.Name())
		}

		// Verify no want/ files inside archive (crucial invariant)
		for _, file := range task.Archive.Files {
			normalized := filepath.ToSlash(file.Name)
			if strings.HasPrefix(normalized, "want/") {
				t.Errorf("fixture %s contains golden want/ file %s: violates anti-leakage invariant", entry.Name(), file.Name)
			}
		}

		// Verify extraction to temp dir
		tmpDir := t.TempDir()
		if err := task.ExtractTo(tmpDir); err != nil {
			t.Fatalf("extract task %s: %v", entry.Name(), err)
		}

		// Verify go.mod was extracted
		if _, err := os.Stat(filepath.Join(tmpDir, "go.mod")); err != nil {
			t.Errorf("task %s failed to extract go.mod: %v", entry.Name(), err)
		}
	}

	if foundTasks < expectedTasks {
		t.Errorf("expected at least %d benchmark tasks, found %d", expectedTasks, foundTasks)
	}
}

func TestOracleMutationPolicyEnforcement(t *testing.T) {
	fixturePath := filepath.Clean(filepath.Join("..", "..", "testdata", "bench", "task_01_rename_local.txtar"))
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	task, err := ParseTask(data)
	if err != nil {
		t.Fatalf("parse task: %v", err)
	}

	tmpDir := t.TempDir()
	if err := task.ExtractTo(tmpDir); err != nil {
		t.Fatalf("extract: %v", err)
	}

	// Case 1: Unauthorized change to go.mod
	res, err := Evaluate(context.Background(), task, tmpDir, []string{"go.mod"})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if res.Passed {
		t.Errorf("expected failure on disallowed go.mod change")
	}
	if res.FailureStage != "level_1_mutation_policy" {
		t.Errorf("expected level_1_mutation_policy failure, got %s", res.FailureStage)
	}
}

func TestOracleAdversarialCheating(t *testing.T) {
	fixturePath := filepath.Clean(filepath.Join("..", "..", "testdata", "bench", "task_01_rename_local.txtar"))
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	task, err := ParseTask(data)
	if err != nil {
		t.Fatalf("parse task: %v", err)
	}

	tmpDir := t.TempDir()
	if err := task.ExtractTo(tmpDir); err != nil {
		t.Fatalf("extract: %v", err)
	}

	// Unmodified initial fixture fails Level 2 AST (newName missing and oldName present)
	res, err := Evaluate(context.Background(), task, tmpDir, nil)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if res.Passed {
		t.Errorf("expected unmodified fixture to fail oracle")
	}
	if res.FailureStage != "level_2_ast" {
		t.Errorf("expected level_2_ast failure, got %s", res.FailureStage)
	}
}

func TestPromptVariantsParsing(t *testing.T) {
	fixturePath := filepath.Clean(filepath.Join("..", "..", "testdata", "scripts", "generate_template_main.txtar"))
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	task, err := ParseTask(data)
	if err != nil {
		t.Fatalf("parse task: %v", err)
	}

	if len(task.Metadata.PromptVariants) != 2 {
		t.Fatalf("expected 2 prompt variants, got %d (%v)", len(task.Metadata.PromptVariants), task.Metadata.PromptVariants)
	}
	if !strings.Contains(task.Metadata.PromptVariants["crypto_rand"], "crypto/rand") {
		t.Errorf("expected crypto/rand in crypto_rand variant, got: %s", task.Metadata.PromptVariants["crypto_rand"])
	}
}

func TestCollectAvailableBenchmarks(t *testing.T) {
	benchDir := filepath.Clean(filepath.Join("..", "..", "testdata", "bench"))
	benchmarks, err := CollectAvailableBenchmarks(benchDir)
	if err != nil {
		t.Fatalf("CollectAvailableBenchmarks failed: %v", err)
	}

	if len(benchmarks) < 10 {
		t.Fatalf("expected at least 10 benchmarks, got %d", len(benchmarks))
	}

	foundGenerateTemplate := false
	for _, b := range benchmarks {
		if b.TaskID == "task-07-generate-template-main" {
			foundGenerateTemplate = true
			if len(b.PromptVariants) != 2 {
				t.Errorf("expected 2 prompt variants for task-07-generate-template-main, got %d", len(b.PromptVariants))
			}
		}
	}
	if !foundGenerateTemplate {
		t.Errorf("expected task-07-generate-template-main in collected benchmarks")
	}

	var buf strings.Builder
	PrintBenchmarksList(&buf, benchmarks)
	output := buf.String()
	if !strings.Contains(output, "task-01-rename-local") {
		t.Errorf("expected task-01-rename-local in formatted output")
	}
}
