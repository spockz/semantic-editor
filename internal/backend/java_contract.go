// Package backend defines the neutral service boundary and keeps Java request configuration with project context.
package backend

// JavaConfig contains only explicit external-tool paths. An empty JavaBin uses
// the user's PATH; JDTLSHome is always required and is never auto-discovered.
type JavaConfig struct {
	JDTLSHome string `json:"jdtls_home,omitempty"`
	JavaBin   string `json:"java_bin,omitempty"`
	// ImportMaven enables JDT LS Maven project import after explicit trust.
	// Gradle import is never enabled by this option.
	ImportMaven bool `json:"import_maven,omitempty"`
}
