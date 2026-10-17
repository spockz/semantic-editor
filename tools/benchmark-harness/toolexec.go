// Package main uses Go's toolexec hook to refresh linker output inodes on macOS tmpfs.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spockz/semantic-editor/internal/atomicfile"
)

func init() {
	if len(os.Args) > 1 && isGoToolInvocation(os.Args[1]) {
		os.Exit(runFixtureToolexec(os.Args[1:]))
	}
}

func runFixtureToolexec(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "fixture toolexec: missing Go tool path")
		return 2
	}
	command := exec.CommandContext(context.Background(), args[0], args[1:]...) // #nosec G204 G702 -- Go's toolexec contract supplies the selected Go tool and its arguments.
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			if code := exitErr.ExitCode(); code >= 0 {
				return code
			}
		}
		fmt.Fprintf(os.Stderr, "fixture toolexec: run %s: %v\n", args[0], err)
		return 1
	}
	if filepath.Base(args[0]) != "link" {
		return 0
	}
	output := linkerOutputPath(args[1:])
	if output == "" {
		return 0
	}
	absoluteOutput, err := filepath.Abs(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fixture toolexec: resolve linker output %q: %v\n", output, err)
		return 1
	}
	outputDir, err := filepath.EvalSymlinks(filepath.Dir(absoluteOutput))
	if err != nil {
		fmt.Fprintf(os.Stderr, "fixture toolexec: resolve linker output directory %q: %v\n", filepath.Dir(absoluteOutput), err)
		return 1
	}
	tmpfs, err := isTmpfsMount(outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fixture toolexec: inspect linker output filesystem %q: %v\n", outputDir, err)
		return 1
	}
	if !tmpfs {
		return 0
	}
	// The successful Go linker's -o argument is the owned output path; require a regular executable before refreshing it.
	info, err := os.Lstat(absoluteOutput) // #nosec G703 -- path is the successful Go linker's own -o output, confined to tmpfs and checked below.
	if err != nil {
		fmt.Fprintf(os.Stderr, "fixture toolexec: stat linker output %q: %v\n", absoluteOutput, err)
		return 1
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return 0
	}
	contents, err := os.ReadFile(absoluteOutput) // #nosec G304 G703 -- same linker output validated as a regular executable on tmpfs above.
	if err != nil {
		fmt.Fprintf(os.Stderr, "fixture toolexec: read linker output %q: %v\n", absoluteOutput, err)
		return 1
	}
	if err := atomicfile.WriteAtomic(absoluteOutput, contents); err != nil {
		fmt.Fprintf(os.Stderr, "fixture toolexec: refresh linker output %q: %v\n", absoluteOutput, err)
		return 1
	}
	return 0
}

func linkerOutputPath(args []string) string {
	for index, arg := range args {
		if arg == "-o" && index+1 < len(args) {
			return args[index+1]
		}
		if output, found := strings.CutPrefix(arg, "-o="); found {
			return output
		}
	}
	return ""
}

func isGoToolInvocation(path string) bool {
	if !filepath.IsAbs(path) || filepath.Base(filepath.Dir(filepath.Dir(path))) != "tool" {
		return false
	}
	if filepath.Base(filepath.Dir(path)) != runtime.GOOS+"_"+runtime.GOARCH {
		return false
	}
	switch filepath.Base(path) {
	case "asm", "buildid", "cgo", "compile", "cover", "link", "pack", "preprofile", "vet":
	default:
		return false
	}
	// Go supplies an absolute host-tool path; only dispatch an existing executable from pkg/tool/<host>.
	info, err := os.Stat(path) // #nosec G703 -- this predicate accepts only recognized Go tool paths before execution.
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0
}

func fixtureToolexecGOFlags(executable string) (string, error) {
	if strings.ContainsAny(executable, "\t\n\r") {
		return "", fmt.Errorf("go toolexec path %q contains unsupported whitespace", executable)
	}
	if strings.Contains(executable, " ") {
		if strings.ContainsAny(executable, "'\"") {
			return "", fmt.Errorf("go toolexec path %q contains quotes and spaces that Go cannot encode in GOFLAGS", executable)
		}
		return fmt.Sprintf(`'-toolexec="%s"'`, executable), nil
	}
	return "-toolexec=" + executable, nil
}
