package main

import (
	"golang.org/x/tools/txtar"
)

// Archive represents a collection of files in txtar format.
type Archive = txtar.Archive

// ArchiveFile represents a single file entry in a txtar archive.
type ArchiveFile = txtar.File

// ParseArchive parses a txtar archive into memory.
func ParseArchive(data []byte) *Archive {
	return txtar.Parse(data)
}
