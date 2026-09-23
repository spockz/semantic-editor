// Package backend retains standalone Haskell request settings without exposing the HLS implementation to ingress packages.
package backend

// HaskellConfig carries explicit toolchain settings in neutral project requests.
type HaskellConfig struct {
	Standalone bool   `json:"standalone,omitempty"`
	GHCBin     string `json:"ghc_bin,omitempty"`
	HLSBin     string `json:"hls_bin,omitempty"`
	GHCVersion string `json:"ghc_version,omitempty"`
	HLSVersion string `json:"hls_version,omitempty"`
}
