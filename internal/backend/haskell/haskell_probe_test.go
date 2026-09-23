// Package haskell tests HLS/GHC version probes without launching real external tools.
package haskell

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeProbeTool(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil { // #nosec G302 -- fake probe must be executable.
		t.Fatal(err)
	}
}

func TestProbeHaskellToolchainRequiresMatchingHLSGHC(t *testing.T) {
	root := t.TempDir()
	ghc := filepath.Join(root, "ghc")
	hls := filepath.Join(root, "hls")
	writeProbeTool(t, ghc, `printf '9.6.5\n'`)
	writeProbeTool(t, hls, `if [ "$1" = "--probe-tools" ]; then printf 'haskell-language-server version: 2.10.0.0 (GHC: 9.4.8)\n'; fi`)
	_, err := probeHaskellToolchain(context.Background(), root, haskellRuntimeConfig{GHCBin: ghc, HLSBin: hls})
	if !errors.Is(err, ErrHaskellVersionMismatch) {
		t.Fatalf("mismatch error = %v", err)
	}
}

func TestProbeHaskellToolchainRequiresIdentifiableVersions(t *testing.T) {
	root := t.TempDir()
	ghc := filepath.Join(root, "ghc")
	hls := filepath.Join(root, "hls")
	writeProbeTool(t, ghc, `printf '9.6.5\n'`)
	writeProbeTool(t, hls, `printf 'unknown\n'`)
	_, err := probeHaskellToolchain(context.Background(), root, haskellRuntimeConfig{GHCBin: ghc, HLSBin: hls})
	if !errors.Is(err, ErrHaskellVersionUnknown) {
		t.Fatalf("unknown-version error = %v", err)
	}
}
