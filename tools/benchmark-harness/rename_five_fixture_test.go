// This file verifies the one-turn five-location rename fixture so its oracle detects incomplete edits while preserving runnable baseline input.

package main

import (
	"context"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFiveLocationRenameFixtureRequiresAllFiveRenames(t *testing.T) {
	data, readErr := os.ReadFile(filepath.Join("..", "..", "testdata", "bench", "task_12_rename_scale_05_multifile.txtar"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	task, parseErr := ParseTask(data)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	if task.Metadata.TaskID != "task-12-rename-scale-05-multifile" || task.Metadata.InteractiveMode != "" || !strings.Contains(task.Metadata.Instruction, "Perform these 5 renames consecutively, possibly batched if you know how") {
		t.Fatalf("unexpected one-turn rename task metadata: %+v", task.Metadata)
	}
	workDir := t.TempDir()
	if err := task.ExtractTo(workDir); err != nil {
		t.Fatal(err)
	}
	initialTest := exec.CommandContext(t.Context(), "go", "test", "./...")
	initialTest.Dir = workDir
	if output, err := initialTest.CombinedOutput(); err != nil {
		t.Fatalf("initial fixture does not pass go test: %v\n%s", err, output)
	}
	renames := []struct {
		oldName, newName, callFile string
	}{
		{"LoadInput", "ReadRequest", "pipeline/stage_input.go"},
		{"CheckInput", "ValidateRequest", "pipeline/stage_validation.go"},
		{"NormalizeValue", "CanonicalizeValue", "pipeline/stage_normalization.go"},
		{"MapFields", "ProjectFields", "pipeline/stage_mapping.go"},
		{"RouteValue", "DispatchValue", "pipeline/stage_routing.go"},
	}
	changed := make([]string, 0, 1+len(renames))
	changed = append(changed, "pipeline/processor.go")
	for index, rename := range renames {
		for _, relative := range []string{"pipeline/processor.go", rename.callFile} {
			path := filepath.Join(workDir, filepath.FromSlash(relative))
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(source), rename.oldName) {
				t.Fatalf("%s does not contain %s", relative, rename.oldName)
			}
			updated := strings.ReplaceAll(string(source), rename.oldName, rename.newName)
			if err := pipeline.WriteAtomic(path, []byte(updated)); err != nil {
				t.Fatal(err)
			}
		}
		changed = append(changed, rename.callFile)
		if index == len(renames)-2 {
			result, err := Evaluate(context.Background(), task, workDir, changed)
			if err != nil {
				t.Fatal(err)
			}
			if result.Passed {
				t.Fatal("oracle accepted only four of five renames")
			}
		}
	}
	result, err := Evaluate(context.Background(), task, workDir, changed)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed {
		t.Fatalf("all five renames failed the oracle: %+v", result)
	}
}
