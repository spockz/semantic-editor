// Package operation adapts only bounded, backend-supported LSP actions for configured verification hooks.
package operation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"semedit/internal/backend"
	"semedit/internal/backends"
	"semedit/internal/projectverify"
)

func serviceForContext(cc CallContext) *backend.Service {
	if cc.Service != nil {
		return cc.Service
	}
	return backends.NewDefaultService()
}

func hasConfiguredCommandHooks(hooks []projectverify.Hook) bool {
	for _, hook := range hooks {
		if hook.Disabled || hook.LSP != nil {
			continue
		}
		if hook.Shell != nil || len(hook.Exec) > 0 {
			return true
		}
	}
	return false
}

func verificationLSPRunner(service *backend.Service, project backend.ProjectContext, phase projectverify.Phase) projectverify.LSPRunner {
	return func(ctx context.Context, hook projectverify.Hook, scope projectverify.Scope) (projectverify.HookResult, error) {
		result := projectverify.HookResult{ID: hook.ID, Kind: "lsp", Signal: hook.Signal, ConfigPath: hook.ConfigPath, ScopePaths: append([]string(nil), scope.Paths...)}
		if hook.LSP == nil {
			return result, fmt.Errorf("hook %q has no LSP action", hook.ID)
		}
		if phase != projectverify.PhaseNormalize {
			return result, fmt.Errorf("LSP code action %q is not supported in the check phase", hook.ID)
		}
		if hook.LSP.Command != "" {
			return result, fmt.Errorf("LSP command %q is unsupported by the selected backend", hook.LSP.Command)
		}
		if hook.LSP.Language != string(backend.LanguageJava) || hook.LSP.ActionKind != "source.organizeImports" {
			return result, fmt.Errorf("unsupported LSP action %q for language %q", hook.LSP.ActionKind, hook.LSP.Language)
		}
		for _, path := range scope.Paths {
			fullPath := filepath.Join(project.RootDir, filepath.FromSlash(path))
			// #nosec G304 -- scope paths come from the trusted project verification plan.
			before, err := os.ReadFile(fullPath)
			if err != nil {
				return result, fmt.Errorf("read Java source before organize imports %q: %w", path, err)
			}
			selected := project
			selected.Language = backend.LanguageJava
			selected.File = path
			verified, err := service.Verify(ctx, backend.VerifyRequest{Project: selected, Path: path, OrganizeImports: true})
			if err != nil {
				return result, fmt.Errorf("organize imports for %q: %w", path, err)
			}
			for _, diagnostic := range verified.Diagnostics {
				result.Diagnostics = append(result.Diagnostics, diagnostic.Message)
			}
			// #nosec G304 -- scope paths come from the trusted project verification plan.
			after, err := os.ReadFile(fullPath)
			if err != nil {
				return result, fmt.Errorf("read Java source after organize imports %q: %w", path, err)
			}
			if string(before) != string(after) {
				result.Changed = append(result.Changed, path)
			}
		}
		result.Status = projectverify.HookSucceeded
		return result, nil
	}
}

func workspaceTrustRequiredHook(plan projectverify.Plan, hooks []projectverify.Hook, scope projectverify.Scope) (string, error) {
	if !hasConfiguredCommandHooks(hooks) && !hasMutatingLSPHooks(hooks) {
		return "", nil
	}
	for _, hook := range hooks {
		if hook.Disabled {
			continue
		}
		_, applies, err := projectverify.SelectHookScope(plan, hook, scope)
		if err != nil {
			return "", fmt.Errorf("select scope for verification hook %q: %w", hook.ID, err)
		}
		if !applies {
			continue
		}
		isCommand := hook.LSP == nil && (hook.Shell != nil || len(hook.Exec) > 0)
		isLSPAction := hook.LSP != nil && (hook.LSP.ActionKind != "" || hook.LSP.Command != "")
		if isCommand || isLSPAction {
			return hook.ID, nil
		}
	}
	return "", nil
}
