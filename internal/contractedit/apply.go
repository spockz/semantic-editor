// Package contractedit owns execution-time source capture, validation, and publication for Go contract edits.
// WHY: This file applies one preflighted plan through the active source buffer and returns revision-bound candidates.
package contractedit

import (
	"context"
	"fmt"
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"

	"semedit/internal/contractplan"
	"semedit/internal/sourcebuf"
)

type candidateCompletion func(*Result) error

func finishPlan(ctx context.Context, snapshot *Snapshot, destination, selected, symbol string, expected *string, plan contractplan.Plan, validate func(*packages.Package) error, complete candidateCompletion) (Result, error) {
	if err := checkRevision(snapshot, selected, expected); err != nil {
		return Result{}, err
	}
	if plan.Declaration == "" {
		return finishNoOp(snapshot, destination, symbol, plan, validate, complete)
	}
	return applyCandidate(ctx, snapshot, destination, symbol, plan, validate, complete)
}

func checkRevision(snapshot *Snapshot, selected string, expected *string) error {
	if expected == nil || *expected == "" {
		return nil
	}
	data, exists := snapshot.Contents[selected]
	if !exists {
		return fmt.Errorf("selected source %q is missing from the current source snapshot", selected)
	}
	actual := Digest(data)
	if actual != *expected {
		return fmt.Errorf("selected source revision changed: expected %q, found %q", *expected, actual)
	}
	return nil
}

func finishNoOp(snapshot *Snapshot, destination, symbol string, plan contractplan.Plan, validate func(*packages.Package) error, complete candidateCompletion) (Result, error) {
	if err := snapshot.Buffer.AssertDisk(); err != nil {
		return Result{}, fmt.Errorf("validate contract no-op read set: %w", err)
	}
	for _, revision := range snapshot.ReadSet {
		path := filepath.Join(snapshot.Root, filepath.FromSlash(revision.Path))
		data, err := sourcebuf.ReadFile(snapshot.Context, path)
		if err != nil {
			return Result{}, fmt.Errorf("validate contract no-op read %q: %w", revision.Path, err)
		}
		if actual := Digest(data); actual != revision.Revision {
			return Result{}, fmt.Errorf("contract no-op read %q changed: expected %q, found %q", revision.Path, revision.Revision, actual)
		}
	}
	if validate != nil {
		if err := validate(snapshot.PackageForFile(destination)); err != nil {
			return Result{}, fmt.Errorf("validate existing contract no-op: %w", err)
		}
	}
	destinationRevision := Digest(snapshot.Contents[destination])
	result := Result{
		Status:            "no_op",
		NoOp:              true,
		File:              displayPath(snapshot.Root, destination),
		Symbol:            symbol,
		Detail:            "The requested contract is already satisfied.",
		ReadSet:           append([]Revision(nil), snapshot.ReadSet...),
		ReceiptCandidates: []ContinuationCandidate{{File: displayPath(snapshot.Root, destination), Symbol: symbol, ExpectedRevision: &destinationRevision}},
		Signatures:        planSignatures(plan),
	}
	if complete != nil {
		if err := complete(&result); err != nil {
			return Result{}, err
		}
	}
	return result, nil
}

func applyCandidate(ctx context.Context, snapshot *Snapshot, destination, symbol string, plan contractplan.Plan, validate func(*packages.Package) error, complete candidateCompletion) (Result, error) {
	destinationFile := snapshot.Syntax[destination]
	if destinationFile == nil {
		return Result{}, fmt.Errorf("destination file %q has no syntax in the current overlay", destination)
	}
	before, exists := snapshot.Contents[destination]
	if !exists {
		return Result{}, fmt.Errorf("destination file %q is absent from the source snapshot", destination)
	}
	candidate, err := RenderCandidate(before, destinationFile, fileSetFor(snapshot, destination), plan)
	if err != nil {
		return Result{}, err
	}
	finalPackage, err := PreflightCandidate(ctx, snapshot, destination, candidate)
	if err != nil {
		return Result{}, err
	}
	if validate != nil {
		if err := validate(finalPackage); err != nil {
			return Result{}, fmt.Errorf("validate generated contract output: %w", err)
		}
	}
	destinationRevision := Digest(candidate)
	result := Result{
		Status: map[bool]string{
			true:  "working_buffer",
			false: "persisted",
		}[sourcebuf.FromContext(ctx) != nil],
		File:              displayPath(snapshot.Root, destination),
		Symbol:            symbol,
		Detail:            "Generated and type-checked the requested contract declarations.",
		Changes:           []Revision{{Path: displayPath(snapshot.Root, destination), Revision: destinationRevision}},
		ReadSet:           append([]Revision(nil), snapshot.ReadSet...),
		ReceiptCandidates: []ContinuationCandidate{{File: displayPath(snapshot.Root, destination), Symbol: symbol, ExpectedRevision: &destinationRevision}},
		Signatures:        planSignatures(plan),
	}
	if complete != nil {
		if err := complete(&result); err != nil {
			return Result{}, err
		}
	}
	change := sourcebuf.Change{Path: destination, Before: before, After: candidate}
	if sourcebuf.FromContext(ctx) != nil {
		if err := snapshot.Buffer.Apply(ctx, []sourcebuf.Change{change}); err != nil {
			return Result{}, fmt.Errorf("apply generated contract to working buffer: %w", err)
		}
	} else if err := sourcebuf.WriteFileChecked(sourcebuf.WithoutBuffer(ctx), destination, candidate, snapshot.Buffer.AssertDisk); err != nil {
		return Result{}, fmt.Errorf("publish generated contract: %w", err)
	}
	return result, nil
}

func fileSetFor(snapshot *Snapshot, destination string) *token.FileSet {
	pkg := snapshot.PackageForFile(destination)
	if pkg == nil {
		return nil
	}
	return pkg.Fset
}

func planSignatures(plan contractplan.Plan) []MethodShape {
	shapes := make([]MethodShape, 0, len(plan.Methods))
	for _, method := range plan.Methods {
		shape := MethodShape{
			Name:       method.Signature.Name,
			Parameters: make([]ParameterShape, 0, len(method.Signature.Parameters)),
			Results:    make([]ResultShape, 0, len(method.Signature.Results)),
		}
		for _, parameter := range method.Signature.Parameters {
			shape.Parameters = append(shape.Parameters, ParameterShape{
				Name:     parameter.Name,
				Type:     parameter.Type,
				Variadic: parameter.Variadic,
			})
		}
		for _, result := range method.Signature.Results {
			shape.Results = append(shape.Results, ResultShape{Name: result.Name, Type: result.Type})
		}
		shapes = append(shapes, shape)
	}
	return shapes
}

func addImplementCandidates(result *Result, snapshot *Snapshot, request ImplementRequest, paths contractPaths, plan contractplan.Plan) error {
	selectedDisplay := displayPath(snapshot.Root, paths.selected)
	selectedRevision := Digest(snapshot.Contents[paths.selected])
	if paths.selected == paths.destination && len(result.Changes) > 0 {
		selectedRevision = changedRevision(result.Changes, selectedDisplay)
		if selectedRevision == "" {
			return fmt.Errorf("changed concrete type file %q lacks final revision evidence", selectedDisplay)
		}
	}
	result.ReceiptCandidates = []ContinuationCandidate{{File: selectedDisplay, Symbol: request.Symbol, ExpectedRevision: &selectedRevision}}
	if len(plan.Methods) == 0 {
		return nil
	}
	destinationDisplay := displayPath(snapshot.Root, paths.destination)
	destinationRevision := changedRevision(result.Changes, destinationDisplay)
	if destinationRevision == "" {
		return fmt.Errorf("generated method destination lacks planned revision evidence")
	}
	for _, method := range plan.Methods {
		qualified := request.Symbol + "." + method.Signature.Name
		if request.ReceiverKind == contractplan.ReceiverPointer {
			qualified = "(*" + request.Symbol + ")." + method.Signature.Name
		}
		candidate := ContinuationCandidate{File: destinationDisplay, Symbol: qualified, ExpectedRevision: &destinationRevision}
		result.ReceiptCandidates = append(result.ReceiptCandidates, candidate)
	}
	return nil
}

func changedRevision(changes []Revision, path string) string {
	for _, change := range changes {
		if change.Path == path {
			return change.Revision
		}
	}
	return ""
}

func displayPath(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return path
	}
	return filepath.ToSlash(relative)
}
