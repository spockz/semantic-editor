// Restores construct candidate inspection and revision guards while preserving the current semantic mutation pipeline.

package astedit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spockz/semantic-editor/internal/sourcebuf"
)

// GoInspectConstructKinds lists nested replacement kinds with path-addressable candidates.
var GoInspectConstructKinds = []ConstructKind{
	ConstructLoop, ConstructIf, ConstructElse, ConstructCase, ConstructSelect, ConstructDefer, ConstructStatementOrExpr,
}

func containsConstructKind(kinds []ConstructKind, kind ConstructKind) bool {
	return slices.Contains(kinds, kind)
}

// ConstructCandidateInfo describes one exact candidate and its normalized syntax.
type ConstructCandidateInfo struct {
	ConstructPath    string `json:"construct_path"`
	EndConstructPath string `json:"end_construct_path,omitempty"`
	Syntax           string `json:"syntax"`
}

// ConstructCandidateSet reports the current candidate list and its opaque revision.
type ConstructCandidateSet struct {
	InFunction        string                   `json:"in_function"`
	Kind              ConstructKind            `json:"kind"`
	Discriminator     string                   `json:"discriminator,omitempty"`
	Candidates        []ConstructCandidateInfo `json:"candidates"`
	SelectionRevision string                   `json:"expected_selection_revision"`
	SourceRevision    string                   `json:"-"`
}

// ConstructSelectionError carries the current matching candidate list when a nested selector fails.
type ConstructSelectionError struct {
	Status           string
	File             string
	Candidates       ConstructCandidateSet
	ExpectedRevision string
	Cause            error
}

func (e *ConstructSelectionError) Error() string {
	if e == nil {
		return "construct selection failed"
	}
	var message strings.Builder
	switch {
	case e.Cause != nil:
		message.WriteString(e.Cause.Error())
	case e.Status == "selection_revision_mismatch":
		message.WriteString("selection revision mismatch")
	default:
		message.WriteString("construct selection failed")
	}
	if e.File != "" {
		fmt.Fprintf(&message, "\nfile: %s", e.File)
	}
	fmt.Fprintf(&message, "\nin_function: %s\nkind: %s", e.Candidates.InFunction, e.Candidates.Kind)
	if e.Candidates.Discriminator != "" {
		fmt.Fprintf(&message, "\ndiscriminator: %s", e.Candidates.Discriminator)
	}
	if e.Status == "selection_revision_mismatch" {
		fmt.Fprintf(&message, "\nCurrent replacement refused before writing for %s. The supplied expected_selection_revision differs from the token recomputed from the current ordered candidate list of paths and normalized constructs. Reselect from the current choices and token for this request target without another lookup while the buffer is current. Do not change source to resolve this mismatch. Earlier batch steps are reported in the batch result.", e.File)
	} else {
		if e.File != "" {
			fmt.Fprintf(&message, "\nCurrent replacement refused before writing for %s.", e.File)
		} else {
			message.WriteString("\nCurrent replacement refused before writing.")
		}
		switch e.Status {
		case "ambiguous":
			message.WriteString(" The selector matches multiple candidates for this request target; choose one construct_path shown below.")
		case "invalid_path":
			if len(e.Candidates.Candidates) > 0 {
				message.WriteString(" The supplied construct_path does not identify a current candidate for this request target; choose a path shown below.")
			} else {
				message.WriteString(" The supplied construct_path does not identify a current candidate; reconsider the selector.")
			}
		case "not_found":
			message.WriteString(" No candidate matches this selector for the current request target; reconsider the selector.")
		}
	}
	fmt.Fprintf(&message, "\nCurrent expected_selection_revision: %s", e.Candidates.SelectionRevision)
	for _, candidate := range e.Candidates.Candidates {
		fmt.Fprintf(&message, "\nconstruct_path %s: %s", candidate.ConstructPath, candidate.Syntax)
	}
	return message.String()
}

func (e *ConstructSelectionError) Unwrap() error { return e.Cause }

type constructSelectionRevisionDescriptor struct {
	Version       string                   `json:"version"`
	Owner         string                   `json:"owner"`
	Kind          ConstructKind            `json:"kind"`
	Discriminator string                   `json:"discriminator"`
	Candidates    []ConstructCandidateInfo `json:"candidates"`
}

// InspectConstructCandidates returns current candidates for one nested replacement selector.
func InspectConstructCandidates(ctx context.Context, filePath, owner string, kind ConstructKind, discriminator string) (ConstructCandidateSet, error) {
	if !containsConstructKind(GoInspectConstructKinds, kind) {
		return ConstructCandidateSet{}, fmt.Errorf("construct inspection does not support kind %q", kind)
	}
	cleanPath := filepath.Clean(filePath)
	content, err := sourcebuf.ReadFile(ctx, cleanPath)
	if err != nil {
		return ConstructCandidateSet{}, fmt.Errorf("read construct inspection target: %w", err)
	}
	return inspectConstructCandidatesFromSource(cleanPath, content, owner, kind, discriminator)
}

func inspectConstructCandidatesFromSource(filePath string, content []byte, owner string, kind ConstructKind, discriminator string) (ConstructCandidateSet, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.AllErrors)
	if err != nil {
		return ConstructCandidateSet{}, &SyntaxError{File: filePath, Pos: extractSyntaxPosition(fset, err), Cause: err, Err: ErrSyntax}
	}
	fn, err := findConstructOwner(file, owner)
	if err != nil {
		return ConstructCandidateSet{}, err
	}
	var candidates []constructCandidate
	if kind == ConstructStatementOrExpr {
		var facts constructTypeFacts
		candidates, facts = collectStatementExpressionCandidates(fset, content, file, fn)
		matches := make([]constructCandidate, 0, len(candidates))
		unresolved := make([]constructCandidate, 0)
		selector := normalizeConstructSelector(discriminator)
		for _, candidate := range candidates {
			if candidate.key == selector {
				if candidate.class == unresolvedExpressionClass {
					unresolved = append(unresolved, candidate)
				} else {
					matches = append(matches, candidate)
				}
			}
		}
		if len(unresolved) > 0 {
			return ConstructCandidateSet{}, unresolvedExpressionError(fset, discriminator, canonicalConstructOwner(fn), facts, unresolved)
		}
		candidates = matches
	} else {
		candidates = matchingConstructs(fset, collectConstructCandidates(fset, fn.Body, kind), kind, discriminator)
	}
	canonicalOwner := canonicalConstructOwner(fn)
	canonicalDiscriminator := normalizeConstructSelector(discriminator)
	infos := make([]ConstructCandidateInfo, 0, len(candidates))
	for _, candidate := range candidates {
		syntax, normalizeErr := normalizedConstructSyntax(candidate.node)
		if normalizeErr != nil {
			return ConstructCandidateSet{}, fmt.Errorf("normalize construct candidate at %s: %w", candidate.path, normalizeErr)
		}
		infos = append(infos, ConstructCandidateInfo{ConstructPath: candidate.path, Syntax: syntax})
	}
	revision, err := constructSelectionRevision(canonicalOwner, kind, canonicalDiscriminator, infos)
	if err != nil {
		return ConstructCandidateSet{}, err
	}
	for index := range infos {
		infos[index].EndConstructPath = infos[index].ConstructPath
	}
	return ConstructCandidateSet{
		InFunction: canonicalOwner, Kind: kind, Discriminator: canonicalDiscriminator,
		Candidates: infos, SelectionRevision: revision, SourceRevision: goReadRevision(content),
	}, nil
}

func canonicalConstructOwner(fn *ast.FuncDecl) string {
	if fn.Recv == nil {
		return fn.Name.Name
	}
	return extractReceiverName(fn.Recv) + "." + fn.Name.Name
}

func normalizedConstructSyntax(node ast.Node) (string, error) {
	var syntax bytes.Buffer
	if err := format.Node(&syntax, token.NewFileSet(), node); err != nil {
		return "", err
	}
	return syntax.String(), nil
}

func constructSelectionRevision(owner string, kind ConstructKind, discriminator string, candidates []ConstructCandidateInfo) (string, error) {
	descriptor := constructSelectionRevisionDescriptor{
		Version: "semedit.construct-selection/v1", Owner: owner, Kind: kind,
		Discriminator: discriminator, Candidates: candidates,
	}
	encoded, err := json.Marshal(descriptor)
	if err != nil {
		return "", fmt.Errorf("encode construct selection revision: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func selectionError(status string, candidates ConstructCandidateSet, cause error, expected string) error {
	return &ConstructSelectionError{Status: status, Candidates: candidates, Cause: cause, ExpectedRevision: expected}
}

func constructSelectionErrorStatus(err error) string {
	if outcome, ok := errors.AsType[*constructSelectionOutcomeError](err); ok {
		return outcome.Status
	}
	return ""
}

type constructSelectionOutcomeError struct {
	Status string
	Cause  error
}

func (e *constructSelectionOutcomeError) Error() string { return e.Cause.Error() }

func (e *constructSelectionOutcomeError) Unwrap() error { return e.Cause }

func selectionOutcome(status string, err error) error {
	return &constructSelectionOutcomeError{Status: status, Cause: err}
}
