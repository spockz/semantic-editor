// matrix_jobs.go isolates benchmark job creation and concurrent execution from CLI setup.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type matrixJob struct {
	taskBase string
	variant  string
	target   Target
	arm      ArmType
	task     *Task
	repeat   int
}

func buildMatrixJobs(benchDir string, taskBases []string, targets []Target, variants []string, arms []ArmType, repeats int) []matrixJob {
	var jobs []matrixJob
	for _, taskBase := range taskBases {
		for _, target := range targets {
			fixtureFile := resolveFixturePath(benchDir, taskBase, "small")
			// #nosec G304 -- reading benchmark fixture
			data, err := os.ReadFile(fixtureFile)
			if err != nil {
				fmt.Printf("❌ Failed reading fixture %s: %v\n", fixtureFile, err)
				continue
			}

			task, err := ParseTask(data)
			if err != nil {
				fmt.Printf("❌ Failed parsing fixture %s: %v\n", fixtureFile, err)
				continue
			}

			if len(declaredPromptVariants(task)) == 0 {
				fmt.Printf("Skipping benchmark %s: no non-empty prompt_variants declared in its txtar fixture\n", task.Metadata.TaskID)
				continue
			}

			for _, variant := range variants {
				effectiveVariants := matrixPromptVariants(task, variant)
				if len(effectiveVariants) == 0 {
					fmt.Printf("Skipping benchmark %s variant %s: prompt variant is not declared in its txtar fixture\n", task.Metadata.TaskID, variant)
					continue
				}

				for _, effVar := range effectiveVariants {
					for repeat := 1; repeat <= repeats; repeat++ {
						for _, arm := range arms {
							jobs = append(jobs, matrixJob{
								taskBase: taskBase,
								variant:  effVar,
								target:   target,
								arm:      arm,
								task:     task,
								repeat:   repeat,
							})
						}
					}
				}
			}
		}
	}

	return jobs
}

func executeMatrixJobs(runner *Runner, jobs []matrixJob, timeout time.Duration, concurrency int) []*RunResult {
	var mu sync.Mutex
	var allRuns []*RunResult

	jobChan := make(chan matrixJob, len(jobs))
	for _, j := range jobs {
		jobChan <- j
	}
	close(jobChan)

	var wg sync.WaitGroup
	for range concurrency {
		wg.Go(func() {
			for j := range jobChan {
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				res, execErr := runner.ExecuteAgentDriver(ctx, j.task, j.target, j.arm, j.variant)
				cancel()

				if execErr != nil {
					mu.Lock()
					fmt.Printf("❌ [Target=%s Task=%s Variant=%s Arm=%s] System error: %v\n",
						j.target.String(), j.taskBase, j.variant, j.arm, execErr)
					mu.Unlock()
					continue
				}

				res.Variant = j.variant
				res.Repeat = j.repeat
				if strings.HasPrefix(j.target.Model, "openrouter/") {
					if res.Provenance == nil {
						res.Provenance = make(ProvenanceSet)
					}
					res.Provenance["openrouter_auth"] = "environment"
					if strings.HasSuffix(j.target.Model, ":free") {
						res.Provenance["openrouter_catalog_as_of"] = openRouterFreeCatalogAsOf
						res.Provenance["openrouter_catalog_source"] = openRouterFreeCatalogURL
					}
				}
				status := "PASS"
				if !res.Success {
					status = "FAIL"
				}

				mu.Lock()
				allRuns = append(allRuns, res)
				fmt.Printf("==> [Target=%s Task=%s Variant=%s Arm=%s] %s (took %v, turns: %d, out_tokens: %d)\n",
					j.target.String(), j.taskBase, j.variant, j.arm, status,
					res.WallClock.Round(time.Millisecond), res.Turns, res.OutputTokens)
				mu.Unlock()
			}
		})
	}

	wg.Wait()
	return allRuns
}
