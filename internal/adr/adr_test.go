// WHY: Prove ADR schema validation, deterministic search paging, repository migration, and scoped index replacement.
package adr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixtureRecord = `# ADR-0001: Workspace Contract

* **Status**: Accepted
* **Date**: 2026-10-21

<!-- semedit-adr-metadata:v1 keys=summary,description,role,scope,area,languages; do not change key names or order. -->
| Key | Value |
| :--- | :--- |
| summary | Keep edits inside the selected workspace boundary. |
| description | Resolve request paths within the selected workspace. |
| role | invariant |
| scope | project |
| area | backend |
| languages | go, rust |

## Context

The workspace determines which files a semantic request may inspect or edit.

## Decision

Resolve request paths relative to the workspace.

## Invariants

* Requests do not escape the selected root.

## Consequences

Returned paths do not expose machine-local usernames.
`

func TestParseRequiresFixedMetadataAndSections(t *testing.T) {
	t.Parallel()

	record, err := Parse(fixtureRecord)
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != "0001" || record.Title != "Workspace Contract" || record.Status != "Accepted" || record.Date != "2026-10-21" {
		t.Fatalf("record identity = %+v", record)
	}
	if record.Summary == "" || record.Description == "" || record.Role != "invariant" || record.Scope != "project" || record.Area != "backend" || strings.Join(record.Languages, ",") != "go,rust" {
		t.Fatalf("record metadata = %+v", record)
	}
	if !strings.Contains(record.Invariants, "selected root") || !strings.Contains(record.Consequences, "usernames") {
		t.Fatalf("record decision content = %+v", record)
	}

	titleInFence := strings.Replace(fixtureRecord, "# ADR-0001: Workspace Contract", "~~~markdown\n# ADR-0001: Workspace Contract\n~~~", 1)
	metadataStart := strings.Index(fixtureRecord, metadataComment)
	metadataEnd := strings.Index(fixtureRecord, "## Context")
	fencedMetadata := fixtureRecord[:metadataStart] + "~~~markdown\n" + fixtureRecord[metadataStart:metadataEnd] + "~~~\n" + fixtureRecord[metadataEnd:]
	cases := []struct {
		name   string
		source string
	}{
		{name: "changed metadata key order", source: strings.Replace(fixtureRecord, "keys=summary,description,role,scope,area,languages", "keys=summary,description,role,scope,languages,area", 1)},
		{name: "missing metadata row", source: strings.Replace(fixtureRecord, "| area | backend |\n", "", 1)},
		{name: "appended duplicate metadata row", source: strings.Replace(fixtureRecord, "| languages | go, rust |\n", "| languages | go, rust |\n| role | principle |\n", 1)},
		{name: "duplicate canonical status", source: strings.Replace(fixtureRecord, "* **Status**: Accepted", "* **Status**: Accepted\n* **Status**: Proposed", 1)},
		{name: "title only in fenced example", source: titleInFence},
		{name: "metadata only in fenced example", source: fencedMetadata},
		{name: "duplicate section", source: fixtureRecord + "\n## Context\nDuplicate section.\n"},
		{name: "fenced heading cannot satisfy required section", source: strings.Replace(fixtureRecord, "## Invariants\n\n* Requests do not escape the selected root.\n", "", 1) + "\n\x60\x60\x60markdown\n## Invariants\nFake section.\n\x60\x60\x60\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, parseErr := Parse(tc.source); parseErr == nil {
				t.Fatal("malformed ADR was accepted")
			}
		})
	}

	withFencedHeaderMetadata, err := Parse(strings.Replace(fixtureRecord, "* **Status**: Accepted", "~~~text\n* **Status**: Proposed\n* **Date**: 2027-01-01\n~~~\n* **Status**: Accepted", 1))
	if err != nil {
		t.Fatal(err)
	}
	if withFencedHeaderMetadata.Status != "Accepted" || withFencedHeaderMetadata.Date != "2026-10-21" {
		t.Fatalf("fenced example replaced canonical header metadata: %+v", withFencedHeaderMetadata)
	}

	withBodyMetadata, err := Parse(fixtureRecord + "\nStatus: Proposed\nDate: 2027-01-01\n")
	if err != nil {
		t.Fatal(err)
	}
	if withBodyMetadata.Status != "Accepted" || withBodyMetadata.Date != "2026-10-21" {
		t.Fatalf("body metadata replaced the header values: %+v", withBodyMetadata)
	}

	fencedInvariant, err := Parse(strings.Replace(fixtureRecord, "* Requests do not escape the selected root.", "~~~text\nInvariantOnlyInsideFence\n~~~", 1))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fencedInvariant.Invariants, "InvariantOnlyInsideFence") {
		t.Fatalf("fenced section content was dropped: %+v", fencedInvariant)
	}

	withFencedHeadings, err := Parse(strings.Replace(fixtureRecord, "The workspace determines which files", "The workspace determines which files\n\n~~~markdown\n## Invariants\nFake section\n## Consequences\nFake section\n~~~\n", 1))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(withFencedHeadings.Invariants, "Fake section") || strings.Contains(withFencedHeadings.Consequences, "Fake section") {
		t.Fatalf("fenced heading was parsed as a section: %+v", withFencedHeadings)
	}
}

func TestSearchRanksFiltersAndPagesDeterministically(t *testing.T) {
	t.Parallel()
	first, err := Parse(fixtureRecord)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Parse(strings.ReplaceAll(fixtureRecord, "0001", "0002"))
	if err != nil {
		t.Fatal(err)
	}
	second.Title = "Workspace Contract Extended"
	second.Path = "docs/adr/0002-workspace-contract-extended.md"
	second.Invariants = "Needlequasar remains true."
	first.Path = "docs/adr/0001-workspace-contract.md"
	page, err := Search([]Record{second, first}, SearchOptions{Query: "workspace boundary", Languages: []string{"rust"}, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Items) != 1 || page.Items[0].ID != "ADR-0001" || page.Items[0].Path != "docs/adr/0001-workspace-contract.md" {
		t.Fatalf("first page = %+v", page)
	}
	page, err = Search([]Record{first, second}, SearchOptions{Query: "workspace boundary", Languages: []string{"go"}, Offset: 1, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Items) != 1 || page.Items[0].ID != "ADR-0002" {
		t.Fatalf("second page = %+v", page)
	}
	page, err = Search([]Record{first, second}, SearchOptions{Query: "Needlequasar"})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != "ADR-0002" {
		t.Fatalf("invariant-only search = %+v", page)
	}
	page, err = Search([]Record{first, second}, SearchOptions{Query: "invariant"})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 {
		t.Fatalf("taxonomy search total = %d, want 2", page.Total)
	}
	if _, err := Search(nil, SearchOptions{Limit: 101}); err == nil {
		t.Fatal("oversized page was accepted")
	}
}

func TestSyncIndexReplacesOnlyMarkedRegister(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	adrDir := filepath.Join(root, "docs", "adr")
	if err := os.MkdirAll(adrDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".adr-dir"), []byte("docs/adr\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(adrDir, "README.md"), []byte("# ADR Index\n\nAuthored intro.\n\n"+registerStart+"\nold\n"+registerEnd+"\n\nAuthored footer.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(adrDir, "0001-workspace-contract.md"), []byte(fixtureRecord), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SyncIndex(root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(adrDir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Authored intro.", "Authored footer.", "[ADR-0001](0001-workspace-contract.md)", "Keep edits inside the selected workspace boundary.", "| invariant | project | backend | go, rust |"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("generated index missing %q", want)
		}
	}
	if strings.Contains(string(data), "old generated content") {
		t.Fatal("generated register was not replaced")
	}
}

func TestReadAllRepositoryRecordsAreMigrated(t *testing.T) {
	t.Parallel()
	root := filepath.Clean(filepath.Join("..", ".."))
	records, err := ReadAll(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) == 0 {
		t.Fatal("repository contains no valid ADR records")
	}
	if !strings.HasPrefix(records[0].Path, "docs/adr/") {
		t.Fatalf("ADR path is not workspace-relative: %q", records[0].Path)
	}
	indexPath := filepath.Join(root, filepath.Dir(records[0].Path), "README.md")
	index, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	want, err := replaceRegister(string(index), records)
	if err != nil {
		t.Fatal(err)
	}
	if want != string(index) {
		t.Fatal("ADR register is stale; regenerate it with semedit generate-adr-index")
	}
}
