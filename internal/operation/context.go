// Package operation carries the ingress-neutral call context between CLI/MCP frontends and operation handlers.
package operation

import (
	"context"

	"semedit/internal/backend"
)

// CallContext carries request-scoped state shared by every operation invocation.
type CallContext struct {
	Ctx          context.Context
	WorkDir      string
	Project      backend.ProjectContext
	Registry     *Registry
	Service      *backend.Service
	InBatch      bool
	DeferImports bool
}

// NewCallContext builds the shared call context for one ingress request.
// The registry is the default operation set so batch entries recurse through Dispatch.
func NewCallContext(workDir string, project backend.ProjectContext) CallContext {
	return CallContext{Ctx: context.Background(), WorkDir: workDir, Project: project, Registry: DefaultRegistry()}
}

// effectiveProject merges an operation request project with ambient call context state.
// Request fields win; empty root and language fall back to the call context, and workspace
// trust is re-scoped to the effective root while preserving the request consent flag.
func effectiveProject(cc CallContext, project backend.ProjectContext) backend.ProjectContext {
	if project.RootDir == "" {
		project.RootDir = cc.WorkDir
	}
	if project.RootDir == "" {
		project.RootDir = cc.Project.RootDir
	}
	if project.RootDir == "" {
		project.RootDir = "."
	}
	if project.Language == "" {
		project.Language = cc.Project.Language
	}
	project.WorkspaceTrust = backend.NewWorkspaceTrust(project.RootDir, project.WorkspaceTrust.Trusted)
	return project
}
