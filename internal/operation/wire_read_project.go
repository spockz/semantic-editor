// WHY: This file owns shared read-project wire fields so lookup, inspect, outline, and reference schemas stay consistent.

package operation

import "semedit/internal/backend"

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

func parseReadProject(raw map[string]any, file string) (backend.ProjectContext, error) {
	config, err := parseReadProjectConfiguration(raw)
	if err != nil {
		return backend.ProjectContext{}, err
	}
	return backend.ProjectContext{
		File:           file,
		Language:       config.language,
		WorkspaceTrust: backend.WorkspaceTrust{Trusted: config.trusted},
		Java:           config.java,
		Scala:          config.scala,
		Haskell:        config.haskell,
		Kotlin:         config.kotlin,
		Bash:           config.bash,
		Make:           config.make,
	}, nil
}

type readProjectConfiguration struct {
	language backend.LanguageID
	trusted  bool
	java     backend.JavaConfig
	scala    backend.ScalaConfig
	haskell  backend.HaskellConfig
	kotlin   backend.KotlinConfig
	bash     backend.BashConfig
	make     backend.MakeConfig
}

func parseReadProjectConfiguration(raw map[string]any) (readProjectConfiguration, error) {
	language, err := ParseEnum(raw, "language", "language", languageEnums, false, "")
	if err != nil {
		return readProjectConfiguration{}, err
	}
	trusted, err := ParseBool(raw, wireTrustWorkspace, wireCLITrustWorkspace, false)
	if err != nil {
		return readProjectConfiguration{}, err
	}
	javaScala, err := parseReadJavaScala(raw)
	if err != nil {
		return readProjectConfiguration{}, err
	}
	tools, err := parseReadProjectTools(raw)
	if err != nil {
		return readProjectConfiguration{}, err
	}
	return readProjectConfiguration{
		language: backend.LanguageID(language),
		trusted:  trusted,
		java:     javaScala.java,
		scala:    javaScala.scala,
		haskell:  tools.haskell,
		kotlin:   tools.kotlin,
		bash:     tools.bash,
		make:     tools.make,
	}, nil
}

func parseReadJavaScala(raw map[string]any) (readProjectConfiguration, error) {
	jdtlsHome, err := ParseString(raw, wireJDTLSHome, "jdtls-home", false)
	if err != nil {
		return readProjectConfiguration{}, err
	}
	javaBin, err := ParseString(raw, "java_bin", "java-bin", false)
	if err != nil {
		return readProjectConfiguration{}, err
	}
	importMaven, err := ParseBool(raw, wireImportMaven, "import-maven", false)
	if err != nil {
		return readProjectConfiguration{}, err
	}
	metalsHome, err := ParseString(raw, "metals_home", "metals-home", false)
	if err != nil {
		return readProjectConfiguration{}, err
	}
	metalsBin, err := ParseString(raw, "metals_bin", "metals-bin", false)
	if err != nil {
		return readProjectConfiguration{}, err
	}
	javaVersion, err := ParseString(raw, "java_version", "java-version", false)
	if err != nil {
		return readProjectConfiguration{}, err
	}
	return readProjectConfiguration{
		java:  backend.JavaConfig{JDTLSHome: jdtlsHome, JavaBin: javaBin, ImportMaven: importMaven},
		scala: backend.ScalaConfig{MetalsHome: metalsHome, MetalsBin: metalsBin, JavaBin: javaBin, JavaVersion: javaVersion},
	}, nil
}

func parseReadProjectTools(raw map[string]any) (readProjectConfiguration, error) {
	standalone, err := ParseBool(raw, "standalone_haskell", "haskell-standalone", false)
	if err != nil {
		return readProjectConfiguration{}, err
	}
	ghcBin, err := ParseString(raw, "ghc_bin", "ghc-bin", false)
	if err != nil {
		return readProjectConfiguration{}, err
	}
	hlsBin, err := ParseString(raw, "hls_bin", "hls-bin", false)
	if err != nil {
		return readProjectConfiguration{}, err
	}
	ghcVersion, err := ParseString(raw, "ghc_version", "ghc-version", false)
	if err != nil {
		return readProjectConfiguration{}, err
	}
	hlsVersion, err := ParseString(raw, "hls_version", "hls-version", false)
	if err != nil {
		return readProjectConfiguration{}, err
	}
	kotlinBin, err := ParseString(raw, "kotlin_bin", "kotlin-bin", false)
	if err != nil {
		return readProjectConfiguration{}, err
	}
	bashBin, err := ParseString(raw, "bash_bin", "bash-bin", false)
	if err != nil {
		return readProjectConfiguration{}, err
	}
	makeBin, err := ParseString(raw, "make_bin", "make-bin", false)
	if err != nil {
		return readProjectConfiguration{}, err
	}
	return readProjectConfiguration{
		haskell: backend.HaskellConfig{Standalone: standalone, GHCBin: ghcBin, HLSBin: hlsBin, GHCVersion: ghcVersion, HLSVersion: hlsVersion},
		kotlin:  backend.KotlinConfig{KotlinBin: kotlinBin},
		bash:    backend.BashConfig{BashBin: bashBin},
		make:    backend.MakeConfig{MakeBin: makeBin},
	}, nil
}
