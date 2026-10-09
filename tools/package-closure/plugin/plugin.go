// Package main exposes the package-closure analyzer as a native golangci-lint Go plugin.
package main

import (
	"encoding/json"
	"fmt"

	"golang.org/x/tools/go/analysis"

	"semedit.tools/package-closure"
)

func New(settings any) ([]*analysis.Analyzer, error) {
	var config packageclosure.Config
	if settings != nil {
		encoded, err := json.Marshal(settings)
		if err != nil {
			return nil, fmt.Errorf("encode packageclosure settings: %w", err)
		}
		if err := json.Unmarshal(encoded, &config); err != nil {
			return nil, fmt.Errorf("decode packageclosure settings: %w", err)
		}
	}
	return []*analysis.Analyzer{packageclosure.NewAnalyzer(config)}, nil
}
