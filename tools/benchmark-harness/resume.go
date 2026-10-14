// resume.go persists one benchmark outcome per job and validates a saved plan before resuming so interrupted runs can continue without repeating completed provider work.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type jobCheckpoint struct {
	FormatVersion int                   `json:"format_version"`
	Job           jobCheckpointMetadata `json:"job"`
	Report        *BenchmarkReport      `json:"report"`
}

type resumeFingerprintJob struct {
	ProviderTransport     string `json:",omitempty"`
	ID                    string
	PairID                string
	TaskID                string
	FixturePath           string
	Context               string
	PromptVariant         string
	Target                Target
	Arm                   ArmType
	Repeat                int
	Policy                SemeditArmRestriction
	Prompt                string
	Followups             []string
	StagedFollowups       []StagedFollowup
	MCPServerInstructions MCPServerInstructionMode
	FixtureSHA256         string
}

type jobCheckpointMetadata struct {
	ID                    string                   `json:"id"`
	PairID                string                   `json:"pair_id,omitempty"`
	TaskID                string                   `json:"task_id"`
	FixturePath           string                   `json:"fixture_path"`
	Context               string                   `json:"context"`
	PromptVariant         string                   `json:"prompt_variant,omitempty"`
	Target                Target                   `json:"target"`
	Arm                   ArmType                  `json:"arm"`
	Repeat                int                      `json:"repeat"`
	Policy                SemeditArmRestriction    `json:"policy,omitempty"`
	Prompt                string                   `json:"resolved_prompt,omitempty"`
	Followups             []string                 `json:"followups,omitempty"`
	StagedFollowups       []StagedFollowup         `json:"staged_followups,omitempty"`
	MCPServerInstructions MCPServerInstructionMode `json:"mcp_server_instructions,omitempty"`
}

type resumeFingerprintOptions struct {
	Timeout               time.Duration
	Concurrency           int
	SemeditArmRestriction SemeditArmRestriction
	SemeditPrewarmVerify  bool
	MCPServerInstructions []MCPServerInstructionMode
	Provenance            ProvenanceSet
}

type runManifest struct {
	FormatVersion int       `json:"format_version"`
	RunID         string    `json:"run_id"`
	Fingerprint   string    `json:"plan_fingerprint"`
	JobIDs        []string  `json:"job_ids"`
	CreatedAt     time.Time `json:"created_at"`
}

func buildRunManifest(plan *BenchmarkPlan) (runManifest, error) {
	if plan == nil {
		return runManifest{}, fmt.Errorf("benchmark plan is nil")
	}
	fingerprintJobs := make([]resumeFingerprintJob, 0, len(plan.Jobs))
	jobIDs := make([]string, 0, len(plan.Jobs))
	for _, job := range plan.Jobs {
		fixtureData, err := os.ReadFile(job.FixturePath)
		if err != nil {
			return runManifest{}, fmt.Errorf("read fixture %s for resume fingerprint: %w", job.FixturePath, err)
		}
		fixtureDigest := sha256.Sum256(fixtureData)
		fingerprintJobs = append(fingerprintJobs, resumeFingerprintJob{
			ProviderTransport:     providerContinuationTransport(job.Target),
			ID:                    job.ID,
			PairID:                job.PairID,
			TaskID:                job.TaskID,
			FixturePath:           job.FixturePath,
			Context:               job.Context,
			PromptVariant:         job.PromptVariant,
			Target:                job.Target,
			Arm:                   job.Arm,
			Repeat:                job.Repeat,
			Policy:                job.Policy,
			Prompt:                job.Execution.Prompt,
			Followups:             slices.Clone(job.Execution.Followups),
			StagedFollowups:       slices.Clone(job.Execution.StagedFollowups),
			MCPServerInstructions: job.MCPServerInstructions,
			FixtureSHA256:         hex.EncodeToString(fixtureDigest[:]),
		})
		jobIDs = append(jobIDs, job.ID)
	}
	input := struct {
		RunID      string
		Options    resumeFingerprintOptions
		Exclusions []FixtureExclusion
		Jobs       []resumeFingerprintJob
	}{
		RunID: plan.Options.RunID,
		Options: resumeFingerprintOptions{
			Timeout:               plan.Options.Timeout,
			Concurrency:           plan.Options.Concurrency,
			SemeditArmRestriction: plan.Options.SemeditArmRestriction,
			SemeditPrewarmVerify:  plan.Options.SemeditPrewarmVerify,
			MCPServerInstructions: plan.Options.MCPServerInstructions,
			Provenance:            plan.Options.Provenance,
		},
		Exclusions: plan.Exclusions,
		Jobs:       fingerprintJobs,
	}
	data, err := json.Marshal(input)
	if err != nil {
		return runManifest{}, fmt.Errorf("marshal resume plan fingerprint: %w", err)
	}
	fingerprint := sha256.Sum256(data)
	return runManifest{
		FormatVersion: 1,
		RunID:         plan.Options.RunID,
		Fingerprint:   hex.EncodeToString(fingerprint[:]),
		JobIDs:        jobIDs,
		CreatedAt:     time.Now().UTC(),
	}, nil
}

func prepareBenchmarkRunDir(plan *BenchmarkPlan) (string, error) {
	if plan == nil {
		return "", fmt.Errorf("benchmark plan is nil")
	}
	if plan.Options.OutDir == "" {
		if plan.Options.Resume {
			return "", fmt.Errorf("--resume requires --out-dir")
		}
		return "", nil
	}
	if err := validateBenchmarkRunID(plan.Options.RunID); err != nil {
		return "", err
	}
	expected, err := buildRunManifest(plan)
	if err != nil {
		return "", err
	}
	runDir := filepath.Join(plan.Options.OutDir, plan.Options.RunID)
	if !plan.Options.Resume {
		runDir, err = createBenchmarkRunDir(plan.Options.OutDir, plan.Options.RunID)
		if err != nil {
			return "", err
		}
		manifestErr := saveRunManifest(runDir, expected)
		if manifestErr != nil {
			if cleanupErr := os.Remove(runDir); cleanupErr != nil {
				return "", errors.Join(fmt.Errorf("save run manifest: %w", manifestErr), fmt.Errorf("remove empty run directory: %w", cleanupErr))
			}
			return "", fmt.Errorf("save run manifest: %w", manifestErr)
		}
		return runDir, nil
	}
	info, err := os.Stat(runDir)
	if err != nil {
		return "", fmt.Errorf("open run directory for resume: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("resume path %s is not a directory", runDir)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	// #nosec G304 -- manifestPath stays under the caller-selected run directory after run ID validation.
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("read run manifest: %w", err)
	}
	var saved runManifest
	if err := json.Unmarshal(data, &saved); err != nil {
		return "", fmt.Errorf("decode run manifest: %w", err)
	}
	if saved.FormatVersion != expected.FormatVersion || saved.RunID != expected.RunID || saved.Fingerprint != expected.Fingerprint || !slices.Equal(saved.JobIDs, expected.JobIDs) {
		return "", fmt.Errorf("saved run manifest does not match the resolved benchmark plan")
	}
	return runDir, nil
}

func saveRunManifest(runDir string, manifest runManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run manifest: %w", err)
	}
	if err := pipeline.WriteAtomic(filepath.Join(runDir, "manifest.json"), data); err != nil {
		return fmt.Errorf("write run manifest: %w", err)
	}
	return nil
}

func jobCheckpointPath(runDir, jobID string) (string, error) {
	if jobID == "" || filepath.Base(jobID) != jobID || strings.ContainsAny(jobID, "/\\") {
		return "", fmt.Errorf("invalid benchmark job id %q", jobID)
	}
	return filepath.Join(runDir, "jobs", jobID+".json"), nil
}

func loadJobCheckpoints(runDir string, plan *BenchmarkPlan) (map[string]*RunResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("benchmark plan is nil")
	}
	results := make(map[string]*RunResult, len(plan.Jobs))
	for _, job := range plan.Jobs {
		path, err := jobCheckpointPath(runDir, job.ID)
		if err != nil {
			return nil, err
		}
		// #nosec G304 -- jobCheckpointPath validates each job ID before joining it under runDir.
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read checkpoint for job %s: %w", job.ID, err)
		}
		var record jobCheckpoint
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, fmt.Errorf("decode checkpoint for job %s: %w", job.ID, err)
		}
		if record.FormatVersion != 1 || record.Job.ID != job.ID || record.Report == nil || len(record.Report.Runs) != 1 || record.Report.Runs[0] == nil || record.Report.Runs[0].JobID != job.ID {
			return nil, fmt.Errorf("checkpoint for job %s has an invalid result identity", job.ID)
		}
		results[job.ID] = record.Report.Runs[0]
	}
	return results, nil
}

func pendingBenchmarkPlan(plan *BenchmarkPlan, completed map[string]*RunResult) *BenchmarkPlan {
	pending := &BenchmarkPlan{Exclusions: append([]FixtureExclusion(nil), plan.Exclusions...), Options: plan.Options}
	for _, job := range plan.Jobs {
		if _, exists := completed[job.ID]; !exists {
			pending.Jobs = append(pending.Jobs, job)
		}
	}
	return pending
}

func mergeJobOutcomes(plan *BenchmarkPlan, completed map[string]*RunResult, fresh []JobOutcome) []JobOutcome {
	freshByID := make(map[string]JobOutcome, len(fresh))
	for _, outcome := range fresh {
		freshByID[outcome.Job.ID] = outcome
	}
	outcomes := make([]JobOutcome, 0, len(plan.Jobs))
	for _, job := range plan.Jobs {
		if result, exists := completed[job.ID]; exists {
			outcomes = append(outcomes, JobOutcome{Job: job, Result: result})
			continue
		}
		if outcome, exists := freshByID[job.ID]; exists {
			outcomes = append(outcomes, outcome)
		}
	}
	return outcomes
}

func saveJobOutcomeCheckpoint(runDir string, outcome JobOutcome) (string, error) {
	if outcome.Result == nil {
		return "", fmt.Errorf("job %s result is nil", outcome.Job.ID)
	}
	if outcome.Result.JobID != outcome.Job.ID {
		return "", fmt.Errorf("job outcome identity mismatch: job %s has result %s", outcome.Job.ID, outcome.Result.JobID)
	}
	path, err := jobCheckpointPath(runDir, outcome.Job.ID)
	if err != nil {
		return "", err
	}
	mkdirErr := os.MkdirAll(filepath.Dir(path), 0o750)
	if mkdirErr != nil {
		return "", fmt.Errorf("create job checkpoint directory: %w", mkdirErr)
	}
	record := jobCheckpoint{
		FormatVersion: 1,
		Job: jobCheckpointMetadata{
			ID:                    outcome.Job.ID,
			PairID:                outcome.Job.PairID,
			TaskID:                outcome.Job.TaskID,
			FixturePath:           outcome.Job.FixturePath,
			Context:               outcome.Job.Context,
			PromptVariant:         outcome.Job.PromptVariant,
			Target:                outcome.Job.Target,
			Arm:                   outcome.Job.Arm,
			Repeat:                outcome.Job.Repeat,
			Policy:                outcome.Job.Policy,
			Prompt:                outcome.Job.Execution.Prompt,
			Followups:             slices.Clone(outcome.Job.Execution.Followups),
			StagedFollowups:       slices.Clone(outcome.Job.Execution.StagedFollowups),
			MCPServerInstructions: outcome.Job.MCPServerInstructions,
		},
		Report: &BenchmarkReport{Timestamp: time.Now().UTC(), Runs: []*RunResult{outcome.Result}},
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal job %s checkpoint: %w", outcome.Job.ID, err)
	}
	if err := pipeline.WriteAtomic(path, data); err != nil {
		return "", fmt.Errorf("write job %s checkpoint: %w", outcome.Job.ID, err)
	}
	return path, nil
}
