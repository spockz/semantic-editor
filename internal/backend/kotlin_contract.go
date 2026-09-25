// Package backend keeps Kotlin server settings neutral at the ingress boundary.
package backend

// KotlinConfig identifies an explicitly preinstalled Kotlin language server.
type KotlinConfig struct {
	KotlinBin string `json:"kotlin_bin,omitempty"`
}
