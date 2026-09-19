// Package capability declares the code-derived language capability matrix consumed by cmd/docgen and internal/operation.
package capability

// Well-known operation keys used as map keys in LanguageMatrix.Operations.
const (
	// OpLookup resolves a symbol location.
	OpLookup = "lookup"
	// OpRename performs a semantic rename.
	OpRename = "rename"
	// OpVerify formats and checks project diagnostics.
	OpVerify = "verify"
)

// LanguageMatrix declares capabilities and constraints for a single language backend.
type LanguageMatrix struct {
	Language           string
	DisplayName        string
	Maturity           string
	SupportedModifiers []string
	Operations         map[string]OpCapability
	Limitations        []Constraint
}

// OpCapability describes backend support and interface metadata for one operation.
type OpCapability struct {
	Supported    bool
	Description  string
	CLICommand   string
	MCPTool      string
	PlacementKey bool
	Level        string
	ReadOnly     bool
}

// Constraint describes a language rule surfaced in generated documentation.
type Constraint struct {
	Title       string
	Description string
	// Severity is one of "error", "warning", or "info".
	Severity string
}
