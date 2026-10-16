// Package mcp_test benchmarks filesystem renames and the next MCP reaction so deferred gopls restarts cannot hide behind fast reads.

package mcp_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/spockz/semantic-editor/internal/verification"
)

type fileRenameObservation struct {
	rename   time.Duration
	reaction time.Duration
	total    time.Duration
	result   json.RawMessage
}

func BenchmarkFileRenameFilesystemCost(b *testing.B) {
	for _, packages := range []int{1, 20} {
		b.Run(fmt.Sprintf("packages_%d", packages), func(b *testing.B) {
			benchmarkFileRenameCost(b, packages, "")
		})
	}
}

func BenchmarkMCPFileRenameCost(b *testing.B) {
	for _, packages := range []int{1, 20} {
		b.Run(fmt.Sprintf("packages_%d", packages), func(b *testing.B) {
			for _, reaction := range []string{"semantic_lookup", "semantic_verify"} {
				b.Run(reaction, func(b *testing.B) { benchmarkFileRenameCost(b, packages, reaction) })
			}
		})
	}
}

func benchmarkFileRenameCost(b *testing.B, packages int, reaction string) {
	fixture := newWarmMCPCostFixture(b, packages)
	original := filepath.Join(fixture.root, "value.go")
	content, err := os.ReadFile(original)
	if err != nil {
		b.Fatal(err)
	}
	var client *warmMCPCostClient
	if reaction != "" {
		client = newWarmMCPCostClient(b, fixture.root)
		client.call(b, "initialize", map[string]any{})
		assertFileRenameVerification(b, client.toolResult(b, "semantic_verify", fileRenameArguments("semantic_verify", "value.go")))
		assertFileRenameReaction(b, reaction, "value.go", client.toolResult(b, reaction, fileRenameArguments(reaction, "value.go")))
		if starts := fileRenameGoplsStarts(b, fixture.counter); starts != 1 {
			b.Fatalf("file rename setup is not warm: gopls starts=%d", starts)
		}
	}
	samples := make([]fileRenameObservation, 0, b.N)
	b.ResetTimer()
	for index := range b.N {
		b.StopTimer()
		from, to := "value.go", "renamed.go"
		if index%2 != 0 {
			from, to = to, from
		}
		source, destination := filepath.Join(fixture.root, from), filepath.Join(fixture.root, to)
		if _, err := os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
			b.Fatalf("rename destination must be absent: %s: %v", destination, err)
		}
		b.StartTimer()
		observed := measureFileRenameCost(b, client, reaction, source, destination)
		b.StopTimer()
		assertFileRenameState(b, source, destination, content)
		if client != nil {
			assertFileRenameContinuity(b, client, fixture, reaction, to, observed)
		}
		samples = append(samples, observed)
		b.StartTimer()
	}
	b.StopTimer()
	reportFileRenameCost(b, samples, reaction)
}

func measureFileRenameCost(b *testing.B, client *warmMCPCostClient, reaction, source, destination string) fileRenameObservation {
	b.Helper()
	started := time.Now()
	if err := os.Rename(source, destination); err != nil {
		b.Fatal(err)
	}
	renamed := time.Now()
	observed := fileRenameObservation{rename: renamed.Sub(started), total: renamed.Sub(started)}
	if client != nil {
		observed.result = client.toolResult(b, reaction, fileRenameArguments(reaction, filepath.Base(destination)))
		completed := time.Now()
		observed.reaction, observed.total = completed.Sub(renamed), completed.Sub(started)
	}
	return observed
}

func fileRenameArguments(reaction, file string) map[string]any {
	if reaction == "semantic_lookup" {
		return map[string]any{"file": file, "symbol": "Value"}
	}
	return map[string]any{"path": ".", "language": "go", "check_only": true}
}

func assertFileRenameState(b *testing.B, source, destination string, expected []byte) {
	b.Helper()
	if _, err := os.Lstat(source); !errors.Is(err, os.ErrNotExist) {
		b.Fatalf("renamed source must be absent: %s: %v", source, err)
	}
	content, err := os.ReadFile(destination)
	if err != nil {
		b.Fatal(err)
	}
	if !bytes.Equal(content, expected) {
		b.Fatalf("rename changed contents of %s", destination)
	}
}

func assertFileRenameReaction(b *testing.B, reaction, file string, raw json.RawMessage) {
	b.Helper()
	if reaction == "semantic_verify" {
		assertFileRenameVerification(b, raw)
		return
	}
	var response struct {
		Result struct {
			File      string `json:"file"`
			Symbol    string `json:"symbol"`
			Ambiguous bool   `json:"ambiguous"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		b.Fatal(err)
	}
	if response.Result.Ambiguous || response.Result.File != file || response.Result.Symbol != "Value" {
		b.Fatalf("lookup did not resolve Value in renamed file %s: %s", file, raw)
	}
}

func assertFileRenameVerification(b *testing.B, raw json.RawMessage) {
	b.Helper()
	var response struct {
		Result verification.Result `json:"result"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		b.Fatal(err)
	}
	if response.Result.Failure != "" || len(response.Result.Diagnostics) != 0 {
		b.Fatalf("renamed workspace did not verify cleanly: %s", raw)
	}
	for _, coverage := range response.Result.Coverage {
		if coverage.Language == "go" && coverage.Status == "verified" {
			return
		}
	}
	b.Fatalf("verification omitted completed Go coverage: %s", raw)
}

func assertFileRenameContinuity(b *testing.B, client *warmMCPCostClient, fixture managedMCPFixture, reaction, file string, observed fileRenameObservation) {
	b.Helper()
	assertFileRenameReaction(b, reaction, file, observed.result)
	synchronization := time.Duration(0)
	if reaction == "semantic_lookup" {
		started := time.Now()
		assertFileRenameVerification(b, client.toolResult(b, "semantic_verify", fileRenameArguments("semantic_verify", file)))
		synchronization = time.Since(started)
	}
	count := fileRenameGoplsStarts(b, fixture.counter)
	if count != 1 {
		b.Logf("file rename observation: rename_ms=%.4f reaction_ms=%.4f synchronization_ms=%.4f gopls_starts=%d",
			float64(observed.rename)/float64(time.Millisecond), float64(observed.reaction)/float64(time.Millisecond),
			float64(synchronization)/float64(time.Millisecond), count)
		b.Fatalf("file rename broke warm gopls continuity after the first observed rename: starts=%d, want 1; refusing to report a warm p99", count)
	}
}

func reportFileRenameCost(b *testing.B, samples []fileRenameObservation, reaction string) {
	if len(samples) == 0 {
		b.Fatal("file rename benchmark has no measured samples")
	}
	totals, renames, reactions := make([]time.Duration, len(samples)), make([]time.Duration, len(samples)), make([]time.Duration, len(samples))
	for index, observed := range samples {
		totals[index], renames[index], reactions[index] = observed.total, observed.rename, observed.reaction
	}
	slices.Sort(totals)
	slices.Sort(renames)
	slices.Sort(reactions)
	budget := 50 * time.Millisecond
	if reaction == "semantic_verify" {
		budget = time.Second
	}
	reportWarmMCPCost(b, totals, budget)
	tail := (99*len(samples)+99)/100 - 1
	b.ReportMetric(float64(renames[tail])/float64(time.Millisecond), "rename_p99_ms")
	b.ReportMetric(float64(reactions[tail])/float64(time.Millisecond), "reaction_p99_ms")
	b.ReportMetric(float64(budget)/float64(time.Millisecond), "budget_ms")
	starts := 0
	if reaction != "" {
		starts = 1
	}
	b.ReportMetric(float64(starts), "gopls_starts")
	if reaction == "semantic_lookup" {
		b.ReportMetric(float64(reactions[tail])/float64(10*time.Millisecond), "reaction_p99/read_budget")
		if os.Getenv("SEMEDIT_BENCH_ENFORCE_BUDGETS") == "1" && reactions[tail] > 10*time.Millisecond {
			b.Errorf("first read p99 %s exceeds 10 ms", reactions[tail])
		}
	}
}

func fileRenameGoplsStarts(b *testing.B, counter string) int {
	b.Helper()
	starts, err := os.ReadFile(counter)
	if err != nil {
		b.Fatal(err)
	}
	return strings.Count(string(starts), "start\n")
}
