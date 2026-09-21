// Package integration installs semedit's preinstalled MCP server into native harness configuration.
package integration

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Target identifies a supported agent harness.
type Target string

const (
	// TargetCopilot installs into GitHub Copilot.
	TargetCopilot Target = "copilot"
	// TargetCodex installs into Codex.
	TargetCodex Target = "codex"
)

// Scope identifies whether a registration belongs to one workspace or a user.
type Scope string

const (
	// ScopeWorkspace writes project-owned configuration.
	ScopeWorkspace Scope = "workspace"
	// ScopeUser writes user-owned configuration.
	ScopeUser Scope = "user"
)

const serverName = "semedit"

// Request is the common installation contract shared by target adapters.
type Request struct {
	Target    Target
	Scope     Scope
	Workspace string
	Binary    string
	Profile   string
	DryRun    bool
	Replace   bool
}

// Result is the redacted, deterministic result returned by an administrative command.
type Result struct {
	Target       Target   `json:"target"`
	Scope        Scope    `json:"scope"`
	Path         string   `json:"path"`
	Status       string   `json:"status"`
	Changed      bool     `json:"changed"`
	Owned        bool     `json:"owned"`
	Binary       string   `json:"binary,omitempty"`
	BinaryExists bool     `json:"binary_exists"`
	Profile      string   `json:"profile,omitempty"`
	Command      []string `json:"command,omitempty"`
	Skill        string   `json:"skill"`
	Detail       string   `json:"detail,omitempty"`
}

// StatusRequest asks for one target registration's status.
type StatusRequest struct {
	Target    Target
	Scope     Scope
	Workspace string
}

// ErrConflict indicates that an existing registration differs from the requested one.
var ErrConflict = errors.New("existing semedit registration conflicts")

// ErrNotOwned indicates that uninstall cannot safely remove a user-authored registration.
var ErrNotOwned = errors.New("semedit registration is not owned by the installer")

// ValidateRequest checks the invariant shared by both target adapters.
func ValidateRequest(req Request) error {
	if req.Target != TargetCopilot && req.Target != TargetCodex {
		return fmt.Errorf("unsupported integration target %q", req.Target)
	}
	if req.Scope != ScopeUser && req.Scope != ScopeWorkspace {
		return fmt.Errorf("scope must be user or workspace")
	}
	if req.Scope == ScopeWorkspace && strings.TrimSpace(req.Workspace) == "" {
		return fmt.Errorf("workspace scope requires a workspace path")
	}
	if req.Profile == "" {
		req.Profile = "full"
	}
	if req.Profile != "full" && req.Profile != "mutations-only" {
		return fmt.Errorf("profile must be full or mutations-only")
	}
	if req.Binary == "" {
		return fmt.Errorf("an absolute preinstalled semedit binary is required")
	}
	if req.Binary != "" && !filepath.IsAbs(req.Binary) {
		return fmt.Errorf("binary must be an absolute path: %q", req.Binary)
	}
	return nil
}

// Install writes one native registration. It never inspects or changes another target.
func Install(req Request) (Result, error) {
	if req.Profile == "" {
		req.Profile = "full"
	}
	if err := ValidateRequest(req); err != nil {
		return Result{}, err
	}
	path, err := configPath(req.Target, req.Scope, req.Workspace)
	if err != nil {
		return Result{}, err
	}
	if req.Binary != "" {
		if err := validateBinary(req.Binary); err != nil {
			return Result{}, err
		}
	}
	result := baseResult(req, path)
	result.Binary = req.Binary
	result.Profile = req.Profile
	result.Command = []string{req.Binary, "mcp", "--profile", req.Profile}
	data, mode, exists, err := readConfig(path)
	if err != nil {
		return Result{}, err
	}
	owned, err := ownedRegistration(path, req.Target, req.Scope, data)
	if err != nil {
		return Result{}, err
	}
	var updated []byte
	var matching bool
	switch req.Target {
	case TargetCopilot:
		updated, matching, err = mergeCopilot(data, req, exists)
	case TargetCodex:
		updated, matching, err = mergeCodex(data, req, exists)
	}
	if err != nil {
		return Result{}, err
	}
	result.Owned = owned
	if matching {
		result.Status = "already-installed"
		result.Changed = false
		return result, nil
	}
	if req.DryRun {
		result.Status = "dry-run"
		result.Changed = true
		result.Detail = "would register semedit MCP server"
		return result, nil
	}
	if err := atomicWrite(path, updated, mode); err != nil {
		return Result{}, err
	}
	if err := writeOwnership(path, req.Target, req.Scope, desiredRegistration(req)); err != nil {
		if rollbackErr := rollbackConfig(path, data, mode, exists); rollbackErr != nil {
			return Result{}, fmt.Errorf("write ownership record: %w; rollback config: %w", err, rollbackErr)
		}
		return Result{}, fmt.Errorf("write ownership record: %w", err)
	}
	result.Status = "installed"
	result.Changed = true
	result.Owned = true
	return result, nil
}

func rollbackConfig(path string, data []byte, mode fs.FileMode, existed bool) error {
	if existed {
		return atomicWrite(path, data, mode)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove newly created config: %w", err)
	}
	return nil
}

// Status reports only the selected target and scope. It does not execute the binary.
func Status(req StatusRequest) (Result, error) {
	if req.Target != TargetCopilot && req.Target != TargetCodex {
		return Result{}, fmt.Errorf("unsupported integration target %q", req.Target)
	}
	if req.Scope != ScopeUser && req.Scope != ScopeWorkspace {
		return Result{}, fmt.Errorf("scope must be user or workspace")
	}
	path, err := configPath(req.Target, req.Scope, req.Workspace)
	if err != nil {
		return Result{}, err
	}
	result := baseResult(Request{Target: req.Target, Scope: req.Scope}, path)
	data, _, exists, err := readConfig(path)
	if err != nil {
		return Result{}, err
	}
	if !exists {
		result.Status = "not-installed"
		result.Skill = "not-installed"
		return result, nil
	}
	var binary, profile string
	var installed bool
	switch req.Target {
	case TargetCopilot:
		binary, profile, installed, err = inspectCopilot(data)
	case TargetCodex:
		binary, profile, installed, err = inspectCodex(data)
	}
	if err != nil {
		result.Status = "malformed"
		result.Detail = err.Error()
		return result, nil
	}
	if !installed {
		result.Status = "not-installed"
		result.Skill = "not-installed"
		return result, nil
	}
	result.Status = "installed"
	result.Binary = binary
	result.Profile = profile
	result.Command = []string{binary, "mcp", "--profile", profile}
	result.BinaryExists = binary != "" && executable(binary)
	result.Owned, err = ownedRegistration(path, req.Target, req.Scope, data)
	if err != nil {
		return Result{}, err
	}
	result.Skill = "not-installed"
	return result, nil
}

// Uninstall removes only a registration previously owned by this installer.
func Uninstall(req StatusRequest) (Result, error) {
	if req.Target != TargetCopilot && req.Target != TargetCodex {
		return Result{}, fmt.Errorf("unsupported integration target %q", req.Target)
	}
	if req.Scope != ScopeUser && req.Scope != ScopeWorkspace {
		return Result{}, fmt.Errorf("scope must be user or workspace")
	}
	path, err := configPath(req.Target, req.Scope, req.Workspace)
	if err != nil {
		return Result{}, err
	}
	result := baseResult(Request{Target: req.Target, Scope: req.Scope}, path)
	data, mode, exists, err := readConfig(path)
	if err != nil {
		return Result{}, err
	}
	if !exists {
		result.Status = "not-installed"
		result.Skill = "not-installed"
		return result, nil
	}
	owned, err := ownedRegistration(path, req.Target, req.Scope, data)
	if err != nil {
		return Result{}, err
	}
	if !owned {
		result.Status = "not-owned"
		result.Skill = "not-installed"
		return result, fmt.Errorf("%w: %s", ErrNotOwned, path)
	}
	var updated []byte
	var removed bool
	switch req.Target {
	case TargetCopilot:
		updated, removed, err = removeCopilot(data)
	case TargetCodex:
		updated, removed, err = removeCodex(data)
	}
	if err != nil {
		return Result{}, err
	}
	if removed {
		if err := atomicWrite(path, updated, mode); err != nil {
			return Result{}, err
		}
	}
	if err := removeOwnership(path, req.Target, req.Scope); err != nil {
		return Result{}, err
	}
	result.Status = "uninstalled"
	result.Changed = removed
	result.Owned = true
	result.Skill = "not-installed"
	return result, nil
}

func baseResult(req Request, path string) Result {
	return Result{Target: req.Target, Scope: req.Scope, Path: path, Skill: "not-installed"}
}

func configPath(target Target, scope Scope, workspace string) (string, error) {
	if scope == ScopeWorkspace {
		root, err := filepath.Abs(workspace)
		if err != nil {
			return "", fmt.Errorf("resolve workspace: %w", err)
		}
		switch target {
		case TargetCopilot:
			// Copilot CLI reads project-local registrations from .mcp.json.
			return filepath.Join(root, ".mcp.json"), nil
		case TargetCodex:
			return filepath.Join(root, ".codex", "config.toml"), nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	switch target {
	case TargetCodex:
		if value := os.Getenv("CODEX_HOME"); value != "" {
			return filepath.Join(value, "config.toml"), nil
		}
		return filepath.Join(home, ".codex", "config.toml"), nil
	case TargetCopilot:
		if value := os.Getenv("SEMEDIT_COPILOT_USER_CONFIG"); value != "" {
			return filepath.Clean(value), nil
		}
		if value := os.Getenv("COPILOT_HOME"); value != "" {
			return filepath.Join(value, "mcp-config.json"), nil
		}
		return filepath.Join(home, ".copilot", "mcp-config.json"), nil
	}
	return "", fmt.Errorf("unsupported integration target %q", target)
}

func validateBinary(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat semedit binary: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		return fmt.Errorf("semedit binary is not an executable regular file: %s", path)
	}
	return nil
}

func executable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode()&0111 != 0
}

func readConfig(path string) ([]byte, fs.FileMode, bool, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is selected from explicit target and scope.
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0o600, false, nil
	}
	if err != nil {
		return nil, 0, false, fmt.Errorf("read %s: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, 0, false, fmt.Errorf("stat %s: %w", path, err)
	}
	return data, info.Mode().Perm(), true, nil
}

func atomicWrite(path string, data []byte, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if mode == 0 {
		mode = 0o600
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".semedit-install-*")
	if err != nil {
		return fmt.Errorf("create atomic config file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set config permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close config: %w", err)
	}
	oldTime := time.Time{}
	if info, statErr := os.Stat(path); statErr == nil {
		oldTime = info.ModTime()
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace config atomically: %w", err)
	}
	if info, statErr := os.Stat(path); statErr == nil && !info.ModTime().After(oldTime) {
		now := time.Now()
		if !oldTime.IsZero() && !now.After(oldTime) {
			now = oldTime.Add(time.Nanosecond)
		}
		_ = os.Chtimes(path, now, now)
	}
	return nil
}

type ownershipRecord struct {
	Target             string `json:"target"`
	Scope              string `json:"scope"`
	Path               string `json:"path"`
	RegistrationDigest string `json:"registration_digest"`
}

func ownershipPath(config string, target Target, scope Scope) string {
	base := filepath.Join(filepath.Dir(config), ".semedit")
	return filepath.Join(base, fmt.Sprintf("%s-%s.json", target, scope))
}

func ownedRegistration(config string, target Target, scope Scope, current []byte) (bool, error) {
	data, err := os.ReadFile(ownershipPath(config, target, scope))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read ownership record: %w", err)
	}
	var record ownershipRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return false, fmt.Errorf("parse ownership record: %w", err)
	}
	if record.Target != string(target) || record.Scope != string(scope) || filepath.Clean(record.Path) != filepath.Clean(config) {
		return false, nil
	}
	digest, err := registrationDigest(current, target)
	if err != nil {
		return false, nil
	}
	return record.RegistrationDigest == digest, nil
}

func writeOwnership(config string, target Target, scope Scope, registration []byte) error {
	digest := fmt.Sprintf("%x", sha256.Sum256(registration))
	data, err := json.Marshal(ownershipRecord{Target: string(target), Scope: string(scope), Path: config, RegistrationDigest: digest})
	if err != nil {
		return fmt.Errorf("encode ownership record: %w", err)
	}
	return atomicWrite(ownershipPath(config, target, scope), append(data, '\n'), 0o600)
}

func registrationDigest(data []byte, target Target) (string, error) {
	registration, err := currentRegistration(data, target)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(registration)), nil
}

func currentRegistration(data []byte, target Target) ([]byte, error) {
	switch target {
	case TargetCopilot:
		return currentCopilotRegistration(data)
	case TargetCodex:
		return currentCodexRegistration(data)
	default:
		return nil, fmt.Errorf("unsupported integration target %q", target)
	}
}

func desiredRegistration(req Request) []byte {
	switch req.Target {
	case TargetCopilot:
		data, _ := json.Marshal(copilotRegistration(req))
		return data
	case TargetCodex:
		return []byte(codexSection(req))
	default:
		return nil
	}
}

func removeOwnership(config string, target Target, scope Scope) error {
	path := ownershipPath(config, target, scope)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove ownership record: %w", err)
	}
	return nil
}

func quoteTOML(value string) string {
	return strconv.Quote(value)
}
