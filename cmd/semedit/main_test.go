// Package main_test verifies entry point functionality for semedit.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/operation"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/rogpeppe/go-internal/txtar"
)

var fakeLSPBuild sync.Once
var errFakeLSPBuild error
var fakeBashLSPPath string
var fakeMakeLSPPath string
var repositoryRoot = func() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve repository root from test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "../.."))
}()

func repoPath(path ...string) string {
	return filepath.Join(append([]string{repositoryRoot}, path...)...)
}

func prepareFakeLSPs() error {
	fakeLSPBuild.Do(func() {
		helperDir := repoPath(".scratch", "testtools")
		helperCache := repoPath(".scratch", "cache", "testtools")
		if err := os.MkdirAll(helperDir, 0o700); err != nil {
			errFakeLSPBuild = err
			return
		}
		if err := os.MkdirAll(helperCache, 0o700); err != nil {
			errFakeLSPBuild = err
			return
		}
		var err error
		helperDir, err = filepath.Abs(helperDir)
		if err != nil {
			errFakeLSPBuild = err
			return
		}
		helperCache, err = filepath.Abs(helperCache)
		if err != nil {
			errFakeLSPBuild = err
			return
		}
		fakeBashLSPPath = filepath.Join(helperDir, "bash-language-server")
		fakeMakeLSPPath = filepath.Join(helperDir, "make-ls")
		for _, target := range []string{fakeBashLSPPath, fakeMakeLSPPath} {
			command := exec.Command("go", "build", "-o", target, "./internal/testtools/lspserver")
			command.Dir = repositoryRoot
			command.Env = append(os.Environ(), "GOCACHE="+helperCache)
			output, buildErr := command.CombinedOutput()
			if buildErr != nil {
				errFakeLSPBuild = fmt.Errorf("build fake LSP %s: %w: %s", target, buildErr, output)
				return
			}
		}
	})
	return errFakeLSPBuild
}

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"semedit": func() {
			cwd, _ := os.Getwd()
			beforePath, beforeHash, hadWork := getGoWorkHash(cwd)
			exitCode := run(os.Args[1:])
			afterPath, afterHash, hasWork := getGoWorkHash(cwd)
			if hadWork && !hasWork {
				fmt.Fprintf(os.Stderr, "ADR-0005 invariant violation: semedit deleted %s\n", beforePath)
				os.Exit(1)
			}
			if !hadWork && hasWork {
				fmt.Fprintf(os.Stderr, "ADR-0005 invariant violation: semedit created %s\n", afterPath)
				os.Exit(1)
			}
			if hadWork && hasWork && (beforePath != afterPath || beforeHash != afterHash) {
				fmt.Fprintf(os.Stderr, "ADR-0005 invariant violation: semedit mutated %s\n", beforePath)
				os.Exit(1)
			}
			os.Exit(exitCode)
		},
		"rust-analyzer":                   runFakeRustAnalyzer,
		"java":                            runFakeJava,
		"metals":                          runFakeMetals,
		"ghc":                             runFakeGHC,
		"haskell-language-server-wrapper": runFakeHLS,
		"kotlin-language-server":          runFakeKotlin,
		"maven":                           runFakeMaven,
	})
}

func getGoWorkHash(startDir string) (string, string, bool) {
	manifestPath := filepath.Join(startDir, "go.work")
	data, err := os.ReadFile(manifestPath)
	if err == nil {
		sum := sha256.Sum256(data)
		return manifestPath, hex.EncodeToString(sum[:]), true
	}
	for dir := startDir; dir != "" && dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		manifestPath = filepath.Join(dir, "go.work")
		if data, err := os.ReadFile(manifestPath); err == nil {
			sum := sha256.Sum256(data)
			return manifestPath, hex.EncodeToString(sum[:]), true
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
	}
	return "", "", false
}

func TestREQ001SemanticIntentCLIContracts(t *testing.T) {
	t.Parallel()
	testscript.Run(t, testscript.Params{
		Dir: repoPath("testdata", "scripts"),
		Setup: func(env *testscript.Env) error {
			for _, key := range []string{"GOPATH", "GOCACHE", "GOROOT", "HOME", "GOBIN", "GOENV", "GOMODCACHE", "GOTMPDIR", "GOWORK", "GOPLSCACHE"} {
				if val := os.Getenv(key); val != "" {
					env.Vars = append(env.Vars, key+"="+val)
				}
			}
			env.Vars = append(env.Vars, "PATH="+os.Getenv("PATH"))
			if err := prepareFakeLSPs(); err != nil {
				return err
			}
			env.Vars = append(env.Vars, "SEMEDIT_TEST_BASH_LS="+fakeBashLSPPath, "SEMEDIT_TEST_MAKE_LS="+fakeMakeLSPPath)

			for variable, command := range map[string]string{
				"SEMEDIT_TEST_JAVA":      "java",
				"SEMEDIT_TEST_GHC":       "ghc",
				"SEMEDIT_TEST_HLS":       "haskell-language-server-wrapper",
				"SEMEDIT_TEST_METALS":    "metals",
				"SEMEDIT_TEST_KOTLIN_LS": "kotlin-language-server",
				"SEMEDIT_TEST_MAVEN":     "maven",
			} {
				path, err := exec.LookPath(command)
				if err != nil {
					return err
				}
				env.Vars = append(env.Vars, variable+"="+path)
			}

			return nil
		},
	})
}

func TestREQ002REQ006PublicRegistryAndCLIContractCoverage(t *testing.T) {
	t.Parallel()
	fixtures := loadPublicFixtures(t)
	registry := operation.DefaultRegistry()
	assertRegistryCommandCoverage(t, registry, fixtures)
	assertBatchFixtureCoverage(t, fixtures)
	assertFullCatalogCoverage(t, registry, fixtures)
}

type publicFixture struct {
	comment string
	files   map[string]string
}

func loadPublicFixtures(t *testing.T) []publicFixture {
	t.Helper()
	entries, err := os.ReadDir(repoPath("testdata", "scripts"))
	if err != nil {
		t.Fatalf("read script fixtures: %v", err)
	}
	fixtures := make([]publicFixture, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".txtar" {
			continue
		}
		data, readErr := os.ReadFile(repoPath("testdata", "scripts", entry.Name()))
		if readErr != nil {
			t.Fatalf("read fixture %s: %v", entry.Name(), readErr)
		}
		archive := txtar.Parse(data)
		files := make(map[string]string, len(archive.Files))
		for _, file := range archive.Files {
			files[filepath.ToSlash(file.Name)] = string(file.Data)
		}
		fixtures = append(fixtures, publicFixture{comment: string(archive.Comment), files: files})
	}
	return fixtures
}

func assertRegistryCommandCoverage(t *testing.T, registry *operation.Registry, fixtures []publicFixture) {
	t.Helper()
	for _, registered := range registry.All() {
		if registered.CLIName == "" {
			continue
		}
		command := regexp.MustCompile(`(?m)^\s*!?\s*(?:exec\s+)?semedit\s+` + regexp.QuoteMeta(registered.CLIName) + `(?:\s|$)`)
		for _, language := range registered.Languages {
			if !fixtureCoversLanguageCommand(fixtures, command, language) {
				t.Errorf("no executable txtar covers registry command %q for supported language %q", registered.CLIName, language)
			}
		}
	}
}

func fixtureCoversLanguageCommand(fixtures []publicFixture, command *regexp.Regexp, language backend.LanguageID) bool {
	for _, candidate := range fixtures {
		for commandLine := range strings.SplitSeq(candidate.comment, "\n") {
			if command.MatchString(commandLine) && supportsFixtureLanguage(candidate, commandLine, language) {
				return true
			}
		}
	}
	return false
}

func supportsFixtureLanguage(candidate publicFixture, commandLine string, language backend.LanguageID) bool {
	languageFlag := regexp.MustCompile(`(?:^|\s)--language(?:=|\s+)([a-z-]+)(?:\s|$)`)
	if match := languageFlag.FindStringSubmatch(commandLine); match != nil {
		return backend.LanguageID(match[1]) == language
	}
	switch language {
	case backend.LanguageAuto:
		return true
	case backend.LanguageGo:
		_, ok := candidate.files["go.mod"]
		return ok
	case backend.LanguageJava:
		_, hasPOM := candidate.files["pom.xml"]
		return hasPOM || fixtureHasExtension(candidate, ".java")
	case backend.LanguageRust:
		_, hasCargo := candidate.files["Cargo.toml"]
		return hasCargo || fixtureHasExtension(candidate, ".rs")
	case backend.LanguageScala:
		return fixtureHasExtension(candidate, ".scala")
	case backend.LanguageHaskell:
		return fixtureHasExtension(candidate, ".hs")
	case backend.LanguageKotlin:
		return fixtureHasExtension(candidate, ".kt") || fixtureHasExtension(candidate, ".kts")
	case backend.LanguageBash:
		return fixtureHasExtension(candidate, ".sh") || fixtureHasExtension(candidate, ".bash")
	case backend.LanguageMake:
		return fixtureHasMakefile(candidate)
	default:
		return false
	}
}

func fixtureHasExtension(candidate publicFixture, extension string) bool {
	for name := range candidate.files {
		if filepath.Ext(name) == extension {
			return true
		}
	}
	return false
}

func fixtureHasMakefile(candidate publicFixture) bool {
	for name := range candidate.files {
		base := strings.ToLower(filepath.Base(name))
		if base == "makefile" || base == "gnumakefile" || filepath.Ext(strings.ToLower(name)) == ".mk" {
			return true
		}
	}
	return false
}

func assertBatchFixtureCoverage(t *testing.T, fixtures []publicFixture) {
	t.Helper()
	for _, candidate := range fixtures {
		input, hasInput := candidate.files["input.json"]
		if hasInput && strings.Contains(candidate.comment, "stdin input.json") && strings.Contains(candidate.comment, "exec semedit mcp") && strings.Contains(input, `"name":"semantic_batch"`) {
			return
		}
	}
	t.Error("no executable txtar calls the MCP-only semantic_batch tool")
}

func assertFullCatalogCoverage(t *testing.T, registry *operation.Registry, fixtures []publicFixture) {
	t.Helper()
	for _, registered := range registry.All() {
		if registered.MCPName != "" && !fixtureAssertsFullCatalogTool(fixtures, registered.MCPName) {
			t.Errorf("no tools/list txtar asserts MCP tool %q", registered.MCPName)
		}
	}
	for _, tool := range []string{"semantic_batch", "semantic_reload"} {
		if !fixtureAssertsFullCatalogTool(fixtures, tool) {
			t.Errorf("no tools/list txtar asserts MCP-layer tool %q", tool)
		}
	}
}

func fixtureAssertsFullCatalogTool(fixtures []publicFixture, tool string) bool {
	assertion := regexp.MustCompile(`(?m)^stdout .*` + regexp.QuoteMeta(tool) + `.*$`)
	for _, candidate := range fixtures {
		input, hasInput := candidate.files["input.json"]
		if hasInput && strings.Contains(input, `"method":"tools/list"`) && strings.Contains(candidate.comment, "exec semedit mcp --tool-profile full --live-reload") && assertion.MatchString(candidate.comment) {
			return true
		}
	}
	return false
}

func TestTxtarsHaveAssertionsOrWantedFiles(t *testing.T) {
	t.Parallel()
	assertion := regexp.MustCompile(`(?m)^\s*(?:stdout|stderr|cmp|exists|grep)\b`)
	failureExpectation := regexp.MustCompile(`(?m)^\s*!\s*(?:exec\s+)?[^\s#]+`)
	benchmarkOracle := regexp.MustCompile(`(?m)^oracle:\s*$`)
	found := 0

	err := filepath.WalkDir(repoPath("testdata"), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".txtar" {
			return nil
		}
		root, err := os.OpenRoot(filepath.Dir(path))
		if err != nil {
			return fmt.Errorf("open txtar directory %s: %w", path, err)
		}
		data, err := root.ReadFile(filepath.Base(path))
		_ = root.Close()
		if err != nil {
			return fmt.Errorf("read txtar %s: %w", path, err)
		}
		archive := txtar.Parse(data)
		relativePath, err := filepath.Rel(repositoryRoot, path)
		if err != nil {
			return fmt.Errorf("relative txtar path %s: %w", path, err)
		}
		if !txtarHasAssertion(relativePath, archive, assertion, failureExpectation, benchmarkOracle) && !txtarHasWantedFile(archive.Files) {
			t.Errorf("txtar %s has no test assertion or wanted file", relativePath)
		}
		found++
		return nil
	})
	if err != nil {
		t.Fatalf("walk txtar fixtures: %v", err)
	}
	if found == 0 {
		t.Fatal("no txtar fixtures found under testdata")
	}
}

func txtarHasWantedFile(files []txtar.File) bool {
	for _, file := range files {
		for component := range strings.SplitSeq(filepath.ToSlash(file.Name), "/") {
			if component == "want" || strings.HasPrefix(component, "want.") || strings.HasPrefix(component, "want_") {
				return true
			}
		}
	}
	return false
}

func txtarHasAssertion(path string, archive *txtar.Archive, assertion, failureExpectation, benchmarkOracle *regexp.Regexp) bool {
	comment := archive.Comment
	if assertion.Match(comment) || failureExpectation.Match(comment) {
		return true
	}
	return strings.HasPrefix(filepath.ToSlash(path), "testdata/bench/") && benchmarkOracle.Match(comment)
}

func TestMCPLiveReloadFlag(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd(".")
	mcpCmd, _, err := cmd.Find([]string{"mcp"})
	if err != nil {
		t.Fatalf("failed to find mcp command: %v", err)
	}

	flag := mcpCmd.Flags().Lookup("live-reload")
	if flag == nil {
		t.Fatalf("expected --live-reload flag on mcp command")
	}
	if flag.DefValue != "false" {
		t.Errorf("expected --live-reload default value 'false', got %q", flag.DefValue)
	}
}

func TestRootCommandWithoutArgumentsRendersHelp(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd(t.TempDir())
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("root command without arguments failed: %v", err)
	}
	if !strings.Contains(output.String(), "Semantic Editor for LLM Intent-Driven Code Refactoring") {
		t.Errorf("root help = %q, want command description", output.String())
	}
}

func TestHarnessInstallationPublicCLI(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	binary := filepath.Join(workspace, "semedit")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"), 0o755); err != nil { // #nosec G306 -- executable fixture.
		t.Fatal(err)
	}
	testInstallOutputModes(t, workspace, binary)
	testIntegrationStatusOutputModes(t, workspace)
	testUninstallOutputModes(t, workspace)
}

func runIntegrationCLI(t *testing.T, workspace string, args ...string) (string, string) {
	t.Helper()
	cmd := newRootCmd(workspace)
	cmd.SetArgs(args)
	var output, stderr bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&stderr)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("CLI command %q failed: %v", args, err)
	}
	return output.String(), stderr.String()
}

func testInstallOutputModes(t *testing.T, workspace, binary string) {
	t.Helper()
	args := []string{"install", "copilot", "--scope", "workspace", "--binary", binary, "--profile", "mutations-only"}
	output, stderr := runIntegrationCLI(t, workspace, args...)
	if !strings.Contains(output, "Installed semedit for copilot (workspace)") || !strings.Contains(output, filepath.Join(workspace, ".mcp.json")) || !strings.Contains(output, "Restart the copilot coding harness") || stderr != "" {
		t.Fatalf("install output/stderr = %s / %s", output, stderr)
	}
	if _, err := os.Stat(filepath.Join(workspace, ".mcp.json")); err != nil {
		t.Fatalf("installed config was not saved: %v", err)
	}
	output, stderr = runIntegrationCLI(t, workspace, args...)
	if !strings.Contains(output, "already installed for copilot (workspace)") || !strings.Contains(output, "Restart the copilot coding harness") || stderr != "" {
		t.Fatalf("repeat install output/stderr = %s / %s", output, stderr)
	}
	jsonOutput, jsonStderr := runIntegrationCLI(t, workspace, append(args, "--json")...)
	assertIntegrationJSONStatus(t, jsonOutput, "already-installed")
	if jsonStderr != "" {
		t.Fatalf("JSON install wrote stderr: %s", jsonStderr)
	}
	dryWorkspace := t.TempDir()
	dryArgs := []string{"install", "copilot", "--scope", "workspace", "--binary", binary, "--dry-run"}
	dryOutput, dryStderr := runIntegrationCLI(t, dryWorkspace, dryArgs...)
	if !strings.Contains(dryOutput, "Preview only: semedit would be installed") || !strings.Contains(dryOutput, "Nothing was written") || !strings.Contains(dryOutput, "rerun without --dry-run") || dryStderr != "" {
		t.Fatalf("dry-run output/stderr = %s / %s", dryOutput, dryStderr)
	}
	if _, err := os.Stat(filepath.Join(dryWorkspace, ".mcp.json")); !os.IsNotExist(err) {
		t.Fatalf("dry-run unexpectedly wrote config: stat error = %v", err)
	}
}

func testIntegrationStatusOutputModes(t *testing.T, workspace string) {
	t.Helper()
	output, _ := runIntegrationCLI(t, workspace, "integration", "status", "copilot", "--scope", "workspace")
	if !strings.Contains(output, "Installed semedit for copilot (workspace)") || !strings.Contains(output, filepath.Join(workspace, ".mcp.json")) {
		t.Fatalf("status output = %s", output)
	}
	jsonOutput, stderr := runIntegrationCLI(t, workspace, "integration", "status", "copilot", "--json")
	var result struct {
		Target   string `json:"target"`
		Statuses []struct {
			Scope  string `json:"scope"`
			Status string `json:"status"`
		} `json:"statuses"`
	}
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil || result.Target != "copilot" || len(result.Statuses) != 2 || result.Statuses[0].Status != "installed" || result.Statuses[0].Scope != "workspace" || result.Statuses[1].Scope != "user" || stderr != "" {
		t.Fatalf("status JSON/stderr = %s / %s, result = %#v, error = %v", jsonOutput, stderr, result, err)
	}
}

func testUninstallOutputModes(t *testing.T, workspace string) {
	t.Helper()
	jsonOutput, stderr := runIntegrationCLI(t, workspace, "uninstall", "copilot", "--scope", "workspace", "--json")
	assertIntegrationJSONStatus(t, jsonOutput, "uninstalled")
	if stderr != "" {
		t.Fatalf("JSON uninstall wrote stderr: %s", stderr)
	}
	binary := filepath.Join(workspace, "semedit")
	output, _ := runIntegrationCLI(t, workspace, "install", "copilot", "--scope", "workspace", "--binary", binary)
	if !strings.Contains(output, "Installed semedit for copilot (workspace)") {
		t.Fatalf("reinstall output = %s", output)
	}
	output, _ = runIntegrationCLI(t, workspace, "uninstall", "copilot", "--scope", "workspace")
	if !strings.Contains(output, "Uninstalled semedit for copilot (workspace)") {
		t.Fatalf("uninstall output = %s", output)
	}
}

func assertIntegrationJSONStatus(t *testing.T, output, want string) {
	t.Helper()
	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil || result.Status != want {
		t.Fatalf("integration JSON = %s, status = %q, error = %v", output, result.Status, err)
	}
}
