// Package packageclosure measures file co-change using Git's own rename detection.
package packageclosure

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type history struct {
	commits [][]string
}

func gitRoot(dir string) (string, error) {
	output, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func readHistory(root, packageDir string, files []fileInfo, limit int) (history, error) {
	index := make(map[string]int, len(files))
	for i, file := range files {
		relative, err := filepath.Rel(root, file.path)
		if err != nil {
			return history{}, fmt.Errorf("relativize %q: %w", file.path, err)
		}
		index[filepath.ToSlash(relative)] = i
	}
	packagePath, err := filepath.Rel(root, packageDir)
	if err != nil {
		return history{}, fmt.Errorf("relativize package directory: %w", err)
	}
	output, err := exec.Command("git", "-C", root, "log", "--find-renames", "--format=%x1e%H", "--name-only", "-n", strconv.Itoa(limit), "--", filepath.ToSlash(packagePath)).Output()
	if err != nil {
		return history{}, fmt.Errorf("git log: %w", err)
	}
	var commits [][]string
	for _, record := range strings.Split(string(output), "\x1e") {
		lines := strings.Split(strings.TrimSpace(record), "\n")
		if len(lines) < 2 || lines[0] == "" {
			continue
		}
		changed := make(map[int]struct{})
		for _, name := range lines[1:] {
			if index, ok := index[filepath.ToSlash(strings.TrimSpace(name))]; ok {
				changed[index] = struct{}{}
			}
		}
		if len(changed) == 0 {
			continue
		}
		commit := make([]string, 0, len(changed))
		for index := range changed {
			commit = append(commit, strconv.Itoa(index))
		}
		commits = append(commits, commit)
	}
	return history{commits: commits}, nil
}

func (h history) splitCochange(group []int) (float64, int) {
	inGroup := make(map[string]struct{}, len(group))
	for _, index := range group {
		inGroup[strconv.Itoa(index)] = struct{}{}
	}
	union, both := 0, 0
	for _, commit := range h.commits {
		touchesGroup, touchesOther := false, false
		for _, value := range commit {
			if _, ok := inGroup[value]; ok {
				touchesGroup = true
			} else {
				touchesOther = true
			}
		}
		if touchesGroup || touchesOther {
			union++
			if touchesGroup && touchesOther {
				both++
			}
		}
	}
	if union == 0 {
		return 1, 0
	}
	return float64(both) / float64(union), union
}
