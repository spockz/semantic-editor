// WHY: This test enforces strict 1:1 index atomicity, continuous 4-digit ADR numbering, and status integrity between ADR/research documents and their README.md indexes.

package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

var (
	adrRowRegex     = regexp.MustCompile(`^\|\s*\[ADR-(\d{4})\]\(([^)]+\.md)\)\s*\|\s*([^|]+)\|\s*(\*{0,2}[A-Za-z]+\*{0,2})\s*\|`)
	rqRowRegex      = regexp.MustCompile(`^\|\s*\[RQ-(\d{4})\]\(([^)]+\.md)\)\s*\|\s*([^|]+)\|\s*([^|]+)\|\s*(\*{0,2}[A-Za-z]+\*{0,2})\s*\|`)
	statusLineRegex = regexp.MustCompile(`(?mi)^(?:\*\s*)?\*?\*?Status\*?\*?:\s*\*?\*?([A-Za-z]+)`)
	adrFileRegex    = regexp.MustCompile(`^(\d{4})-[a-z0-9-]+\.md$`)
	rqFileRegex     = regexp.MustCompile(`^RQ-(\d{4})-[a-z0-9-]+\.md$`)
)

func TestADRIndexAtomicity(t *testing.T) {
	t.Parallel()

	rootDir := findDocgenRepoRoot(t)
	adrDir := filepath.Join(rootDir, "docs", "adr")
	readmePath := filepath.Join(adrDir, "README.md")

	readmeContent, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read %s: %v", readmePath, err)
	}

	tableEntries := parseADRTable(t, string(readmeContent))
	if len(tableEntries) == 0 {
		t.Fatal("no ADR table rows parsed from docs/adr/README.md")
	}

	dirEntries, err := os.ReadDir(adrDir)
	if err != nil {
		t.Fatalf("read dir %s: %v", adrDir, err)
	}

	diskFiles := make(map[string]int) // filename -> number
	for _, entry := range dirEntries {
		name := entry.Name()
		if entry.IsDir() || name == "README.md" || name == "ARCHIVED.md" {
			continue
		}
		matches := adrFileRegex.FindStringSubmatch(name)
		if matches == nil {
			t.Errorf("ADR file %s does not follow 4-digit zero-padded convention 'XXXX-<kebab-case>.md'", name)
			continue
		}
		num, _ := strconv.Atoi(matches[1])
		diskFiles[name] = num
	}

	// 1:1 correspondence check
	for filename, num := range diskFiles {
		entry, found := tableEntries[filename]
		if !found {
			t.Errorf("ADR file %s is present on disk but missing from docs/adr/README.md index table", filename)
			continue
		}
		if entry.number != num {
			t.Errorf("ADR file %s number %d does not match table number %d", filename, num, entry.number)
		}

		filePath := filepath.Join(adrDir, filename)
		fileContent, readErr := os.ReadFile(filePath)
		if readErr != nil {
			t.Errorf("read %s: %v", filePath, readErr)
			continue
		}

		fileStatus := extractDocStatus(string(fileContent))
		if fileStatus == "" {
			t.Errorf("ADR file %s is missing 'Status:' header", filename)
		} else if !strings.EqualFold(fileStatus, entry.status) {
			t.Errorf("ADR file %s status %q does not match index table status %q", filename, fileStatus, entry.status)
		}
	}

	for filename := range tableEntries {
		if _, found := diskFiles[filename]; !found {
			t.Errorf("docs/adr/README.md table references file %s which does not exist on disk", filename)
		}
	}

	// Numbering stays continuous across active and archived records.
	archivedNumbers := parseArchivedADRNumbers(t, filepath.Join(adrDir, "ARCHIVED.md"))
	assertADRSequence(t, diskFiles, archivedNumbers)
}

func TestResearchIndexAtomicity(t *testing.T) {
	t.Parallel()

	rootDir := findDocgenRepoRoot(t)
	rqDir := filepath.Join(rootDir, "docs", "research")
	readmePath := filepath.Join(rqDir, "README.md")

	readmeContent, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read %s: %v", readmePath, err)
	}

	tableEntries := parseRQTable(t, string(readmeContent))
	if len(tableEntries) == 0 {
		t.Fatal("no RQ table rows parsed from docs/research/README.md")
	}

	dirEntries, err := os.ReadDir(rqDir)
	if err != nil {
		t.Fatalf("read dir %s: %v", rqDir, err)
	}

	diskFiles := make(map[string]int)
	for _, entry := range dirEntries {
		name := entry.Name()
		if entry.IsDir() || name == "README.md" || !strings.HasSuffix(name, ".md") {
			continue
		}
		matches := rqFileRegex.FindStringSubmatch(name)
		if matches == nil {
			t.Errorf("research file %s does not follow 4-digit zero-padded convention 'RQ-XXXX-<kebab-case>.md'", name)
			continue
		}
		num, _ := strconv.Atoi(matches[1])
		diskFiles[name] = num
	}

	// 1:1 correspondence check
	for filename, num := range diskFiles {
		entry, found := tableEntries[filename]
		if !found {
			t.Errorf("research file %s is present on disk but missing from docs/research/README.md index table", filename)
			continue
		}
		if entry.number != num {
			t.Errorf("research file %s number %d does not match table number %d", filename, num, entry.number)
		}

		filePath := filepath.Join(rqDir, filename)
		fileContent, readErr := os.ReadFile(filePath)
		if readErr != nil {
			t.Errorf("read %s: %v", filePath, readErr)
			continue
		}

		fileStatus := extractDocStatus(string(fileContent))
		if fileStatus == "" {
			t.Errorf("research file %s is missing 'Status:' header", filename)
		} else if !strings.EqualFold(fileStatus, entry.status) {
			t.Errorf("research file %s status %q does not match index table status %q", filename, fileStatus, entry.status)
		}
	}

	for filename := range tableEntries {
		if _, found := diskFiles[filename]; !found {
			t.Errorf("docs/research/README.md table references file %s which does not exist on disk", filename)
		}
	}
}

type indexEntry struct {
	number int
	status string
}

func parseADRTable(t *testing.T, content string) map[string]indexEntry {
	t.Helper()
	entries := make(map[string]indexEntry)
	for line := range strings.SplitSeq(content, "\n") {
		match := adrRowRegex.FindStringSubmatch(strings.TrimSpace(line))
		if match == nil {
			continue
		}
		num, _ := strconv.Atoi(match[1])
		filename := match[2]
		status := cleanStatus(match[4])
		entries[filename] = indexEntry{number: num, status: status}
	}
	return entries
}

func parseRQTable(t *testing.T, content string) map[string]indexEntry {
	t.Helper()
	entries := make(map[string]indexEntry)
	for line := range strings.SplitSeq(content, "\n") {
		match := rqRowRegex.FindStringSubmatch(strings.TrimSpace(line))
		if match == nil {
			continue
		}
		num, _ := strconv.Atoi(match[1])
		filename := match[2]
		status := cleanStatus(match[5])
		entries[filename] = indexEntry{number: num, status: status}
	}
	return entries
}

func cleanStatus(raw string) string {
	return strings.Trim(raw, "* \t\r\n")
}

func extractDocStatus(content string) string {
	if match := statusLineRegex.FindStringSubmatch(content); match != nil {
		return match[1]
	}
	sectionRegex := regexp.MustCompile(`(?mi)^##\s*Status\s*\n+([A-Za-z]+)`)
	if match := sectionRegex.FindStringSubmatch(content); match != nil {
		return match[1]
	}
	return ""
}

func parseArchivedADRNumbers(t *testing.T, path string) []int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	archivedRegex := regexp.MustCompile(`\bADR-(\d{4})\b`)
	matches := archivedRegex.FindAllStringSubmatch(string(data), -1)
	var nums []int
	for _, m := range matches {
		num, _ := strconv.Atoi(m[1])
		nums = append(nums, num)
	}
	return nums
}

func assertADRSequence(t *testing.T, diskFiles map[string]int, archived []int) {
	t.Helper()
	allNums := make([]int, 0, len(diskFiles)+len(archived))
	for _, num := range diskFiles {
		allNums = append(allNums, num)
	}
	allNums = append(allNums, archived...)
	slices.Sort(allNums)

	if len(allNums) == 0 {
		t.Fatal("no ADR numbers found")
	}
	for i, num := range allNums {
		if num < 1 || num > 9999 {
			t.Errorf("ADR number %d is outside the 4-digit positive range", num)
		}
		if i > 0 && num == allNums[i-1] {
			t.Errorf("duplicate ADR number ADR-%04d exists in active or archived records", num)
			continue
		}
		if i > 0 && num != allNums[i-1]+1 {
			t.Errorf("ADR numbering gap between ADR-%04d and ADR-%04d", allNums[i-1], num)
		}
	}
}

func findDocgenRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repository root containing go.mod")
		}
		dir = parent
	}
}
