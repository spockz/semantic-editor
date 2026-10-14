// Package contractedit defines typed inputs and evidence outputs for contract edits outside the wire registry.
// WHY: This file keeps domain DTOs independent from the operation registry and receipt types.
package contractedit

import (
	"context"
	"github.com/spockz/semantic-editor/internal/contractplan"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"go/ast"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

// MemberTarget selects one exact direct method declaration.
type MemberTarget struct {
	File   string
	Symbol string
}

// ExtractRequest describes an interface extraction independent of CLI or MCP input parsing.
type ExtractRequest struct {
	WorkspaceRoot    string
	File             string
	InType           string
	Symbol           string
	DestinationFile  string
	ReceiverKind     contractplan.ReceiverKind
	Members          []MemberTarget
	ExpectedRevision *string
}

// ImplementRequest describes contract implementation independent of CLI or MCP input parsing.
type ImplementRequest struct {
	WorkspaceRoot    string
	File             string
	Symbol           string
	Contract         string
	ContractFile     string
	DestinationFile  string
	ReceiverKind     contractplan.ReceiverKind
	BodyPolicy       contractplan.BodyPolicy
	ExpectedRevision *string
}

// Revision binds a workspace-relative source path to its full-content digest.
type Revision struct {
	Path     string
	Revision string
}

// ContinuationCandidate identifies a symbol that may be projected after final-source checks.
type ContinuationCandidate struct {
	File             string
	Symbol           string
	ExpectedRevision *string
}

// Result carries execution evidence for operation-owned receipt finalization.
type Result struct {
	Status            string
	NoOp              bool
	File              string
	Symbol            string
	Contract          string
	ContractFile      string
	Detail            string
	Changes           []Revision
	ReadSet           []Revision
	Signatures        []MethodShape
	ReceiptCandidates []ContinuationCandidate
}

// MethodShape preserves a typed method signature for operation output projection.
type MethodShape struct {
	Name       string
	Parameters []ParameterShape
	Results    []ResultShape
}

// ParameterShape describes one method parameter.
type ParameterShape struct {
	Name     string
	Type     string
	Variadic bool
}

// ResultShape describes one method result.
type ResultShape struct {
	Name string
	Type string
}

// Snapshot is the immutable package and source view used by contract execution.
type Snapshot struct {
	Context          context.Context
	Buffer           *sourcebuf.Buffer
	Root             string
	ModuleRoot       string
	Env              []string
	Packages         []*packages.Package
	ByPath           map[string]*packages.Package
	RequiredPkgPaths map[string]bool
	Syntax           map[string]*ast.File
	Contents         map[string][]byte
	ReadSet          []Revision
}

// PackageForFile returns the captured package that owns one canonical source path.
func (snapshot *Snapshot) PackageForFile(path string) *packages.Package {
	if snapshot == nil {
		return nil
	}
	for _, pkg := range snapshot.ByPath {
		for _, file := range pkg.GoFiles {
			absolute, err := filepath.Abs(file)
			if err != nil {
				continue
			}
			canonical, err := filepath.EvalSymlinks(absolute)
			if err == nil && canonical == path {
				return pkg
			}
		}
	}
	return nil
}
