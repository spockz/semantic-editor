// Package packageclosure tests coupling scores and candidate group enumeration.
package packageclosure

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCandidateGroupsFindsWeakCutBetweenCohesiveClusters(t *testing.T) {
	files := []fileInfo{
		{path: "a.go"},
		{path: "b.go"},
		{path: "c.go"},
		{path: "d.go"},
	}
	weights := map[pair]float64{
		{left: 0, right: 1}: 4,
		{left: 2, right: 3}: 4,
		{left: 0, right: 2}: 0.05,
		{left: 0, right: 3}: 0.05,
		{left: 1, right: 2}: 0.05,
		{left: 1, right: 3}: 0.05,
	}

	candidates := candidateGroups(files, weights, 2, 12)
	for _, candidate := range candidates {
		if candidate.groupNames != "a.go(imports=0,exports=0), b.go(imports=0,exports=0)" {
			continue
		}
		if got, want := candidate.cut, 0.02439; got < want-0.001 || got > want+0.001 {
			t.Fatalf("candidate cut = %.3f, want %.3f", got, want)
		}
		return
	}
	t.Fatalf("candidateGroups() did not return the cohesive cluster split: %#v", candidates)
}

func TestCandidateGroupsAllowsSingleSourceFilePackage(t *testing.T) {
	files := []fileInfo{
		{path: "presentation.go"},
		{path: "server.go"},
		{path: "batch.go"},
		{path: "buffers.go"},
	}
	weights := map[pair]float64{
		{left: 0, right: 1}: 1,
		{left: 0, right: 2}: 1,
		{left: 0, right: 3}: 1,
		{left: 1, right: 2}: 4,
		{left: 1, right: 3}: 2,
		{left: 2, right: 3}: 2,
	}

	candidates := candidateGroups(files, weights, 1, 12)
	for _, candidate := range candidates {
		if candidate.groupNames == "presentation.go(imports=0,exports=0)" {
			return
		}
	}
	t.Fatalf("candidateGroups() did not retain a one-file package candidate: %#v", candidates)
}

func TestCandidateGroupsGrowsCohesiveClustersBottomUp(t *testing.T) {
	files := []fileInfo{
		{path: "a.go"},
		{path: "b.go"},
		{path: "c.go"},
		{path: "d.go"},
		{path: "e.go"},
		{path: "f.go"},
	}
	weights := make(map[pair]float64)
	for left := range files {
		for right := left + 1; right < len(files); right++ {
			weights[orderedPair(left, right)] = 0.05
		}
	}
	for _, group := range [][]int{{0, 1}, {2, 3}, {4, 5}} {
		weights[orderedPair(group[0], group[1])] = 4
	}

	candidates := candidateGroups(files, weights, 2, 12)
	for _, wanted := range []string{
		"a.go(imports=0,exports=0), b.go(imports=0,exports=0)",
		"c.go(imports=0,exports=0), d.go(imports=0,exports=0)",
		"e.go(imports=0,exports=0), f.go(imports=0,exports=0)",
	} {
		found := false
		for _, candidate := range candidates {
			if candidate.groupNames == wanted {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("candidateGroups() omitted cohesive cluster %q: %#v", wanted, candidates)
		}
	}
}

func TestInitialClustersKeepSiblingSourceAndTestTogether(t *testing.T) {
	files := []fileInfo{
		{path: "presentation.go"},
		{path: "server.go"},
		{path: "presentation_test.go", test: true},
	}
	clusters := initialClusters(files)
	if len(clusters) != 2 {
		t.Fatalf("initialClusters() returned %d clusters, want 2: %#v", len(clusters), clusters)
	}
	for _, cluster := range clusters {
		if len(cluster.members) == 2 && cluster.members[0] == 0 && cluster.members[1] == 2 {
			return
		}
	}
	t.Fatalf("initialClusters() did not pair presentation source and test: %#v", clusters)
}

func TestCrossingTestFileCountIsSeparateBoundaryEvidence(t *testing.T) {
	testSpans := [][]int{{0, 1}, {1, 2}}
	if got := crossingTestCount([]int{0}, 3, testSpans); got != 1 {
		t.Fatalf("crossingTestCount() = %d, want 1", got)
	}
	if got := crossingTestCount([]int{0, 1}, 3, testSpans); got != 1 {
		t.Fatalf("crossingTestCount() for a two-file group = %d, want 1", got)
	}
}

func TestNamedTestCoverageIdentifiesFileOwnedTests(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"presentation.go", "presentation_test.go", "shared_test.go"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("package testdata\n"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	fileByName := map[string]int{filepath.Join(directory, "presentation.go"): 0}
	coverage, err := namedTestCoverage(directory, fileByName)
	if err != nil {
		t.Fatalf("namedTestCoverage() error = %v", err)
	}
	if coverage[0] != 1 {
		t.Fatalf("presentation.go dedicated test count = %d, want 1", coverage[0])
	}
}

func TestHistorySplitCochangeUsesOnlyGitReportedNames(t *testing.T) {
	history := history{commits: [][]string{{"0"}, {"0", "2"}, {"2"}, {"0"}}}
	got, count := history.splitCochange([]int{0, 1})
	if count != 4 {
		t.Fatalf("history sample count = %d, want 4", count)
	}
	if got != 0.25 {
		t.Fatalf("split co-change = %.2f, want 0.25", got)
	}
}
