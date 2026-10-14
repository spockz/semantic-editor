// Package main renders repository MCP configuration templates for local development installs.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	root := flag.String("workspace", ".", "repository root containing MCP configuration templates")
	flag.Parse()
	if len(flag.Args()) != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(2)
	}
	if err := install(filepath.Clean(*root)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
