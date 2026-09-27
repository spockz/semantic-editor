// This file resolves the selected project root into source and verification-signal plans using strict project configuration.

// Package projectverify discovers configured verification signals and runs ordered project hooks.
package projectverify

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"semedit/internal/sourcefiles"

	"gopkg.in/yaml.v3"
)

// Discover returns source files and configured verification hooks for root.
func Discover(root string) (Plan, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return Plan{}, fmt.Errorf("resolve project root: %w", err)
	}
	absolute, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		return Plan{}, fmt.Errorf("resolve project root symlinks: %w", err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return Plan{}, fmt.Errorf("stat project root %q: %w", absolute, err)
	}
	if !info.IsDir() {
		return Plan{}, fmt.Errorf("project root %q is not a directory", absolute)
	}
	sources, err := discoverSources(absolute)
	if err != nil {
		return Plan{}, fmt.Errorf("discover project sources: %w", err)
	}
	checks, err := discoverSignalChecks(absolute, sources)
	if err != nil {
		return Plan{}, fmt.Errorf("discover verification signals: %w", err)
	}
	plan := Plan{Root: absolute, Sources: sources, Checks: checks}
	configPath := filepath.Join(absolute, ".semedit.yaml")
	if _, err := os.Lstat(configPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return plan, nil
		}
		return Plan{}, fmt.Errorf("inspect project verification config: %w", err)
	}
	configInfo, err := os.Lstat(configPath)
	if err != nil {
		return Plan{}, fmt.Errorf("inspect project verification config: %w", err)
	}
	if !configInfo.Mode().IsRegular() {
		return Plan{}, fmt.Errorf("project verification config %q must be a regular file", configPath)
	}
	cfg, err := readConfig(configPath)
	if err != nil {
		return Plan{}, err
	}
	plan.ConfigPath = ".semedit.yaml"
	plan.Normalize, err = convertHooks(absolute, cfg.Verify.Normalize, PhaseNormalize, plan.ConfigPath)
	if err != nil {
		return Plan{}, err
	}
	explicitChecks, err := convertHooks(absolute, cfg.Verify.Check, PhaseCheck, plan.ConfigPath)
	if err != nil {
		return Plan{}, err
	}
	plan.Checks, err = mergeChecks(explicitChecks, checks)
	if err != nil {
		return Plan{}, err
	}
	return plan, nil
}

type lspConfig struct {
	Language   string `yaml:"language"`
	ActionKind string `yaml:"action_kind"`
	Command    string `yaml:"command"`
	Arguments  any    `yaml:"arguments"`
}

type hookConfig struct {
	ID        string     `yaml:"id"`
	Languages []string   `yaml:"languages"`
	Disabled  bool       `yaml:"disabled"`
	Cwd       string     `yaml:"cwd"`
	Timeout   string     `yaml:"timeout"`
	Exec      []string   `yaml:"exec"`
	Shell     *ShellSpec `yaml:"shell"`
	LSP       *lspConfig `yaml:"lsp"`
}

type verifyConfig struct {
	Normalize []hookConfig `yaml:"normalize"`
	Check     []hookConfig `yaml:"check"`
}

type fileConfig struct {
	Version int          `yaml:"version"`
	Verify  verifyConfig `yaml:"verify"`
}

func readConfig(path string) (fileConfig, error) {
	file, err := os.Open(path) // #nosec G304 -- path comes from the validated project root config location
	if err != nil {
		return fileConfig{}, fmt.Errorf("open project verification config %q: %w", path, err)
	}
	defer func() { _ = file.Close() }()
	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	var cfg fileConfig
	if err := decoder.Decode(&cfg); err != nil {
		return fileConfig{}, fmt.Errorf("decode project verification config %q: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fileConfig{}, fmt.Errorf("decode project verification config %q: multiple YAML documents are not supported", path)
		}
		return fileConfig{}, fmt.Errorf("decode project verification config %q trailing document: %w", path, err)
	}
	if cfg.Version != 1 {
		return fileConfig{}, fmt.Errorf("project verification config %q has unsupported version %d; supported version is 1", path, cfg.Version)
	}
	return cfg, nil
}

func discoverSources(root string) ([]Source, error) {
	var sources []Source
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("make source path relative: %w", err)
		}
		if entry.IsDir() {
			if path != root && skippedDiscoveryDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return nil
		}
		language := sourceLanguage(entry.Name())
		if language != "" {
			sources = append(sources, Source{Path: filepath.ToSlash(rel), Language: language})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(sources, func(a, b Source) int { return strings.Compare(a.Path, b.Path) })
	return sources, nil
}

func skippedDiscoveryDir(name string) bool {
	return sourcefiles.IsIgnoredDirectory(name)
}

func sourceLanguage(name string) string {
	return sourcefiles.LanguageForPath(name)
}

func discoverSignalChecks(root string, sources []Source) ([]Hook, error) {
	type lintScope struct {
		configDir  string
		config     string
		moduleRoot string
		files      []string
	}
	scopes := make(map[string]*lintScope)
	for _, source := range sources {
		if source.Language != "go" {
			continue
		}
		sourceDir := filepath.Dir(filepath.Join(root, filepath.FromSlash(source.Path)))
		config, configDir, err := nearestGolangciConfig(root, sourceDir)
		if err != nil {
			return nil, err
		}
		if config == "" {
			continue
		}
		moduleRoot, err := nearestGoModule(root, sourceDir)
		if err != nil {
			return nil, err
		}
		key := configDir + "\x00" + moduleRoot
		scope := scopes[key]
		if scope == nil {
			scope = &lintScope{configDir: configDir, config: config, moduleRoot: moduleRoot}
			scopes[key] = scope
		}
		scope.files = append(scope.files, source.Path)
	}
	keys := make([]string, 0, len(scopes))
	for key := range scopes {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var hooks []Hook
	for _, key := range keys {
		scope := scopes[key]
		relConfig, err := filepath.Rel(root, scope.config)
		if err != nil {
			return nil, fmt.Errorf("resolve linter config: %w", err)
		}
		relModule, err := filepath.Rel(root, scope.moduleRoot)
		if err != nil {
			return nil, fmt.Errorf("resolve Go module scope: %w", err)
		}
		relConfigDir, err := filepath.Rel(root, scope.configDir)
		if err != nil {
			return nil, fmt.Errorf("resolve linter config scope: %w", err)
		}
		id := "golangci-lint"
		if relModule != "." || relConfigDir != "." {
			id += "@" + filepath.ToSlash(relModule) + "#" + filepath.ToSlash(relConfigDir)
		}
		slices.Sort(scope.files)
		hooks = append(hooks, Hook{
			ID:         id,
			Languages:  []string{"go"},
			Cwd:        filepath.ToSlash(relModule),
			Exec:       []string{"golangci-lint"},
			Signal:     "golangci-config",
			ConfigPath: filepath.ToSlash(relConfig),
			ScopePaths: scope.files,
		})
	}
	return hooks, nil
}

func nearestGolangciConfig(root, start string) (string, string, error) {
	dir := start
	for {
		for _, name := range []string{".golangci.yml", ".golangci.yaml", ".golangci.toml", ".golangci.json"} {
			candidate := filepath.Join(dir, name)
			info, err := os.Lstat(candidate)
			if err == nil {
				if !info.Mode().IsRegular() {
					return "", "", fmt.Errorf("golangci config %q is not a regular file", candidate)
				}
				return candidate, dir, nil
			}
			if !errors.Is(err, fs.ErrNotExist) {
				return "", "", fmt.Errorf("inspect golangci config %q: %w", candidate, err)
			}
		}
		if dir == root {
			return "", "", nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", nil
		}
		rel, err := filepath.Rel(root, parent)
		if err != nil {
			return "", "", fmt.Errorf("resolve parent while discovering golangci config: %w", err)
		} else if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", "", nil
		}
		dir = parent
	}
}

func convertHooks(root string, configured []hookConfig, phase Phase, configPath string) ([]Hook, error) {
	hooks := make([]Hook, 0, len(configured))
	seen := make(map[string]struct{}, len(configured))
	for i, item := range configured {
		where := fmt.Sprintf("verify.%s[%d]", phase, i)
		if strings.TrimSpace(item.ID) == "" {
			return nil, fmt.Errorf("%s: id is required", where)
		}
		if _, exists := seen[item.ID]; exists {
			return nil, fmt.Errorf("%s: duplicate hook id %q", where, item.ID)
		}
		seen[item.ID] = struct{}{}
		hook := Hook{ID: item.ID, Languages: canonicalLanguages(item.Languages), Disabled: item.Disabled, Signal: "explicit-config", ConfigPath: configPath}
		if !hook.Disabled {
			for _, language := range hook.Languages {
				if !knownLanguage(language) {
					return nil, fmt.Errorf("%s: unknown language %q", where, language)
				}
			}
			if len(hook.Languages) == 0 {
				return nil, fmt.Errorf("%s: at least one language filter is required", where)
			}
		}
		if item.Cwd != "" {
			cwd, err := safeRelativePath(item.Cwd)
			if err != nil {
				return nil, fmt.Errorf("%s cwd: %w", where, err)
			}
			info, err := os.Stat(filepath.Join(root, cwd))
			if err != nil {
				return nil, fmt.Errorf("%s cwd %q: %w", where, item.Cwd, err)
			}
			if !info.IsDir() {
				return nil, fmt.Errorf("%s cwd %q is not a directory", where, item.Cwd)
			}
			hook.Cwd = filepath.ToSlash(cwd)
		}
		if item.Timeout != "" {
			timeout, err := time.ParseDuration(item.Timeout)
			if err != nil || timeout <= 0 || timeout > time.Hour {
				if err == nil {
					err = fmt.Errorf("timeout must be greater than zero and at most one hour")
				}
				return nil, fmt.Errorf("%s timeout: %w", where, err)
			}
			hook.Timeout = timeout
		}
		if hook.Disabled {
			if len(item.Exec) > 0 || item.Shell != nil || item.LSP != nil {
				return nil, fmt.Errorf("%s: disabled hook cannot also define an execution form", where)
			}
			hooks = append(hooks, hook)
			continue
		}
		forms := 0
		if len(item.Exec) > 0 {
			forms++
		}
		if item.Shell != nil {
			forms++
		}
		if item.LSP != nil {
			forms++
		}
		if forms != 1 {
			return nil, fmt.Errorf("%s: specify exactly one of exec, shell, or lsp", where)
		}
		if len(item.Exec) > 0 {
			if strings.TrimSpace(item.Exec[0]) == "" {
				return nil, fmt.Errorf("%s exec[0] must name an executable", where)
			}
			hook.Exec = append([]string(nil), item.Exec...)
		}
		if item.Shell != nil {
			if strings.TrimSpace(item.Shell.Executable) == "" || strings.TrimSpace(item.Shell.Script) == "" {
				return nil, fmt.Errorf("%s shell requires executable and script", where)
			}
			shell := *item.Shell
			shell.Args = append([]string(nil), item.Shell.Args...)
			hook.Shell = &shell
		}
		if item.LSP != nil {
			lsp := &LSPSpec{
				Language:   strings.ToLower(strings.TrimSpace(item.LSP.Language)),
				ActionKind: strings.TrimSpace(item.LSP.ActionKind),
				Command:    strings.TrimSpace(item.LSP.Command),
				Arguments:  item.LSP.Arguments,
			}
			if lsp.Language == "" || (lsp.ActionKind == "") == (lsp.Command == "") {
				return nil, fmt.Errorf("%s lsp requires language and exactly one of action_kind or command", where)
			}
			if !slices.Contains(hook.Languages, lsp.Language) {
				return nil, fmt.Errorf("%s lsp language %q must be included in its language filters", where, lsp.Language)
			}
			if phase == PhaseCheck && lsp.ActionKind != "" {
				return nil, fmt.Errorf("%s lsp code actions are only allowed during normalization", where)
			}
			hook.LSP = lsp
		}
		hooks = append(hooks, hook)
	}
	return hooks, nil
}

func mergeChecks(explicit, detected []Hook) ([]Hook, error) {
	if len(explicit) == 0 {
		return detected, nil
	}
	var merged []Hook
	overridden := make(map[string]bool)
	seen := make(map[string]bool)
	for _, hook := range explicit {
		overridden[hook.ID] = true
		if !hook.Disabled {
			merged = append(merged, hook)
			seen[hook.ID] = true
		}
	}
	for _, hook := range detected {
		if overridden[hook.ID] || (hook.ID != "golangci-lint" && overridden["golangci-lint"]) {
			continue
		}
		if seen[hook.ID] {
			return nil, fmt.Errorf("duplicate discovered verification hook id %q", hook.ID)
		}
		seen[hook.ID] = true
		merged = append(merged, hook)
	}
	return merged, nil
}

func canonicalLanguages(languages []string) []string {
	seen := make(map[string]struct{}, len(languages))
	var result []string
	for _, language := range languages {
		language = strings.ToLower(strings.TrimSpace(language))
		if language == "" {
			continue
		}
		if _, exists := seen[language]; exists {
			continue
		}
		seen[language] = struct{}{}
		result = append(result, language)
	}
	return result
}

func safeRelativePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}
	clean := filepath.Clean(path)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes project root")
	}
	return clean, nil
}

func knownLanguage(language string) bool {
	switch language {
	case "go", "rust", "java", "scala", "haskell", "kotlin", "bash", "make":
		return true
	default:
		return false
	}
}

func nearestGoModule(root, start string) (string, error) {
	dir := start
	for {
		candidate := filepath.Join(dir, "go.mod")
		info, err := os.Lstat(candidate)
		if err == nil {
			if !info.Mode().IsRegular() {
				return "", fmt.Errorf("go module manifest %q is not a regular file", candidate)
			}
			return dir, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("inspect Go module manifest %q: %w", candidate, err)
		}
		if dir == root {
			return root, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return root, nil
		}
		rel, err := filepath.Rel(root, parent)
		if err != nil {
			return "", fmt.Errorf("resolve parent while discovering Go module: %w", err)
		} else if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return root, nil
		}
		dir = parent
	}
}
