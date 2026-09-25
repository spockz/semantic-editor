// Package operation wires bounded Java Maven actions into the central registry.
package operation

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"semedit/internal/backend"
	"semedit/internal/maven"
)

// MavenReq requests one fixed Java Maven action.
type MavenReq struct {
	Project      backend.ProjectContext
	Root         string
	Tool         string
	SystemBin    string
	AllowNetwork bool
	Goal         string
}

// GetProjectContext returns the request project.
func (r MavenReq) GetProjectContext() backend.ProjectContext { return r.Project }

// SetProjectContext sets the effective request project.
func (r *MavenReq) SetProjectContext(p backend.ProjectContext) { r.Project = p }

// MavenRes reports one Maven action.
type MavenRes = maven.Result

var mavenParams = []ParameterContract{
	{Name: "language", CLIName: "language", JSONName: "language", Type: ParamString, Default: "java", Enums: []string{"java"}},
	{Name: "root", CLIName: "root", JSONName: "root", Type: ParamString, Description: "Canonical Maven workspace root (defaults to the request working directory)"},
	{Name: wireTrustWorkspace, CLIName: wireCLITrustWorkspace, JSONName: wireTrustWorkspace, Type: ParamBoolean, Default: false, Description: "Explicitly trust this workspace for this request"},
	{Name: "maven_tool", CLIName: "maven-tool", JSONName: "maven_tool", Type: ParamString, Default: "auto", Enums: []string{"auto", "wrapper", "system"}, Description: "Maven launcher selection"},
	{Name: "maven_bin", CLIName: "maven-bin", JSONName: "maven_bin", Type: ParamString, Description: "Absolute system Maven executable"},
	{Name: "allow_network", CLIName: "allow-network", JSONName: "allow_network", Type: ParamBoolean, Default: false, Description: "Allow Maven to access the network for this request"},
}

func parseMaven(raw map[string]any, goal string) (MavenReq, error) {
	if err := CheckParams(raw, mavenParams); err != nil {
		return MavenReq{}, err
	}
	lang, err := ParseEnum(raw, "language", "language", []string{"java"}, false, "java")
	if err != nil {
		return MavenReq{}, err
	}
	trusted, err := ParseBool(raw, wireTrustWorkspace, wireCLITrustWorkspace, false)
	if err != nil {
		return MavenReq{}, err
	}
	tool, err := ParseEnum(raw, "maven_tool", "maven-tool", []string{"auto", "wrapper", "system"}, false, "auto")
	if err != nil {
		return MavenReq{}, err
	}
	bin, err := ParseString(raw, "maven_bin", "maven-bin", false)
	if err != nil {
		return MavenReq{}, err
	}
	allowNetwork, err := ParseBool(raw, "allow_network", "allow-network", false)
	if err != nil {
		return MavenReq{}, err
	}
	root, err := ParseString(raw, "root", "root", false)
	if err != nil {
		return MavenReq{}, err
	}
	return MavenReq{Project: backend.ProjectContext{RootDir: root, Language: backend.LanguageID(lang), WorkspaceTrust: backend.WorkspaceTrust{Trusted: trusted}}, Root: root, Tool: tool, SystemBin: bin, AllowNetwork: allowNetwork, Goal: goal}, nil
}
func mavenRun(_ context.Context, cc CallContext, req MavenReq) (MavenRes, error) {
	trustRoot := backend.CanonicalWorkspaceRoot(cc.WorkDir)
	root := req.Project.RootDir
	if root == "" {
		root = trustRoot
	}
	root = backend.CanonicalWorkspaceRoot(root)
	if !withinRoot(trustRoot, root) {
		return MavenRes{}, fmt.Errorf("%w: %s", maven.ErrRootOutsideTrust, root)
	}
	req.Project.WorkspaceTrust = backend.NewWorkspaceTrust(trustRoot, req.Project.WorkspaceTrust.Trusted)
	return maven.Run(cc.Ctx, maven.Request{Root: root, Trust: req.Project.WorkspaceTrust, Tool: maven.Tool(req.Tool), SystemBin: req.SystemBin, AllowNetwork: req.AllowNetwork, Goal: req.Goal})
}

func withinRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
func mavenDef(key, cli, mcp, goal, summary string) Def[MavenReq, MavenRes] {
	return Def[MavenReq, MavenRes]{Key: key, Summary: summary, Params: mavenParams, Level: LevelBuild, CLIName: cli, MCPName: mcp, Batchable: false, ReadOnly: false,
		Parse:    func(raw map[string]any) (MavenReq, error) { return parseMaven(raw, goal) },
		Handlers: map[backend.LanguageID]func(context.Context, CallContext, MavenReq) (MavenRes, error){backend.LanguageJava: mavenRun},
		Format: func(res MavenRes) (string, error) {
			data, err := json.Marshal(res)
			if err != nil {
				return "", fmt.Errorf("format Maven result: %w", err)
			}
			return string(data), nil
		},
		ExampleRaw: map[string]any{"language": "java", wireTrustWorkspace: true, "maven_tool": "auto", "allow_network": false}}
}
func registerMavenOps(registry *Registry) error {
	if err := Register(registry, mavenDef("maven_compile", "maven-compile", "semantic_maven_compile", "test-compile", "Use this tool instead of a shell Maven invocation when compiling tests for a trusted Java root POM with fixed test-compile scope")); err != nil {
		return err
	}
	return Register(registry, mavenDef("maven_test", "maven-test", "semantic_maven_test", "test", "Use this tool instead of a shell Maven invocation when running tests for a trusted Java root POM with fixed test scope"))
}
