// Package navigation owns language-backend lookup and rename execution.
// WHY: Semantic navigation, including post-rename identity proof, forms one execution feature outside registry ingress.
package navigation

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"semedit/internal/backend"
	"semedit/internal/backend/pathutil"
)

// RenameRequest carries execution inputs and publication context for one backend rename.
type RenameRequest struct {
	Project           backend.ProjectContext
	Symbol            string
	To                string
	OrganizeImports   bool
	DeferVerification bool
	Buffered          bool
}

type postRenameIdentity struct {
	Query          string
	ExpectedSymbol string
}

type revisionMatch struct {
	Revision string
	Count    int
}

// Lookup resolves one symbol through the selected language backend.
func Lookup(ctx context.Context, service *backend.Service, project backend.ProjectContext, symbol string) (*backend.LookupResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if service == nil {
		return nil, fmt.Errorf("navigation backend service is required")
	}
	return service.Lookup(ctx, project, symbol)
}

// Rename performs a rename and proves the resulting identity when the call is not buffered.
func Rename(ctx context.Context, service *backend.Service, request RenameRequest) (*backend.RenameResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if service == nil {
		return nil, fmt.Errorf("navigation backend service is required")
	}
	project := request.Project
	result, err := service.Rename(ctx, backend.RenameRequest{
		Project:           project,
		Symbol:            request.Symbol,
		To:                request.To,
		OrganizeImports:   request.OrganizeImports,
		DeferFormatting:   request.DeferVerification,
		DeferVerification: request.DeferVerification,
	})
	if err != nil {
		var diagnosticsError *backend.RenameDiagnosticsError
		if errors.As(err, &diagnosticsError) && diagnosticsError.Result != nil {
			diagnosticsError.Result.Source = cloneRenameLookup(diagnosticsError.Result.Lookup)
			diagnosticsError.Result.Lookup = nil
			diagnosticsError.Result.ProjectRoot = project.RootDir
			diagnosticsError.Result.ApplicationState = "persisted"
			if request.Buffered {
				diagnosticsError.Result.ApplicationState = "applied_working"
			}
		}
		return result, err
	}
	if result == nil {
		return nil, fmt.Errorf("rename backend returned no result")
	}
	result.ProjectRoot = project.RootDir
	result.ApplicationState = "persisted"
	if request.Buffered {
		result.ApplicationState = "applied_working"
		result.Source = cloneRenameLookup(result.Lookup)
		result.Lookup = nil
		result.ContinuationUnavailable = "rename is part of an in-progress workspace batch"
		return result, nil
	}
	return resolvePostRenameIdentity(ctx, service, project, request.To, result), nil
}

func resolvePostRenameIdentity(ctx context.Context, service *backend.Service, project backend.ProjectContext, requestedName string, result *backend.RenameResult) *backend.RenameResult {
	if result.Lookup == nil {
		result.ContinuationUnavailable = "backend returned no pre-rename identity"
		return result
	}
	result.Source = cloneRenameLookup(result.Lookup)
	result.Lookup = nil
	identity := postRenameLookupName(result.Source, requestedName)
	if identity.Query == "" {
		result.ContinuationUnavailable = "rename source did not contain a qualified identity"
		return result
	}
	lookupProject := project
	if result.TargetFile != "" {
		lookupProject.File = result.TargetFile
	}
	resolved, err := service.Lookup(ctx, lookupProject, identity.Query)
	if err != nil {
		result.ContinuationUnavailable = "post-rename lookup failed: " + err.Error()
		return result
	}
	if resolved == nil || resolved.IsAmbiguous() || resolved.Definition == nil {
		result.ContinuationUnavailable = "post-rename lookup did not resolve one declaration"
		return result
	}
	resolvedSymbol := resolved.Definition.Identity()
	if resolvedSymbol == "" || resolvedSymbol != identity.ExpectedSymbol {
		result.ContinuationUnavailable = "post-rename lookup did not resolve the expected qualified identity"
		return result
	}
	if result.Source.Definition == nil {
		result.ContinuationUnavailable = "rename source did not contain a qualified identity"
		return result
	}
	sourceFile, err := pathutil.FilePathFromURI(result.Source.Definition.Location.URI)
	if err != nil {
		result.ContinuationUnavailable = "rename source location was invalid: " + err.Error()
		return result
	}
	resolvedFile, err := pathutil.FilePathFromURI(resolved.Definition.Location.URI)
	if err != nil {
		result.ContinuationUnavailable = "post-rename location was invalid: " + err.Error()
		return result
	}
	if result.Source.Definition == nil || (result.TargetFile == "" && !sameWorkspacePath(project.RootDir, sourceFile, resolvedFile)) ||
		(result.TargetFile != "" && !sameWorkspacePath(project.RootDir, result.TargetFile, resolvedFile)) {
		result.ContinuationUnavailable = "post-rename lookup resolved a different source file"
		return result
	}
	result.Lookup = resolved
	return resolvePostRenameRevision(ctx, service, project, resolvedSymbol, resolvedFile, result)
}

func resolvePostRenameRevision(ctx context.Context, service *backend.Service, project backend.ProjectContext, symbol, file string, result *backend.RenameResult) *backend.RenameResult {
	inspectProject := project
	inspectProject.File = file
	inspection, err := service.Inspect(ctx, backend.InspectRequest{Project: inspectProject, Symbol: symbol})
	if err != nil || inspection == nil {
		if err != nil {
			result.ContinuationUnavailable = "post-rename inspection failed: " + err.Error()
		} else {
			result.ContinuationUnavailable = "post-rename inspection returned no result"
		}
		return result
	}
	match := matchingRevision(project.RootDir, symbol, file, inspection.Matches)
	result.Revision = match.Revision
	if match.Count > 1 {
		result.Revision = ""
		result.ContinuationUnavailable = "post-rename inspection matched multiple declarations"
	} else if match.Count != 1 || match.Revision == "" {
		result.ContinuationUnavailable = "post-rename inspection did not prove a matching file revision"
	}
	return result
}

func matchingRevision(root, symbol, file string, matches []backend.InspectedSymbol) revisionMatch {
	var result revisionMatch
	for _, match := range matches {
		if match.QualifiedName != symbol || !sameWorkspacePath(root, match.File, file) {
			continue
		}
		result.Count++
		if result.Count > 1 {
			return revisionMatch{Count: result.Count}
		}
		result.Revision = match.Revision
	}
	return result
}

func postRenameLookupName(source *backend.LookupResult, requestedName string) postRenameIdentity {
	if source == nil {
		return postRenameIdentity{}
	}
	oldIdentity := ""
	if source.Definition != nil {
		oldIdentity = source.Definition.Identity()
	}
	newLeaf := backend.NormalizeRenameInput(requestedName)
	if oldIdentity == "" || newLeaf == "" {
		return postRenameIdentity{}
	}
	dot := strings.LastIndex(oldIdentity, ".")
	colon := strings.LastIndex(oldIdentity, "::")
	separator, width := dot, 1
	if colon > separator {
		separator, width = colon, 2
	}
	if separator < 0 {
		return postRenameIdentity{Query: newLeaf, ExpectedSymbol: newLeaf}
	}
	qualified := oldIdentity[:separator+width] + newLeaf
	return postRenameIdentity{Query: qualified, ExpectedSymbol: qualified}
}

func cloneRenameLookup(source *backend.LookupResult) *backend.LookupResult {
	if source == nil {
		return nil
	}
	clone := *source
	if source.Definition != nil {
		definition := *source.Definition
		clone.Definition = &definition
	}
	clone.Usages = append([]*backend.NamedTarget(nil), source.Usages...)
	clone.Candidates = append([]*backend.NamedTarget(nil), source.Candidates...)
	return &clone
}

func sameWorkspacePath(root, left, right string) bool {
	leftPath, leftErr := canonicalPath(root, left)
	rightPath, rightErr := canonicalPath(root, right)
	return leftErr == nil && rightErr == nil && leftPath == rightPath
}

func canonicalPath(root, path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}
