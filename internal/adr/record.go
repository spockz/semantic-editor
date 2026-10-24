// Package adr owns the validated ADR source schema used by indexing and search.
package adr

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const metadataComment = "<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->"

var metadataKeys = []string{"summary", "description", "role", "scope", "area", "languages"}

var (
	adrTitlePattern = regexp.MustCompile(`^# ADR-([0-9]{1,4}): (.+)$`)
	adrFilePattern  = regexp.MustCompile(`^[0-9]{4}-.+\.md$`)
	sectionPattern  = regexp.MustCompile(`^## ([A-Za-z][A-Za-z ]*)\s*$`)
	labelPattern    = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

// Record is one validated decision and its workspace-relative source path.
type Record struct {
	ID           string
	Title        string
	Status       string
	Date         string
	Summary      string
	Description  string
	Role         string
	Scope        string
	Area         string
	Languages    []string
	Path         string
	Context      string
	Decision     string
	Invariants   string
	Consequences string
}

// Directory reads the ADR directory selected by the repository's adr-tools marker.
func Directory(root string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve ADR workspace root: %w", err)
	}
	marker := filepath.Join(root, ".adr-dir")
	markerInfo, err := os.Lstat(marker)
	if err != nil {
		return "", fmt.Errorf("inspect ADR directory marker %s: %w", marker, err)
	}
	if !markerInfo.Mode().IsRegular() {
		return "", fmt.Errorf("ADR directory marker must be a regular file")
	}
	data, err := os.ReadFile(marker) // #nosec G304 -- marker is fixed directly under the resolved workspace root
	if err != nil {
		return "", fmt.Errorf("read ADR directory marker %s: %w", marker, err)
	}
	rel := strings.TrimSpace(string(data))
	if rel == "" || filepath.IsAbs(rel) || filepath.Clean(rel) != rel || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("ADR directory marker must contain a clean repository-relative path")
	}
	dir := filepath.Join(root, rel)
	current := root
	for component := range strings.SplitSeq(rel, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			return "", fmt.Errorf("inspect ADR directory component %s: %w", component, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("ADR directory component %s must not be a symlink", component)
		}
	}
	info, err := os.Lstat(dir) // #nosec G703 -- dir is constructed from the validated repository-relative marker
	if err != nil {
		return "", fmt.Errorf("inspect ADR directory %s: %w", rel, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("ADR directory %s must be a directory", rel)
	}
	return dir, nil
}

// ReadAll parses and validates every numbered ADR under root.
func ReadAll(root string) ([]Record, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve ADR workspace root: %w", err)
	}
	dir, err := Directory(root)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read ADR directory: %w", err)
	}
	base, err := filepath.Rel(root, dir)
	if err != nil {
		return nil, fmt.Errorf("make ADR directory relative to workspace: %w", err)
	}
	records := make([]Record, 0, len(entries))
	ids := make(map[string]string, len(entries))
	for _, entry := range entries {
		if !adrFilePattern.MatchString(entry.Name()) {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return nil, fmt.Errorf("ADR %s must be a regular file", entry.Name())
		}
		path := filepath.Join(dir, entry.Name())
		data, readErr := os.ReadFile(path) // #nosec G304 -- path is formed from the validated ADR directory and enumerated entry name
		if readErr != nil {
			return nil, fmt.Errorf("read ADR %s: %w", entry.Name(), readErr)
		}
		record, parseErr := Parse(string(data))
		if parseErr != nil {
			return nil, fmt.Errorf("ADR %s: %w", entry.Name(), parseErr)
		}
		if expected := strings.TrimSuffix(entry.Name(), ".md")[:4]; record.ID != expected {
			return nil, fmt.Errorf("ADR filename ID %s does not match title ID %s", expected, record.ID)
		}
		if previous, exists := ids[record.ID]; exists {
			return nil, fmt.Errorf("duplicate ADR ID %s in %s and %s", record.ID, previous, entry.Name())
		}
		ids[record.ID] = entry.Name()
		record.Path = filepath.ToSlash(filepath.Join(base, entry.Name()))
		records = append(records, record)
	}
	slices.SortFunc(records, func(a, b Record) int {
		ai, _ := strconv.Atoi(a.ID)
		bi, _ := strconv.Atoi(b.ID)
		return ai - bi
	})
	return records, nil
}

// Parse reads one ADR that follows the repository's fixed record schema.
func Parse(source string) (Record, error) {
	var record Record
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	preamble := canonicalPreamble(lines)
	for _, line := range preamble {
		if match := adrTitlePattern.FindStringSubmatch(line); match != nil {
			number, err := strconv.Atoi(match[1])
			if err != nil || number < 1 {
				return Record{}, fmt.Errorf("invalid ADR number %q", match[1])
			}
			record.ID, record.Title = fmt.Sprintf("%04d", number), strings.TrimSpace(match[2])
			break
		}
	}
	if record.ID == "" || record.Title == "" {
		return Record{}, fmt.Errorf("missing ADR title")
	}
	for _, line := range preamble {
		name, value, found := metadataLine(line)
		if !found {
			continue
		}
		switch name {
		case "status":
			if record.Status != "" {
				return Record{}, fmt.Errorf("duplicate status metadata")
			}
			record.Status = statusLabel(value)
		case "date":
			if record.Date != "" {
				return Record{}, fmt.Errorf("duplicate date metadata")
			}
			record.Date = value
		}
	}
	if record.Status == "" || record.Date == "" {
		return Record{}, fmt.Errorf("missing status or date")
	}
	if _, err := time.Parse("2006-01-02", record.Date); err != nil {
		return Record{}, fmt.Errorf("invalid date %q", record.Date)
	}
	if err := parseMetadata(preamble, &record); err != nil {
		return Record{}, err
	}
	sections, err := sectionBodies(lines)
	if err != nil {
		return Record{}, err
	}
	for _, required := range []string{"Context", "Decision", "Invariants", "Consequences"} {
		if strings.TrimSpace(sections[required]) == "" {
			return Record{}, fmt.Errorf("missing or empty %s section", required)
		}
	}
	record.Context = strings.TrimSpace(sections["Context"])
	record.Decision = strings.TrimSpace(sections["Decision"])
	record.Invariants = strings.TrimSpace(sections["Invariants"])
	record.Consequences = strings.TrimSpace(sections["Consequences"])
	return record, nil
}

func statusLabel(value string) string {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[0], "*.,;:")
}

func metadataLine(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	colon := strings.IndexByte(line, ':')
	if colon < 0 {
		return "", "", false
	}
	name := strings.ToLower(strings.Trim(strings.TrimSpace(line[:colon]), "* "))
	if name != "status" && name != "date" {
		return "", "", false
	}
	value := strings.Trim(strings.TrimSpace(line[colon+1:]), "* ")
	return name, value, true
}

func parseMetadata(lines []string, record *Record) error {
	metadataIndex := -1
	for i, line := range lines {
		if !strings.Contains(line, "semedit-adr-metadata:") {
			continue
		}
		if metadataIndex >= 0 {
			return fmt.Errorf("expected exactly one metadata schema comment")
		}
		metadataIndex = i
		if strings.TrimSpace(line) != metadataComment {
			return fmt.Errorf("metadata schema comment changed; required keys are %s", strings.Join(metadataKeys, ","))
		}
	}
	if metadataIndex < 0 {
		return fmt.Errorf("expected exactly one metadata schema comment")
	}
	values, err := parseMetadataTable(lines, metadataIndex)
	if err != nil {
		return err
	}
	record.Summary = values["summary"]
	record.Description = values["description"]
	record.Role = values["role"]
	record.Scope = values["scope"]
	record.Area = values["area"]
	record.Languages = parseLabels(values["languages"])
	return validateMetadataLabels(record)
}

func parseTableRow(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return "", "", false
	}
	parts := strings.Split(line[1:len(line)-1], "|")
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}

func tableRow(line, first, second string) bool {
	a, b, ok := parseTableRow(line)
	return ok && a == first && b == second
}

func separatorRow(line string) bool {
	a, b, ok := parseTableRow(line)
	return ok && strings.Trim(a, ":- ") == "" && strings.Trim(b, ":- ") == ""
}

func parseLabels(value string) []string {
	labels := make([]string, 0)
	for item := range strings.SplitSeq(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			labels = append(labels, item)
		}
	}
	return slices.Compact(labels)
}

func sectionBodies(lines []string) (map[string]string, error) {
	sections := make(map[string]string)
	current := ""
	var body strings.Builder
	fenceChar := byte(0)
	fenceLength := 0
	flush := func() {
		if current != "" {
			sections[current] = body.String()
			body.Reset()
		}
	}
	appendBody := func(line string) {
		if current != "" {
			body.WriteString(line)
			body.WriteByte('\n')
		}
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fenceChar != 0 {
			appendBody(line)
			if len(trimmed) >= fenceLength && strings.Trim(trimmed, string(fenceChar)) == "" {
				fenceChar, fenceLength = 0, 0
			}
			continue
		}
		if len(trimmed) >= 3 && (trimmed[0] == 96 || trimmed[0] == '~') {
			char := trimmed[0]
			n := 0
			for n < len(trimmed) && trimmed[n] == char {
				n++
			}
			if n >= 3 {
				appendBody(line)
				fenceChar, fenceLength = char, n
				continue
			}
		}
		if match := sectionPattern.FindStringSubmatch(line); match != nil {
			name := match[1]
			if _, exists := sections[name]; exists || name == current {
				return nil, fmt.Errorf("duplicate %s section", name)
			}
			flush()
			current = name
			continue
		}
		appendBody(line)
	}
	flush()
	return sections, nil
}

// ValidateFiles checks all numbered ADRs and reports malformed source records.
func ValidateFiles(root string) error {
	_, err := ReadAll(root)
	return err
}

func parseMetadataTable(lines []string, index int) (map[string]string, error) {
	if index+len(metadataKeys)+2 >= len(lines) {
		return nil, fmt.Errorf("metadata table is incomplete")
	}
	if !tableRow(lines[index+1], "Key", "Value") || !separatorRow(lines[index+2]) {
		return nil, fmt.Errorf("metadata table must use the fixed Key/Value header")
	}
	values := make(map[string]string, len(metadataKeys))
	for j, key := range metadataKeys {
		parsedKey, value, ok := parseTableRow(lines[index+3+j])
		if !ok || parsedKey != key || strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("metadata row %d must contain non-empty key %q", j+1, key)
		}
		values[key] = strings.TrimSpace(value)
	}
	tableEnd := index + 3 + len(metadataKeys)
	if tableEnd < len(lines) && strings.TrimSpace(lines[tableEnd]) != "" {
		return nil, fmt.Errorf("metadata table has unexpected or duplicate rows after %q", metadataKeys[len(metadataKeys)-1])
	}
	return values, nil
}

func validateMetadataLabels(record *Record) error {
	for _, label := range []string{record.Role, record.Scope, record.Area} {
		if !labelPattern.MatchString(label) {
			return fmt.Errorf("metadata role, scope, and area must be lowercase labels")
		}
	}
	if len(record.Languages) == 0 {
		return fmt.Errorf("metadata languages must contain at least one label")
	}
	for _, label := range record.Languages {
		if !labelPattern.MatchString(label) {
			return fmt.Errorf("metadata language %q must be a lowercase label", label)
		}
	}
	return nil
}

func canonicalPreamble(lines []string) []string {
	preamble := make([]string, len(lines))
	copy(preamble, lines)
	fenceChar := byte(0)
	fenceLength := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fenceChar != 0 {
			preamble[i] = ""
			if len(trimmed) >= fenceLength && strings.Trim(trimmed, string(fenceChar)) == "" {
				fenceChar, fenceLength = 0, 0
			}
			continue
		}
		if len(trimmed) >= 3 && (trimmed[0] == 96 || trimmed[0] == '~') {
			char := trimmed[0]
			n := 0
			for n < len(trimmed) && trimmed[n] == char {
				n++
			}
			if n >= 3 {
				preamble[i] = ""
				fenceChar, fenceLength = char, n
				continue
			}
		}
		if sectionPattern.MatchString(line) {
			for j := i; j < len(preamble); j++ {
				preamble[j] = ""
			}
			break
		}
	}
	return preamble
}
