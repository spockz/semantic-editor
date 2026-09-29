// Package mcp_test benchmarks cold startup and semantic verification through the actual MCP subprocess.

package mcp_test

import (
	"bufio"
	"bytes"
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
	var startupTotal, initializeTotal, verifyTotal time.Duration
	b.ResetTimer()
	for range b.N {
		b.StopTimer()
		workspace := filepath.Join(b.TempDir(), "workspace")
		if err := os.MkdirAll(workspace, 0o700); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/coldverify\n\ngo 1.27.1\n"), 0o600); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workspace, "main.go"), []byte("package coldverify\n\nfunc Value() int { return 1 }\n"), 0o600); err != nil {
			b.Fatal(err)
		}
		envRoot := filepath.Join(b.TempDir(), "env")
		dirs := map[string]string{
			"GOENV":      filepath.Join(envRoot, "goenv"),
			"GOCACHE":    filepath.Join(envRoot, "gocache"),
			"GOMODCACHE": filepath.Join(envRoot, "gomodcache"),
			"GOTMPDIR":   filepath.Join(envRoot, "gotmp"),
			"GOBIN":      filepath.Join(envRoot, "gobin"),
			"GOPATH":     filepath.Join(envRoot, "gopath"),
			"HOME":       filepath.Join(envRoot, "home"),
		}
		for _, dir := range dirs {
			if err := os.MkdirAll(dir, 0o700); err != nil {
				b.Fatal(err)
			}
		}
		entries, err := os.ReadDir(dirs["GOMODCACHE"])
		if err != nil || len(entries) != 0 {
			b.Fatalf("module cache is not empty before iteration: entries=%d err=%v", len(entries), err)
		}
		cmd := exec.Command(binary, "mcp", "--profile", "full")
		cmd.Dir = workspace
		cmd.Env = coldGoEnv(dirs)
		stdin, err := cmd.StdinPipe()
		if err != nil {
			b.Fatal(err)
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			b.Fatal(err)
		}
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		reader := bufio.NewReader(stdout)

		b.StartTimer()
		started := time.Now()
		if err := cmd.Start(); err != nil {
			b.Fatal(err)
		}
		spawned := time.Now()
		startupTotal += spawned.Sub(started)
		if _, err := io.WriteString(stdin, "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"2025-06-18\"}}\n"); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			b.Fatalf("write initialize: %v", err)
		}
		initLine, err := reader.ReadString('\n')
		initAt := time.Now()
		initializeTotal += initAt.Sub(spawned)
		if err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			b.Fatalf("read initialize response: %v; stderr: %s", err, stderr.String())
		}
		if !strings.Contains(initLine, "\"id\":1") || strings.Contains(initLine, "\"error\":") {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			b.Fatalf("invalid initialize response: %s", initLine)
		}
		if _, err := io.WriteString(stdin, "{\"jsonrpc\":\"2.0\",\"method\":\"notifications/initialized\"}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"semantic_verify\",\"arguments\":{\"path\":\".\",\"language\":\"go\",\"check_only\":true}}}\n"); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			b.Fatalf("write semantic_verify: %v", err)
		}
		verifyLine, err := reader.ReadString('\n')
		verifyAt := time.Now()
		verifyTotal += verifyAt.Sub(initAt)
		b.StopTimer()
		if err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			b.Fatalf("read semantic_verify response: %v; stderr: %s", err, stderr.String())
		}
		if !strings.Contains(verifyLine, "\"id\":2") || strings.Contains(verifyLine, "\"isError\":true") || strings.Contains(verifyLine, "\"error\":") {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			b.Fatalf("semantic_verify failed: %s", verifyLine)
		}
		_ = stdin.Close()
		if err := cmd.Wait(); err != nil {
			b.Fatalf("MCP process exit: %v; stderr: %s", err, stderr.String())
		}
	}
	if b.N > 0 {
		denom := float64(b.N)
		b.ReportMetric(float64(startupTotal)/denom/float64(time.Millisecond), "startup_ms/op")
		b.ReportMetric(float64(initializeTotal)/denom/float64(time.Millisecond), "initialize_ms/op")
		b.ReportMetric(float64(verifyTotal)/denom/float64(time.Millisecond), "verify_ms/op")
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
