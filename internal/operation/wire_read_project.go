// WHY: This file owns shared read-project wire fields so lookup, inspect, outline, and reference schemas stay consistent.

package operation

var readProjectParams = []ParameterContract{
	{Name: "language", CLIName: "language", JSONName: "language", Description: "Language backend (default auto; Haskell requires standalone_haskell=true; Bash and Make require a selected source file)", Type: ParamString, Enums: languageEnums, LanguageSelector: true},
	{Name: wireTrustWorkspace, CLIName: wireCLITrustWorkspace, JSONName: wireTrustWorkspace, Description: "Explicitly trust this workspace for this request; required by trusted external-tool backends including Bash, Make, Kotlin, Java, Rust, Scala, and Haskell (default false)", Type: ParamBoolean, Default: false},
	{Name: wireJDTLSHome, CLIName: "jdtls-home", JSONName: wireJDTLSHome, Description: "Explicit preinstalled JDT LS distribution home required for Java lookup", Type: ParamString},
	{Name: "java_bin", CLIName: "java-bin", JSONName: "java_bin", Description: "Optional Java 21+ executable; defaults to java on PATH", Type: ParamString},
	{Name: wireImportMaven, CLIName: "import-maven", JSONName: wireImportMaven, Description: "Explicitly enable trusted JDT LS Maven import; never runs Maven", Type: ParamBoolean, Default: false},
	{Name: "metals_home", CLIName: "metals-home", JSONName: "metals_home", Description: "Explicit preinstalled pinned Metals distribution home required for Scala lookup", Type: ParamString},
	{Name: "metals_bin", CLIName: "metals-bin", JSONName: "metals_bin", Description: "Direct pinned Metals executable, alternative to metals_home", Type: ParamString},
	{Name: "java_version", CLIName: "java-version", JSONName: "java_version", Description: "Recorded Java major version required for Scala lookup", Type: ParamString},
	{Name: "standalone_haskell", CLIName: "haskell-standalone", JSONName: "standalone_haskell", Description: "Explicitly select standalone Haskell .hs lookup; rejects project markers", Type: ParamBoolean, Default: false},
	{Name: "ghc_bin", CLIName: "ghc-bin", JSONName: "ghc_bin", Description: "Preinstalled GHC executable for standalone Haskell lookup", Type: ParamString},
	{Name: "hls_bin", CLIName: "hls-bin", JSONName: "hls_bin", Description: "Preinstalled haskell-language-server-wrapper executable", Type: ParamString},
	{Name: "ghc_version", CLIName: "ghc-version", JSONName: "ghc_version", Description: "Recorded GHC version to require", Type: ParamString},
	{Name: "hls_version", CLIName: "hls-version", JSONName: "hls_version", Description: "Recorded HLS version to require", Type: ParamString},
	{Name: "kotlin_bin", CLIName: "kotlin-bin", JSONName: "kotlin_bin", Description: "Path to a preinstalled fwcd/kotlin-language-server executable; defaults to the executable on PATH", Type: ParamString},
	{Name: "bash_bin", CLIName: "bash-bin", JSONName: "bash_bin", Description: "Path to a preinstalled bash-language-server executable; defaults to PATH", Type: ParamString},
	{Name: "make_bin", CLIName: "make-bin", JSONName: "make_bin", Description: "Path to a preinstalled make-ls executable; defaults to PATH", Type: ParamString},
}

func readRequestParams(prefix ...ParameterContract) []ParameterContract {
	params := make([]ParameterContract, 0, len(prefix)+len(readProjectParams))
	params = append(params, prefix...)
	return append(params, readProjectParams...)
}
