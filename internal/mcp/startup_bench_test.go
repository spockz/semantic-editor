// Package mcp_test benchmarks cold startup and semantic verification through the actual MCP subprocess.

package mcp_test

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func BenchmarkMCPColdStartupVerify(b *testing.B) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		b.Fatal(err)
	}
	binaryDir := b.TempDir()
	binary := filepath.Join(binaryDir, "semedit")
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Dir = repoRoot
	build.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
	if output, err := build.CombinedOutput(); err != nil {
		b.Fatalf("build semedit: %v: %s", err, output)
	}
	b.ReportAllocs()
	var totals coldMCPBenchmarkDurations
	b.ResetTimer()
	for range b.N {
		b.StopTimer()
		fixture := newColdMCPBenchmarkFixture(b)
		process := newColdMCPBenchmarkProcess(b, binary, fixture)
		b.StartTimer()
		iteration := runColdMCPBenchmarkProtocol(b, process)
		totals.add(iteration)
	}
	if b.N > 0 {
		denom := float64(b.N)
		b.ReportMetric(float64(totals.startup)/denom/float64(time.Millisecond), "startup_ms/op")
		b.ReportMetric(float64(totals.initialize)/denom/float64(time.Millisecond), "initialize_ms/op")
		b.ReportMetric(float64(totals.verify)/denom/float64(time.Millisecond), "verify_ms/op")
	}
}

type coldMCPBenchmarkFixture struct {
	workspace string
	goDirs    map[string]string
}

type coldMCPBenchmarkProcess struct {
	command *exec.Cmd
	stdin   io.WriteCloser
	reader  *bufio.Reader
	stderr  *bytes.Buffer
}

type coldMCPBenchmarkDurations struct {
	startup    time.Duration
	initialize time.Duration
	verify     time.Duration
}

func (durations *coldMCPBenchmarkDurations) add(iteration coldMCPBenchmarkDurations) {
	durations.startup += iteration.startup
	durations.initialize += iteration.initialize
	durations.verify += iteration.verify
}

func newColdMCPBenchmarkFixture(b *testing.B) coldMCPBenchmarkFixture {
	b.Helper()
	workspace := filepath.Join(b.TempDir(), "workspace")
	if err := os.MkdirAll(workspace, 0o700); err != nil {
		b.Fatal(err)
	}
	writeColdMCPFixtureFile(b, filepath.Join(workspace, "go.mod"), "module example.com/coldverify\n\ngo 1.27.1\n")
	writeColdMCPFixtureFile(b, filepath.Join(workspace, "main.go"), "package coldverify\n\nfunc Value() int { return 1 }\n")
	envRoot := filepath.Join(b.TempDir(), "env")
	goDirs := coldMCPGoDirectories(envRoot)
	for _, dir := range goDirs {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			b.Fatal(err)
		}
	}
	entries, err := os.ReadDir(goDirs["GOMODCACHE"])
	if err != nil || len(entries) != 0 {
		b.Fatalf("module cache is not empty before iteration: entries=%d err=%v", len(entries), err)
	}
	return coldMCPBenchmarkFixture{workspace: workspace, goDirs: goDirs}
}

func writeColdMCPFixtureFile(b *testing.B, path, content string) {
	b.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		b.Fatal(err)
	}
}

func coldMCPGoDirectories(envRoot string) map[string]string {
	return map[string]string{
		"GOENV":      filepath.Join(envRoot, "goenv"),
		"GOCACHE":    filepath.Join(envRoot, "gocache"),
		"GOMODCACHE": filepath.Join(envRoot, "gomodcache"),
		"GOTMPDIR":   filepath.Join(envRoot, "gotmp"),
		"GOBIN":      filepath.Join(envRoot, "gobin"),
		"GOPATH":     filepath.Join(envRoot, "gopath"),
		"HOME":       filepath.Join(envRoot, "home"),
	}
}

func newColdMCPBenchmarkProcess(b *testing.B, binary string, fixture coldMCPBenchmarkFixture) coldMCPBenchmarkProcess {
	b.Helper()
	command := exec.Command(binary, "mcp", "--profile", "full")
	command.Dir = fixture.workspace
	command.Env = coldGoEnv(fixture.goDirs)
	stdin, err := command.StdinPipe()
	if err != nil {
		b.Fatal(err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		b.Fatal(err)
	}
	stderr := &bytes.Buffer{}
	command.Stderr = stderr
	return coldMCPBenchmarkProcess{command: command, stdin: stdin, reader: bufio.NewReader(stdout), stderr: stderr}
}

func runColdMCPBenchmarkProtocol(b *testing.B, process coldMCPBenchmarkProcess) coldMCPBenchmarkDurations {
	started := time.Now()
	if err := process.command.Start(); err != nil {
		b.Fatal(err)
	}
	completed := false
	defer func() {
		if !completed {
			abortColdMCPBenchmarkProcess(b, process)
		}
	}()
	spawned := time.Now()
	writeColdMCPRequest(b, process, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`+"\n", "initialize")
	initLine := readColdMCPResponse(b, process, "initialize")
	initAt := time.Now()
	if !strings.Contains(initLine, `"id":1`) || strings.Contains(initLine, `"error":`) {
		b.Fatalf("invalid initialize response: %s", initLine)
	}
	writeColdMCPRequest(b, process, "{\"jsonrpc\":\"2.0\",\"method\":\"notifications/initialized\"}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_verify\",\"arguments\":{\"path\":\".\",\"language\":\"go\",\"check_only\":true}}}\n", "semantic_verify")
	verifyLine := readColdMCPResponse(b, process, "semantic_verify")
	verifyAt := time.Now()
	b.StopTimer()
	if !strings.Contains(verifyLine, `"id":2`) || strings.Contains(verifyLine, `"isError":true`) || strings.Contains(verifyLine, `"error":`) {
		b.Fatalf("semantic_verify failed: %s", verifyLine)
	}
	finishColdMCPBenchmarkProcess(b, process)
	completed = true
	return coldMCPBenchmarkDurations{startup: spawned.Sub(started), initialize: initAt.Sub(spawned), verify: verifyAt.Sub(initAt)}
}

func abortColdMCPBenchmarkProcess(b *testing.B, process coldMCPBenchmarkProcess) {
	b.Helper()
	if process.command.ProcessState != nil {
		return
	}
	if err := process.stdin.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		b.Errorf("close MCP stdin after protocol failure: %v", err)
	}
	if err := process.command.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		b.Errorf("kill MCP process after protocol failure: %v", err)
	}
	if err := process.command.Wait(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		b.Errorf("wait for MCP process after protocol failure: %v", err)
	}
}

func writeColdMCPRequest(b *testing.B, process coldMCPBenchmarkProcess, request, operation string) {
	b.Helper()
	if _, err := io.WriteString(process.stdin, request); err != nil {
		failColdMCPProtocol(b, process, "write "+operation, err)
	}
}

func readColdMCPResponse(b *testing.B, process coldMCPBenchmarkProcess, operation string) string {
	b.Helper()
	response, err := process.reader.ReadString('\n')
	if err != nil {
		failColdMCPProtocol(b, process, "read "+operation+" response", err)
	}
	return response
}

func failColdMCPProtocol(b *testing.B, process coldMCPBenchmarkProcess, operation string, cause error) {
	b.Helper()
	abortColdMCPBenchmarkProcess(b, process)
	b.Fatalf("%s: %v; stderr: %s", operation, cause, process.stderr.String())
}

func finishColdMCPBenchmarkProcess(b *testing.B, process coldMCPBenchmarkProcess) {
	b.Helper()
	if err := process.stdin.Close(); err != nil {
		b.Fatalf("close MCP stdin: %v", err)
	}
	if err := process.command.Wait(); err != nil {
		b.Fatalf("MCP process exit: %v; stderr: %s", err, process.stderr.String())
	}
}

func coldGoEnv(dirs map[string]string) []string {
	env := make([]string, 0, len(dirs)+4)
	env = append(env, "PATH="+os.Getenv("PATH"), "GOWORK=off", "GOFLAGS=")
	for name, value := range dirs {
		env = append(env, name+"="+value)
	}
	return env
}
