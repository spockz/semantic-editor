// Package packageclosure analyzes source-file coupling and ranks candidate package boundaries.
package packageclosure

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

type Config struct {
	MinFiles             int  `json:"min-files"`
	IncludeTests         bool `json:"include-tests"`
	MinGroupFiles        int  `json:"min-group-files"`
	HistoryCommits       int  `json:"history-commits"`
	MaxHistoryCandidates int  `json:"max-history-candidates"`
	MaxReports           int  `json:"max-reports"`
	MinHistoryCommits    int  `json:"min-history-commits"`
}

type fileInfo struct {
	path         string
	file         *ast.File
	test         bool
	imports      map[string]struct{}
	exportedDecl int
}

type pair struct{ left, right int }

type candidate struct {
	group             []int
	cut               float64
	cochange          float64
	historyN          int
	groupTestCoverage int
	otherTestCoverage int
	crossingTestFiles int
	groupNames        string
	otherNames        string
}

// NewAnalyzer constructs the package-boundary analyzer with safe defaults.
func NewAnalyzer(config Config) *analysis.Analyzer {
	if config.MinFiles < 3 {
		config.MinFiles = 3
	}
	if config.MinGroupFiles < 1 {
		config.MinGroupFiles = 1
	}
	if config.HistoryCommits <= 0 {
		config.HistoryCommits = 300
	}
	if config.MaxHistoryCandidates <= 0 {
		config.MaxHistoryCandidates = 512
	}
	if config.MaxReports <= 0 {
		config.MaxReports = 3
	}
	if config.MinHistoryCommits <= 0 {
		config.MinHistoryCommits = 4
	}
	return &analysis.Analyzer{
		Name: "packageclosure",
		Doc:  "ranks candidate package splits using compiler references and Git co-change history",
		Run: func(pass *analysis.Pass) (any, error) {
			return analyze(pass, config)
		},
	}
}

func analyze(pass *analysis.Pass, config Config) (any, error) {
	files, fileByName, tests := collectFiles(pass, config.IncludeTests)
	if len(files) < config.MinFiles {
		return nil, nil
	}
	weights, testCoverage, testSpans := couplingWeights(pass, files, fileByName, tests)
	candidates := candidateGroups(files, weights, config.MinGroupFiles, config.MaxHistoryCandidates)
	if len(candidates) == 0 {
		return nil, nil
	}
	namedCoverage, err := namedTestCoverage(filepath.Dir(files[0].path), fileByName)
	if err != nil {
		return nil, fmt.Errorf("read test ownership for package %s: %w", pass.Pkg.Path(), err)
	}
	for index, count := range namedCoverage {
		testCoverage[index] += count
	}
	root, err := gitRoot(filepath.Dir(files[0].path))
	if err != nil {
		return nil, fmt.Errorf("find Git root for package %s: %w", pass.Pkg.Path(), err)
	}
	history, err := readHistory(root, filepath.Dir(files[0].path), files, config.HistoryCommits)
	if err != nil {
		return nil, fmt.Errorf("read package history for %s: %w", pass.Pkg.Path(), err)
	}
	packageImports, packageExports := len(pass.Pkg.Imports()), 0
	for _, name := range pass.Pkg.Scope().Names() {
		if pass.Pkg.Scope().Lookup(name).Exported() {
			packageExports++
		}
	}
	var findings []*candidate
	for _, candidate := range candidates {
		candidate.cochange, candidate.historyN = history.splitCochange(candidate.group)
		candidate.groupTestCoverage, candidate.otherTestCoverage = testCoverageForSplit(candidate.group, len(files), testCoverage)
		candidate.crossingTestFiles = crossingTestCount(candidate.group, len(files), testSpans)
		if candidate.historyN < config.MinHistoryCommits || candidate.cochange > 0.25 {
			continue
		}
		findings = append(findings, candidate)
	}
	sort.Slice(findings, func(i, j int) bool {
		iTestSeparation := findings[i].groupTestCoverage > 0 && findings[i].otherTestCoverage > 0
		jTestSeparation := findings[j].groupTestCoverage > 0 && findings[j].otherTestCoverage > 0
		if iTestSeparation != jTestSeparation {
			return iTestSeparation
		}
		if findings[i].cochange != findings[j].cochange {
			return findings[i].cochange < findings[j].cochange
		}
		if findings[i].cut != findings[j].cut {
			return findings[i].cut < findings[j].cut
		}
		return findings[i].groupNames < findings[j].groupNames
	})
	if len(findings) == 0 {
		return nil, nil
	}
	if len(findings) > config.MaxReports {
		findings = findings[:config.MaxReports]
	}
	seams := make([]string, len(findings))
	for i, finding := range findings {
		groupSize := len(finding.group)
		otherSize := len(files) - groupSize
		smallSide, smallTests := finding.groupNames, finding.groupTestCoverage
		otherTests := finding.otherTestCoverage
		if groupSize > otherSize {
			smallSide, smallTests, otherTests = finding.otherNames, finding.otherTestCoverage, finding.groupTestCoverage
		}
		seams[i] = fmt.Sprintf("split off {%s} (%d/%d files; cut %.2f, co-change %.2f over %d commits, dedicated test files %d/%d, cross-boundary test files %d)", smallSide, min(groupSize, otherSize), len(files), finding.cut, finding.cochange, finding.historyN, smallTests, otherTests, finding.crossingTestFiles)
	}
	pass.Report(analysis.Diagnostic{
		Pos:      pass.Files[0].Package,
		Message:  fmt.Sprintf("possible common-closure seams: %s; package imports=%d exports=%d", strings.Join(seams, "; "), packageImports, packageExports),
		Category: "packageclosure-seam",
	})
	return nil, nil
}

func collectFiles(pass *analysis.Pass, includeTests bool) ([]fileInfo, map[string]int, []*ast.File) {
	all := make([]*ast.File, 0, len(pass.Files))
	for _, file := range pass.Files {
		all = append(all, file)
	}
	sort.Slice(all, func(i, j int) bool {
		return pass.Fset.Position(all[i].Pos()).Filename < pass.Fset.Position(all[j].Pos()).Filename
	})
	files := make([]fileInfo, 0, len(all))
	fileByName := make(map[string]int, len(all))
	var tests []*ast.File
	for _, file := range all {
		path := filepath.Clean(pass.Fset.Position(file.Pos()).Filename)
		isTest := strings.HasSuffix(path, "_test.go")
		if isTest && !includeTests {
			tests = append(tests, file)
			continue
		}
		fileByName[path] = len(files)
		files = append(files, fileInfo{path: path, file: file, test: isTest, imports: importedPaths(file)})
	}
	for _, objectName := range pass.Pkg.Scope().Names() {
		object := pass.Pkg.Scope().Lookup(objectName)
		if !object.Exported() || object.Pos() == token.NoPos {
			continue
		}
		path := filepath.Clean(pass.Fset.Position(object.Pos()).Filename)
		if index, ok := fileByName[path]; ok {
			files[index].exportedDecl++
		}
	}
	return files, fileByName, tests
}

func importedPaths(file *ast.File) map[string]struct{} {
	paths := make(map[string]struct{}, len(file.Imports))
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err == nil {
			paths[path] = struct{}{}
		}
	}
	return paths
}

func couplingWeights(pass *analysis.Pass, files []fileInfo, fileByName map[string]int, tests []*ast.File) (map[pair]float64, map[int]int, [][]int) {
	weights := make(map[pair]float64)
	seenReferences := make(map[objectEdge]struct{})
	testUses := make(map[string]map[int]struct{})
	testCoverage := make(map[int]int)
	for _, file := range appendFiles(files, tests) {
		path := filepath.Clean(pass.Fset.Position(file.Pos()).Filename)
		index := fileByName[path]
		isTest := strings.HasSuffix(path, "_test.go")
		for _, ident := range identifiers(file) {
			object := pass.TypesInfo.Uses[ident]
			if object == nil || object.Pkg() != pass.Pkg || object.Pos() == token.NoPos {
				continue
			}
			definition := filepath.Clean(pass.Fset.Position(object.Pos()).Filename)
			definedIn, ok := fileByName[definition]
			if !ok || files[definedIn].test || (!isTest && definedIn == index) {
				continue
			}
			if isTest {
				if testUses[path] == nil {
					testUses[path] = make(map[int]struct{})
				}
				testUses[path][definedIn] = struct{}{}
				continue
			}
			edge := objectEdge{from: index, to: definedIn, object: object}
			if _, exists := seenReferences[edge]; !exists {
				seenReferences[edge] = struct{}{}
				weights[orderedPair(index, definedIn)] += 2
			}
		}
	}
	var testSpans [][]int
	for _, usedFiles := range testUses {
		if len(usedFiles) == 1 {
			for index := range usedFiles {
				testCoverage[index]++
			}
			continue
		}
		span := make([]int, 0, len(usedFiles))
		for index := range usedFiles {
			span = append(span, index)
		}
		sort.Ints(span)
		testSpans = append(testSpans, span)
	}
	for left := range files {
		for right := left + 1; right < len(files); right++ {
			shared := 0
			for imported := range files[left].imports {
				if _, ok := files[right].imports[imported]; ok {
					shared++
				}
			}
			if shared > 0 {
				weights[orderedPair(left, right)] += 0.05 * float64(shared)
			}
		}
	}
	return weights, testCoverage, testSpans
}

func crossingTestCount(group []int, fileCount int, testSpans [][]int) int {
	belongs := make([]bool, fileCount)
	for _, index := range group {
		belongs[index] = true
	}
	crossing := 0
	for _, span := range testSpans {
		inside, outside := false, false
		for _, index := range span {
			if belongs[index] {
				inside = true
			} else {
				outside = true
			}
		}
		if inside && outside {
			crossing++
		}
	}
	return crossing
}

func testCoverageForSplit(group []int, fileCount int, testCoverage map[int]int) (groupCoverage, otherCoverage int) {
	belongs := make([]bool, fileCount)
	for _, index := range group {
		belongs[index] = true
	}
	for index, count := range testCoverage {
		if belongs[index] {
			groupCoverage += count
		} else {
			otherCoverage += count
		}
	}
	return groupCoverage, otherCoverage
}

func namedTestCoverage(directory string, fileByName map[string]int) (map[int]int, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read directory %q: %w", directory, err)
	}
	coverage := make(map[int]int)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, "_test.go") {
			continue
		}
		sourceName := strings.TrimSuffix(name, "_test.go") + ".go"
		if index, exists := fileByName[filepath.Join(directory, sourceName)]; exists {
			coverage[index]++
		}
	}
	return coverage, nil
}

func appendFiles(files []fileInfo, tests []*ast.File) []*ast.File {
	result := make([]*ast.File, 0, len(files)+len(tests))
	for _, file := range files {
		result = append(result, file.file)
	}
	return append(result, tests...)
}

type objectEdge struct {
	from, to int
	object   types.Object
}

func identifiers(file *ast.File) []*ast.Ident {
	var result []*ast.Ident
	ast.Inspect(file, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok {
			result = append(result, ident)
		}
		return true
	})
	return result
}

func orderedPair(left, right int) pair {
	if left > right {
		left, right = right, left
	}
	return pair{left: left, right: right}
}
