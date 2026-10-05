// OpenCode benchmark isolation belongs here so every provider process runs behind the same fixture boundary.

package main

import (
	"context"
	"fmt"

	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"semedit/internal/pipeline"
	"strconv"
	"strings"
)

func (r *Runner) openCodeSandboxProfile(workDir, openCodePath, modelURL string) (string, string, error) {
	if runtime.GOOS != "darwin" {
		return "", "", fmt.Errorf("OpenCode benchmark containment requires macOS Seatbelt")
	}
	endpoint, endpointErr := url.ParseRequestURI(modelURL)
	if endpointErr != nil || endpoint.Host == "" {
		return "", "", fmt.Errorf("OpenCode model URL %q is invalid", modelURL)
	}
	sandboxPath, err := exec.LookPath("sandbox-exec")
	if err != nil {
		return "", "", fmt.Errorf("locate macOS sandbox-exec: %w", err)
	}
	fixtureRoot, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		return "", "", fmt.Errorf("resolve OpenCode fixture root: %w", err)
	}
	openCodePath, err = filepath.EvalSymlinks(openCodePath)
	if err != nil {
		return "", "", fmt.Errorf("resolve OpenCode executable: %w", err)
	}
	repositoryRoot := filepath.Dir(filepath.Dir(r.baseScratchDir))
	semeditPath := filepath.Join(repositoryRoot, "bin", "semedit-next")
	if resolvedPath, resolveErr := filepath.EvalSymlinks(semeditPath); resolveErr == nil {
		semeditPath = resolvedPath
	}
	goplsPath, _ := exec.LookPath("gopls")
	if goplsPath != "" {
		goplsPath, err = filepath.EvalSymlinks(goplsPath)
		if err != nil {
			return "", "", fmt.Errorf("resolve gopls executable: %w", err)
		}
	}
	profilePath := filepath.Join(r.baseScratchDir, ".opencode-sandboxes", filepath.Base(fixtureRoot)+".sb")
	mkdirErr := os.MkdirAll(filepath.Dir(profilePath), 0o750)
	if mkdirErr != nil {
		return "", "", fmt.Errorf("create OpenCode sandbox profile directory: %w", mkdirErr)
	}
	profile, profileErr := renderOpenCodeSeatbeltProfile(fixtureRoot, "1234", openCodePath, semeditPath, goplsPath)
	if profileErr != nil {
		return "", "", profileErr
	}
	writeErr := pipeline.WriteAtomic(profilePath, []byte(profile))
	if writeErr != nil {
		return "", "", fmt.Errorf("write OpenCode Seatbelt profile: %w", writeErr)
	}
	return sandboxPath, profilePath, nil
}

func renderOpenCodeSeatbeltProfile(fixtureRoot, endpointHost, openCodePath, semeditPath, goplsPath string) (string, error) {
	if filepath.Clean(fixtureRoot) == string(filepath.Separator) {
		return "", fmt.Errorf("refuse to expose filesystem root as an OpenCode fixture")
	}
	port, portErr := strconv.Atoi(endpointHost)
	if portErr != nil || port < 1 || port > 65535 {
		return "", fmt.Errorf("OpenCode model URL port %q is invalid", endpointHost)
	}
	var profile strings.Builder
	profile.WriteString("(version 1)\n(deny default)\n(allow process*)\n(allow sysctl-read)\n")
	for _, systemRoot := range []string{"/System", "/usr", "/bin", "/sbin", "/opt/homebrew"} {
		fmt.Fprintf(&profile, "(allow file-read* (subpath %s))\n", strconv.Quote(systemRoot))
	}
	for _, device := range []string{"/dev/null", "/dev/urandom", "/dev/random"} {
		fmt.Fprintf(&profile, "(allow file-read* (literal %s))\n", strconv.Quote(device))
	}
	fmt.Fprintf(&profile, "(allow file-read* (subpath %s))\n", strconv.Quote(fixtureRoot))
	fmt.Fprintf(&profile, "(allow file-read* (literal %s))\n", strconv.Quote(semeditPath))
	if goplsPath != "" {
		fmt.Fprintf(&profile, "(allow file-read* (literal %s))\n", strconv.Quote(goplsPath))
	}
	for _, ancestor := range []string{fixtureRoot, openCodePath, semeditPath, goplsPath} {
		if ancestor == "" {
			continue
		}
		for parent := filepath.Dir(ancestor); parent != string(filepath.Separator); parent = filepath.Dir(parent) {
			fmt.Fprintf(&profile, "(allow file-read-metadata (literal %s))\n", strconv.Quote(parent))
		}
	}
	for _, executableRoot := range []string{"/usr/bin", "/bin", "/sbin", "/opt/homebrew/bin", "/opt/homebrew/sbin"} {
		fmt.Fprintf(&profile, "(allow process-exec (subpath %s))\n", strconv.Quote(executableRoot))
	}
	fmt.Fprintf(&profile, "(allow process-exec (literal %s))\n", strconv.Quote(openCodePath))
	fmt.Fprintf(&profile, "(allow process-exec (literal %s))\n", strconv.Quote(semeditPath))
	if goplsPath != "" {
		fmt.Fprintf(&profile, "(allow process-exec (literal %s))\n", strconv.Quote(goplsPath))
	}
	fmt.Fprintf(&profile, "(allow file-write* (subpath %s))\n", strconv.Quote(fixtureRoot))
	fmt.Fprintf(&profile, "(allow file-write* (literal %s))\n", strconv.Quote("/dev/null"))
	return profile.String(), nil
}

func initializeOpenCodeFixtureGit(ctx context.Context, workDir string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("initialize OpenCode fixture Git metadata: %w", err)
	}
	gitDir := filepath.Join(workDir, ".git")
	for _, dir := range []string{filepath.Join(gitDir, "objects", "info"), filepath.Join(gitDir, "refs", "heads")} {
		if mkdirErr := os.MkdirAll(dir, 0o750); mkdirErr != nil {
			return fmt.Errorf("create isolated OpenCode Git metadata: %w", mkdirErr)
		}
	}
	config := "[core]\n\trepositoryformatversion = 0\n\tfilemode = true\n\tbare = false\n\tlogallrefupdates = true\n"
	if writeErr := pipeline.WriteAtomic(filepath.Join(gitDir, "config"), []byte(config)); writeErr != nil {
		return fmt.Errorf("write isolated OpenCode Git configuration: %w", writeErr)
	}
	if writeErr := pipeline.WriteAtomic(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n")); writeErr != nil {
		return fmt.Errorf("write isolated OpenCode Git HEAD: %w", writeErr)
	}
	return nil
}

func openCodeModelBaseURL(target Target) string {
	model := openCodeProviderModel(target.Model)
	provider, _, _ := strings.Cut(model, "/")
	if provider == "amdbeast" {
		if override := strings.TrimSpace(os.Getenv(openCodeQwenBaseURLEnv)); override != "" {
			return override
		}
		return openCodeQwenBaseURL
	}
	return "https://openrouter.ai/api/v1"
}

func sanitizedOpenCodeEnvironment(source []string) []string {
	clean := make([]string, 0, len(source))
	for _, entry := range source {
		key, _, found := strings.Cut(entry, "=")
		if !found || key == "PWD" || key == "OLDPWD" || strings.HasPrefix(key, "OPENCODE_") || strings.HasPrefix(key, "GIT_") {
			continue
		}
		clean = append(clean, entry)
	}
	return clean
}
