// Package main provides the standalone vet driver for fatalorder.
package main

import (
	"os"

	"golang.org/x/tools/go/analysis/singlechecker"

	"semedit/tools/linters/fatalorder"
)

func main() {
	// Filter -json so unitchecker outputs plain text and exits with code 1 on violations.
	var filtered []string
	for _, arg := range os.Args {
		if arg == "-json" || arg == "--json" {
			continue
		}
		filtered = append(filtered, arg)
	}
	os.Args = filtered

	singlechecker.Main(fatalorder.Analyzer)
}
