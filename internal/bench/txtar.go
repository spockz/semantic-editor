// Package bench implements the evaluation harness and multi-level correctness oracle for semedit benchmarks.
package bench

import (
	"bytes"
	"strings"
)

// Archive represents a collection of files in txtar format.
type Archive struct {
	Comment []byte
	Files   []ArchiveFile
}

// ArchiveFile represents a single file entry in a txtar archive.
type ArchiveFile struct {
	Name string
	Data []byte
}

var (
	newlineMarker = []byte("\n-- ")
	markerPrefix  = []byte("-- ")
	markerSuffix  = []byte(" --\n")
)

// ParseArchive parses a txtar archive into memory.
func ParseArchive(data []byte) *Archive {
	a := new(Archive)
	var name string
	data, name = findFile(data)
	a.Comment = data
	for name != "" {
		f := ArchiveFile{Name: name}
		data, name = findFile(data)
		f.Data = data
		a.Files = append(a.Files, f)
	}
	return a
}

func findFile(data []byte) (before []byte, name string) {
	var i int
	for {
		if bytes.HasPrefix(data[i:], markerPrefix) {
			if i == 0 || data[i-1] == '\n' {
				if m := bytes.Index(data[i:], markerSuffix); m >= 0 {
					before = data[:i]
					name = strings.TrimSpace(string(data[i+len(markerPrefix) : i+m]))
					data = data[i+m+len(markerSuffix):]
					return before, name
				}
			}
		}
		j := bytes.Index(data[i:], newlineMarker)
		if j < 0 {
			return data, ""
		}
		i += j + 1
	}
}
