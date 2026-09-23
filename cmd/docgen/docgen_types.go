// docgen_types.go keeps the shared data shapes used by extraction and rendering together.
package main

// CodeCapability represents extracted language capability metadata.
type CodeCapability struct {
	Language           string
	DisplayName        string
	Maturity           string
	SupportedModifiers []string
	Operations         map[string]OpMetadata
	Limitations        []Constraint
}

// OpMetadata describes an operation.
type OpMetadata struct {
	Supported    bool
	Description  string
	CLICommand   string
	MCPTool      string
	PlacementKey bool
}

// Constraint describes a language rule.
type Constraint struct {
	Title       string
	Description string
	Severity    string // "error", "warning", "info"
}

// TxtarStep captures a single execution step from a txtar test script.
type TxtarStep struct {
	Number      int
	Description string
	Command     string
	IsNegated   bool
	MCPTool     string
	MCPArgsJSON string
	ExpectedOut string
	ExpectedErr string
}

// TxtarFileOutput captures the expected post-transformation state of a file in a txtar scenario.
type TxtarFileOutput struct {
	Path      string
	Content   string
	DiffLines []DiffLine
}

// TxtarExample captures parsed executable scenario from a .txtar file.
type TxtarExample struct {
	Filename    string
	Title       string
	Description string
	Steps       []TxtarStep
	InputFile   string
	InputCode   string
	OutputFile  string
	OutputCode  string
	DiffLines   []DiffLine
	Outputs     []TxtarFileOutput
}

// DiffLine represents a line in a unified diff.
type DiffLine struct {
	Type    string // "add", "del", "same"
	Content string
}
