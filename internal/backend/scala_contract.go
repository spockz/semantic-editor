// Package backend keeps Scala request settings neutral so ingress packages do not depend on the Metals implementation.
package backend

// ScalaConfig carries explicit external-tool paths and runtime metadata in backend requests.
type ScalaConfig struct {
	MetalsHome  string `json:"metals_home,omitempty"`
	MetalsBin   string `json:"metals_bin,omitempty"`
	JavaBin     string `json:"java_bin,omitempty"`
	JavaVersion string `json:"java_version,omitempty"`
}
