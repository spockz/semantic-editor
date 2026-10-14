// Package mcp keeps request dispatch concise while workspace policy lives in an independently testable package.
package mcp

import (
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/mcp/workspace"
)

// ResolveRequestRoot delegates root policy to the workspace package.
func ResolveRequestRoot(configuredRoot, requestedRoot string) (string, error) {
	return workspace.ResolveRequestRoot(configuredRoot, requestedRoot)
}

// CanonicalTrust delegates trust binding to the workspace package.
func CanonicalTrust(root string, trusted bool) backend.WorkspaceTrust {
	return workspace.CanonicalTrust(root, trusted)
}

// ValidateRequestPaths delegates path checks to the workspace package.
func ValidateRequestPaths(root string, raw map[string]any) error {
	return workspace.ValidateRequestPaths(root, raw)
}

// PrepareWorkspaceArguments delegates workspace argument validation to the workspace package.
func PrepareWorkspaceArguments(configuredRoot string, raw map[string]any) (string, error) {
	return workspace.PrepareWorkspaceArguments(configuredRoot, raw)
}

// RequestTrustedRoot delegates request trust evaluation to the workspace package.
func RequestTrustedRoot(root string, configured backend.WorkspaceTrust, raw map[string]any) bool {
	return workspace.RequestTrustedRoot(root, configured, raw)
}
