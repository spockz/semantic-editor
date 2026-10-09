// Package packageclosure groups files from the most split-out state by coupling.
package packageclosure

import (
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

type fileCluster struct {
	members []int
	key     string
}

func candidateGroups(files []fileInfo, weights map[pair]float64, minGroupFiles, limit int) []*candidate {
	if len(files) < 2*minGroupFiles || limit < 1 {
		return nil
	}
	totalWeight := 0.0
	for _, weight := range weights {
		totalWeight += weight
	}
	if totalWeight == 0 {
		return nil
	}

	seenPartitions := make(map[string]struct{})
	var candidates []*candidate
	addCandidate := func(group []int) {
		if len(group) < minGroupFiles || len(files)-len(group) < minGroupFiles {
			return
		}
		key := partitionKey(group, len(files))
		if _, exists := seenPartitions[key]; exists {
			return
		}
		seenPartitions[key] = struct{}{}
		_, _, crossing := partitionWeight(group, len(files), weights)
		other := complement(group, len(files))
		candidates = append(candidates, &candidate{
			group:      slices.Clone(group),
			cut:        crossing / totalWeight,
			groupNames: names(files, group),
			otherNames: names(files, other),
		})
	}

	clusters := initialClusters(files)
	for _, cluster := range clusters {
		addCandidate(cluster.members)
	}
	for len(clusters) > 1 {
		cross, degree := clusterCoupling(clusters, len(files), weights)
		left, right := strongestClusterPair(clusters, cross, degree)
		if left < 0 {
			break
		}
		members := append(slices.Clone(clusters[left].members), clusters[right].members...)
		sort.Ints(members)
		merged := fileCluster{members: members, key: groupKey(members)}
		next := make([]fileCluster, 0, len(clusters)-1)
		for index, cluster := range clusters {
			if index != left && index != right {
				next = append(next, cluster)
			}
		}
		next = append(next, merged)
		sort.Slice(next, func(i, j int) bool { return next[i].key < next[j].key })
		clusters = next
		addCandidate(merged.members)
	}

	sortCandidates(candidates)
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

func initialClusters(files []fileInfo) []fileCluster {
	indexByPath := make(map[string]int, len(files))
	for index, file := range files {
		indexByPath[filepath.Clean(file.path)] = index
	}
	paired := make([]bool, len(files))
	var clusters []fileCluster
	for index, file := range files {
		if paired[index] {
			continue
		}
		members := []int{index}
		if file.test {
			sourcePath := strings.TrimSuffix(filepath.Clean(file.path), "_test.go") + ".go"
			if sourceIndex, ok := indexByPath[sourcePath]; ok && !paired[sourceIndex] {
				members = append(members, sourceIndex)
				paired[sourceIndex] = true
			}
		} else {
			testPath := strings.TrimSuffix(filepath.Clean(file.path), ".go") + "_test.go"
			if testIndex, ok := indexByPath[testPath]; ok && !paired[testIndex] {
				members = append(members, testIndex)
				paired[testIndex] = true
			}
		}
		paired[index] = true
		sort.Ints(members)
		clusters = append(clusters, fileCluster{members: members, key: groupKey(members)})
	}
	sort.Slice(clusters, func(i, j int) bool { return clusters[i].key < clusters[j].key })
	return clusters
}

func clusterCoupling(clusters []fileCluster, fileCount int, weights map[pair]float64) ([][]float64, []float64) {
	owner := make([]int, fileCount)
	for clusterIndex, cluster := range clusters {
		for _, member := range cluster.members {
			owner[member] = clusterIndex
		}
	}
	cross := make([][]float64, len(clusters))
	for index := range cross {
		cross[index] = make([]float64, len(clusters))
	}
	degree := make([]float64, len(clusters))
	for edge, weight := range weights {
		left, right := owner[edge.left], owner[edge.right]
		if left == right {
			continue
		}
		cross[left][right] += weight
		cross[right][left] += weight
		degree[left] += weight
		degree[right] += weight
	}
	return cross, degree
}

func strongestClusterPair(clusters []fileCluster, cross [][]float64, degree []float64) (int, int) {
	bestLeft, bestRight, bestScore := -1, -1, -1.0
	for left := range clusters {
		for right := left + 1; right < len(clusters); right++ {
			shared := cross[left][right]
			if shared <= 0 {
				continue
			}
			score := shared / (degree[left] + degree[right] - shared)
			if score > bestScore+1e-12 || (math.Abs(score-bestScore) <= 1e-12 && clusterPairKey(clusters[left], clusters[right]) < clusterPairKey(clusters[bestLeft], clusters[bestRight])) {
				bestLeft, bestRight, bestScore = left, right, score
			}
		}
	}
	return bestLeft, bestRight
}

func clusterPairKey(left, right fileCluster) string {
	if left.key < right.key {
		return left.key + "|" + right.key
	}
	return right.key + "|" + left.key
}

func partitionWeight(group []int, fileCount int, weights map[pair]float64) (insideGroup, insideOther, crossing float64) {
	belongs := make([]bool, fileCount)
	for _, index := range group {
		belongs[index] = true
	}
	for edge, weight := range weights {
		left, right := belongs[edge.left], belongs[edge.right]
		switch {
		case left && right:
			insideGroup += weight
		case !left && !right:
			insideOther += weight
		default:
			crossing += weight
		}
	}
	return insideGroup, insideOther, crossing
}

func complement(group []int, count int) []int {
	selected := make([]bool, count)
	for _, index := range group {
		selected[index] = true
	}
	var result []int
	for i := range count {
		if !selected[i] {
			result = append(result, i)
		}
	}
	return result
}

func groupKey(group []int) string {
	parts := make([]string, len(group))
	for i, value := range group {
		parts[i] = fmt.Sprint(value)
	}
	return strings.Join(parts, ",")
}

func partitionKey(group []int, count int) string {
	other := complement(group, count)
	left, right := groupKey(group), groupKey(other)
	if left < right {
		return left + "|" + right
	}
	return right + "|" + left
}

func sortCandidates(candidates []*candidate) {
	sort.Slice(candidates, func(i, j int) bool {
		if math.Abs(candidates[i].cut-candidates[j].cut) > 1e-9 {
			return candidates[i].cut < candidates[j].cut
		}
		return candidates[i].groupNames < candidates[j].groupNames
	})
}

func names(files []fileInfo, group []int) string {
	values := make([]string, len(group))
	for i, index := range group {
		file := files[index]
		values[i] = fmt.Sprintf("%s(imports=%d,exports=%d)", filepath.Base(file.path), len(file.imports), file.exportedDecl)
	}
	return strings.Join(values, ", ")
}
