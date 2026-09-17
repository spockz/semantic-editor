//go:build windows

package mcp

import (
	"errors"
)

// ErrReloadUnsupported indicates in-place exec is unsupported on Windows.
var ErrReloadUnsupported = errors.New("live reload via in-place exec is not supported on Windows")

// execReload falls back on Windows where execve in-place is unsupported.
func execReload() error {
	return ErrReloadUnsupported
}
