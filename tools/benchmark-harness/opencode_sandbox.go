// OpenCode benchmark isolation belongs here so every provider process runs behind the same fixture boundary.

package main

import (
	"context"
	"fmt"
	"io"

	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

func (r *Runner) openCodeSandboxProfile(workDir, openCodePath, modelURL string) (string, string, error) {
	if r.openCodeSandboxProfileFactory != nil {
		return r.openCodeSandboxProfileFactory(workDir, openCodePath, modelURL)
	}
	if runtime.GOOS != "darwin" {
		return "", "", fmt.Errorf("OpenCode benchmark containment requires macOS Seatbelt")
	}
	relayPort, endpointErr := openCodeSandboxRelayPort(modelURL)
	if endpointErr != nil {
		return "", "", endpointErr
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
	goRoot, goErr := resolveOpenCodeGoRoot(os.Environ())
	if goErr != nil {
		return "", "", goErr
	}
	goplsPath, goplsRuntimePaths, goplsErr := resolveOpenCodeGopls()
	if goplsErr != nil {
		return "", "", goplsErr
	}
	goPath, goPathErr := exec.LookPath("go")
	if goPathErr != nil {
		return "", "", fmt.Errorf("locate Go executable for OpenCode sandbox: %w", goPathErr)
	}
	goPath, goPathErr = filepath.EvalSymlinks(goPath)
	if goPathErr != nil {
		return "", "", fmt.Errorf("resolve Go executable for OpenCode sandbox: %w", goPathErr)
	}
	goplsRuntimePaths = append(goplsRuntimePaths, goPath)
	repositoryRoot := filepath.Dir(filepath.Dir(r.baseScratchDir))
	semeditPath := filepath.Join(repositoryRoot, "bin", "semedit-next")
	if resolvedPath, resolveErr := filepath.EvalSymlinks(semeditPath); resolveErr == nil {
		semeditPath = resolvedPath
	}
	profilePath := filepath.Join(fixtureRoot, ".scratch", "opencode", "seatbelt.sb")
	mkdirErr := prepareRuntimeLayout(fixtureRoot, []string{filepath.Join(".scratch", "opencode")}, nil)
	if mkdirErr != nil {
		return "", "", fmt.Errorf("create OpenCode sandbox profile directory: %w", mkdirErr)
	}
	profile, profileErr := renderOpenCodeSeatbeltProfileWithRuntime(fixtureRoot, relayPort, openCodePath, semeditPath, goplsPath, goRoot, goplsRuntimePaths)
	if profileErr != nil {
		return "", "", profileErr
	}
	if r.moduleCache != "" {
		profile += fmt.Sprintf("(allow file-read* (subpath %s))\n", strconv.Quote(r.moduleCache))
	}
	writeErr := prepareRuntimeLayout(fixtureRoot, nil, map[string][]byte{filepath.Join(".scratch", "opencode", "seatbelt.sb"): []byte(profile)})
	if writeErr != nil {
		return "", "", fmt.Errorf("write OpenCode Seatbelt profile: %w", writeErr)
	}
	return sandboxPath, profilePath, nil
}

func (r *Runner) openCodeModelEndpointURL(target Target) (string, error) {
	provider, _, _ := strings.Cut(openCodeProviderModel(target.Model), "/")
	if provider == "amdbeast" {
		if relayURL := strings.TrimSpace(r.openCodeRelayURL); relayURL != "" {
			return relayURL, nil
		}
		return "", fmt.Errorf("AMD Beast requires a run-owned OpenCode relay; configure the Runner with WithOpenCodeRelayURL")
	}
	return openCodeModelBaseURL(target), nil
}

func openCodeSandboxRelayPort(modelURL string) (string, error) {
	endpoint, err := url.ParseRequestURI(modelURL)
	if err != nil || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" {
		return "", fmt.Errorf("OpenCode model URL %q is invalid", modelURL)
	}
	if endpoint.Scheme == "https" && endpoint.Hostname() == "openrouter.ai" {
		return "", nil
	}
	_, _, relayErr := parseOpenCodeRelayEndpoint(modelURL)
	if relayErr != nil || !net.ParseIP(endpoint.Hostname()).IsLoopback() {
		return "", fmt.Errorf("OpenCode relay URL %q must use a loopback IP address and explicit port", modelURL)
	}
	return endpoint.Port(), nil
}

func renderOpenCodeSeatbeltProfile(fixtureRoot, relayPort, openCodePath, semeditPath, goplsPath string) (string, error) {
	if filepath.Clean(fixtureRoot) == string(filepath.Separator) {
		return "", fmt.Errorf("refuse to expose filesystem root as an OpenCode fixture")
	}
	var profile strings.Builder
	profile.WriteString("(version 1)\n(deny default)\n(allow process*)\n(allow sysctl-read)\n")
	if relayPort != "" {
		port, portErr := strconv.Atoi(relayPort)
		if portErr != nil || port < 1 || port > 65535 {
			return "", fmt.Errorf("OpenCode relay port %q is invalid", relayPort)
		}
		fmt.Fprintf(&profile, "(allow network-outbound (remote tcp %s))\n", strconv.Quote(fmt.Sprintf("localhost:%d", port)))
	}
	// macOS 26 dyld reads the root vnode while locating its shared cache.
	profile.WriteString("(allow file-read* (literal \"/\"))\n")
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
	return prepareRuntimeLayout(workDir, []string{filepath.Join(".git", "objects", "info"), filepath.Join(".git", "refs", "heads")}, map[string][]byte{
		filepath.Join(".git", "config"): []byte("[core]\n\trepositoryformatversion = 0\n\tfilemode = true\n\tbare = false\n\tlogallrefupdates = true\n"),
		filepath.Join(".git", "HEAD"):   []byte("ref: refs/heads/main\n"),
	})
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

func (r *Runner) openCodeSessionSandboxProfile(workDir, openCodePath, modelURL, port string) (string, string, error) {
	sandboxPath, profilePath, err := r.openCodeSandboxProfile(workDir, openCodePath, modelURL)
	if err != nil {
		return "", "", err
	}
	// #nosec G304 -- profilePath is created under the runner's isolated scratch root.
	profile, err := os.ReadFile(profilePath)
	if err != nil {
		return "", "", fmt.Errorf("read OpenCode sandbox profile: %w", err)
	}
	profile, err = appendOpenCodeSessionLoopbackRule(profile, port)
	if err != nil {
		return "", "", err
	}
	if err := prepareRuntimeLayout(workDir, nil, map[string][]byte{filepath.Join(".scratch", "opencode", "seatbelt.sb"): profile}); err != nil {
		return "", "", fmt.Errorf("write OpenCode loopback server permission: %w", err)
	}
	return sandboxPath, profilePath, nil
}

func appendOpenCodeSessionLoopbackRule(profile []byte, port string) ([]byte, error) {
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return nil, fmt.Errorf("OpenCode server port %q is invalid", port)
	}
	rule := fmt.Sprintf("\n(allow network-bind (local tcp \"localhost:%d\"))\n(allow network-inbound (local tcp \"localhost:%d\"))\n(allow network-outbound (remote tcp \"localhost:%d\"))\n", portNumber, portNumber, portNumber)
	return append(profile, []byte(rule)...), nil
}

func resolveOpenCodeGoRoot(environment []string) (string, error) {
	if goRoot := environmentLookup(environment, "GOROOT"); goRoot != "" {
		resolved, err := filepath.EvalSymlinks(goRoot)
		if err != nil {
			return "", fmt.Errorf("resolve OpenCode Go runtime directory %q: %w", goRoot, err)
		}
		info, err := os.Stat(resolved)
		if err != nil {
			return "", fmt.Errorf("inspect OpenCode Go runtime directory %q: %w", resolved, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("OpenCode Go runtime path %q is not a directory", resolved)
		}
		return resolved, nil
	}
	goPath, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("locate Go for OpenCode sandbox: %w", err)
	}
	goPath, err = filepath.EvalSymlinks(goPath)
	if err != nil {
		return "", fmt.Errorf("resolve Go executable for OpenCode sandbox: %w", err)
	}
	// #nosec G204 -- The selected Go executable is resolved from PATH with fixed arguments.
	cmd := exec.CommandContext(context.Background(), goPath, "env", "GOROOT")
	cmd.Env = environment
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("resolve Go runtime for OpenCode sandbox: %w: %s", err, strings.TrimSpace(string(output)))
	}
	goRoot := strings.TrimSpace(string(output))
	if goRoot == "" {
		return "", fmt.Errorf("go returned an empty GOROOT for OpenCode sandbox")
	}
	goRoot, err = filepath.EvalSymlinks(goRoot)
	if err != nil {
		return "", fmt.Errorf("resolve OpenCode Go runtime directory %q: %w", goRoot, err)
	}
	info, err := os.Stat(goRoot)
	if err != nil {
		return "", fmt.Errorf("inspect OpenCode Go runtime directory %q: %w", goRoot, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("OpenCode Go runtime path %q is not a directory", goRoot)
	}
	return goRoot, nil
}

func renderOpenCodeSeatbeltProfileWithRuntime(fixtureRoot, relayPort, openCodePath, semeditPath, goplsPath, goRoot string, goplsRuntimePaths []string) (string, error) {
	profile, err := renderOpenCodeSeatbeltProfile(fixtureRoot, relayPort, openCodePath, semeditPath, goplsPath)
	if err != nil {
		return "", err
	}
	var extra strings.Builder
	fmt.Fprintf(&extra, "(allow file-read* (subpath %s))\n", strconv.Quote(goRoot))
	fmt.Fprintf(&extra, "(allow process-exec (subpath %s))\n", strconv.Quote(goRoot))
	for _, path := range goplsRuntimePaths {
		if path == "" {
			continue
		}
		fmt.Fprintf(&extra, "(allow file-read* (literal %s))\n", strconv.Quote(path))
		fmt.Fprintf(&extra, "(allow process-exec (literal %s))\n", strconv.Quote(path))
		for parent := filepath.Dir(path); parent != string(filepath.Separator); parent = filepath.Dir(parent) {
			fmt.Fprintf(&extra, "(allow file-read-metadata (literal %s))\n", strconv.Quote(parent))
		}
	}
	for parent := filepath.Dir(goRoot); parent != string(filepath.Separator); parent = filepath.Dir(parent) {
		fmt.Fprintf(&extra, "(allow file-read-metadata (literal %s))\n", strconv.Quote(parent))
	}
	return profile + extra.String(), nil
}

func resolveOpenCodeGopls() (string, []string, error) {
	goplsPath, err := exec.LookPath("gopls")
	if err != nil {
		return "", nil, fmt.Errorf("locate gopls for OpenCode sandbox: %w", err)
	}
	goplsPath, err = filepath.EvalSymlinks(goplsPath)
	if err != nil {
		return "", nil, fmt.Errorf("resolve gopls executable for OpenCode sandbox: %w", err)
	}
	file, err := os.Open(goplsPath)
	if err != nil {
		return "", nil, fmt.Errorf("inspect gopls executable for OpenCode sandbox: %w", err)
	}
	prefix := make([]byte, 2)
	n, readErr := file.Read(prefix)
	closeErr := file.Close()
	if readErr != nil && readErr != io.EOF {
		return "", nil, fmt.Errorf("read gopls executable header: %w", readErr)
	}
	if closeErr != nil {
		return "", nil, fmt.Errorf("close gopls executable: %w", closeErr)
	}
	if n != len(prefix) || string(prefix) != "#!" {
		return goplsPath, []string{goplsPath}, nil
	}
	wrapper, err := os.ReadFile(goplsPath)
	if err != nil {
		return "", nil, fmt.Errorf("read gopls wrapper: %w", err)
	}
	header, _, _ := strings.Cut(string(wrapper), "\n")
	if header != "#!/bin/sh" || !slices.Contains(strings.Split(string(wrapper), "\n"), `exec "$semedit_bin_dir/gopls-go127" "$@"`) {
		return "", nil, fmt.Errorf("unsupported gopls wrapper %q: sandbox supports the repository's gopls-go127 wrapper or a direct binary", goplsPath)
	}
	interpreter, err := filepath.EvalSymlinks("/bin/sh")
	if err != nil {
		return "", nil, fmt.Errorf("resolve gopls wrapper interpreter: %w", err)
	}
	binaryPath, err := filepath.EvalSymlinks(filepath.Join(filepath.Dir(goplsPath), "gopls-go127"))
	if err != nil {
		return "", nil, fmt.Errorf("resolve gopls wrapper executable: %w", err)
	}
	return goplsPath, []string{goplsPath, interpreter, binaryPath}, nil
}
