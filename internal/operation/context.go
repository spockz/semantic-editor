// Package operation carries the ingress-neutral call context between CLI/MCP frontends and operation handlers.
package operation

import (
	"context"
	"slices"

	"github.com/spockz/semantic-editor/internal/backend"
)

// CallContext carries request-scoped state shared by every operation invocation.
type CallContext struct {
	Ctx               context.Context
	WorkDir           string
	Project           backend.ProjectContext
	Registry          *Registry
	Service           *backend.Service
	InBatch           bool
	DeferImports      bool
	DeferVerification bool
	// GoFormattingPaths scopes batch built-in Go formatting; nil preserves standalone scope and empty skips formatting.
	GoFormattingPaths []string
	// GoPackagePatterns freezes Go diagnostics to the packages selected before batch publication.
	GoPackagePatterns map[string][]string
	// GoPackageScopeDetail explains whether frozen verification fell back to a broader scope.
	GoPackageScopeDetail string
	// AllowedLanguages applies an ingress language policy to verification; nil leaves CLI calls unrestricted.
	AllowedLanguages []backend.LanguageID
	StageRuntime     StageRuntime
}

// StageRuntime owns volatile MCP-only retained staging state.
type StageRuntime interface {
	Start(context.Context, string) (StageReceipt, error)
	Drop(context.Context, string, string) (StageReceipt, error)
	Abort(context.Context, string) (StageReceipt, error)
	Commit(context.Context, string, bool, bool) (StageCommitResult, error)
}

// AllowsLanguage reports whether the ingress policy permits one language.
func (cc CallContext) AllowsLanguage(language backend.LanguageID) bool {
	return cc.AllowedLanguages == nil || slices.Contains(cc.AllowedLanguages, language)
}

// NewCallContext builds the shared call context for one ingress request.
// The registry is the default operation set so batch entries recurse through Dispatch.
func NewCallContext(workDir string, project backend.ProjectContext) CallContext {
	return CallContext{Ctx: context.Background(), WorkDir: workDir, Project: project, Registry: DefaultRegistry()}
}

// WithMutationObserver attaches ingress-owned session synchronization to mutating operations.
func WithMutationObserver(ctx context.Context, observer func(context.Context) error) context.Context {
	return context.WithValue(ctx, mutationObserverContextKey{}, observer)
}

type mutationObserverContextKey struct{}

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
	requestTrust := project.WorkspaceTrust.Trusted && (project.WorkspaceTrust.WorkspaceRoot == "" || project.WorkspaceTrust.Allows(project.RootDir))
	trusted := requestTrust || cc.Project.WorkspaceTrust.Allows(project.RootDir)
	project.WorkspaceTrust = backend.NewWorkspaceTrust(project.RootDir, trusted)
	return project
}
