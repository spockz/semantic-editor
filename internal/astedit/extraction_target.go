// Package astedit shares extraction planning with failure-time target validation
// so owner alternatives obey the same selection and type constraints without publishing edits.
package astedit

import (
	"context"
	"go/format"
	"go/parser"
	"go/token"
)

// ValidateFunctionExtractionTarget checks the normal extraction plan without applying it.
func ValidateFunctionExtractionTarget(ctx context.Context, filePath, owner, helper string, opts ExtractOptions) error {
	input, candidate, err := prepareFunctionExtraction(ctx, filePath, owner, helper, opts)
	if err != nil {
		return err
	}
	_, err = prepareFunctionExtractionOutput(input, candidate)
	return err
}

func prepareFunctionExtraction(ctx context.Context, filePath, owner, helper string, opts ExtractOptions) (extractionInput, extractionCandidate, error) {
	if err := validateExtractRequest(helper, opts); err != nil {
		return extractionInput{}, extractionCandidate{}, err
	}
	input, err := loadExtractionInput(ctx, filePath, owner, opts)
	if err != nil {
		return extractionInput{}, extractionCandidate{}, err
	}
	candidate, err := planExtraction(input, owner, helper, opts)
	return input, candidate, err
}

// ValidateVariableExtractionTarget checks the native selection and typed argument constraints without applying an edit.
func ValidateVariableExtractionTarget(ctx context.Context, filePath, owner, name string, opts ExtractVariableOptions) error {
	input, err := prepareVariableExtraction(ctx, filePath, owner, name, opts)
	if err != nil {
		return err
	}
	updated, err := rewriteVariableArgument(input, name, opts.Discriminator)
	if err != nil {
		return err
	}
	_, err = prepareVariableExtractionOutput(input, owner, name, opts.Discriminator, updated)
	return err
}

func prepareVariableExtraction(ctx context.Context, filePath, owner, name string, opts ExtractVariableOptions) (variableExtractionInput, error) {
	if err := validateVariableExtractionRequest(name, opts); err != nil {
		return variableExtractionInput{}, err
	}
	input, err := loadVariableExtractionInput(ctx, filePath, owner, opts)
	if err != nil {
		return variableExtractionInput{}, err
	}
	if err := validateVariableExtractionTarget(input, owner, name, opts); err != nil {
		return variableExtractionInput{}, err
	}
	return input, nil
}

func prepareFunctionExtractionOutput(input extractionInput, candidate extractionCandidate) ([]byte, error) {
	formatted, err := format.Source(candidate.source)
	if err != nil {
		return nil, &SyntaxError{File: input.cleanPath, Snippet: candidate.declaration, Cause: err, Err: ErrSyntax}
	}
	formatted = normalizeExtractionNewlines(formatted)
	if _, err := parser.ParseFile(token.NewFileSet(), input.cleanPath, formatted, parser.ParseComments|parser.AllErrors); err != nil {
		return nil, &SyntaxError{File: input.cleanPath, Snippet: candidate.declaration, Cause: err, Err: ErrSyntax}
	}
	if err := validateExtractionCandidate(input.ctx, input.cleanPath, formatted); err != nil {
		return nil, err
	}
	return formatted, nil
}

func prepareVariableExtractionOutput(input variableExtractionInput, owner, name, discriminator string, updated []byte) ([]byte, error) {
	formatted, err := format.Source(updated)
	if err != nil {
		return nil, &SyntaxError{File: input.cleanPath, Snippet: discriminator, Cause: err, Err: ErrSyntax}
	}
	formatted = normalizeExtractionNewlines(formatted)
	if _, err := parser.ParseFile(token.NewFileSet(), input.cleanPath, formatted, parser.ParseComments|parser.AllErrors); err != nil {
		return nil, &SyntaxError{File: input.cleanPath, Snippet: discriminator, Cause: err, Err: ErrSyntax}
	}
	if err := validateVariableCandidate(input.ctx, input.cleanPath, owner, name, discriminator, input.candidate.path, input.content, formatted); err != nil {
		return nil, err
	}
	return formatted, nil
}
