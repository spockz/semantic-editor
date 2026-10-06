// Package operation derives per-language capability views from registered operations and static profiles.
package operation

import (
	"fmt"
	"slices"

	"semedit/internal/astedit"
	"semedit/internal/backend"
	"semedit/internal/capability"
)

// Profile carries the static language metadata backing a capability matrix view.
type Profile struct {
	Language           backend.LanguageID
	DisplayName        string
	Maturity           string
	SupportedModifiers []string
	Limitations        []capability.Constraint
}

const readOnlyPreview = "Read-only preview"

func goModifiers() []string {
	modifiers := make([]string, 0, 3)
	for _, modifier := range (astedit.GolangBackend{}).SupportedAccessModifiers() {
		modifiers = append(modifiers, string(modifier))
	}
	return modifiers
}

func supportedAccessModifiers(candidate backend.Backend) []string {
	provider, ok := candidate.(backend.MatrixProvider)
	if ok {
		return slices.Clone(provider.CapabilityMatrix().SupportedModifiers)
	}
	if candidate.Language() == backend.LanguageGo {
		return goModifiers()
	}
	return nil
}

func profileFor(language backend.LanguageID) (Profile, bool) {
	switch language {
	case backend.LanguageGo:
		return goProfile(), true
	case backend.LanguageRust:
		return rustProfile(), true
	case backend.LanguageJava:
		return javaProfile(), true
	case backend.LanguageScala:
		return scalaProfile(), true
	case backend.LanguageKotlin:
		return kotlinProfile(), true
	case backend.LanguageBash:
		return bashProfile(), true
	case backend.LanguageMake:
		return makeProfile(), true
	case backend.LanguageHaskell:
		return haskellProfile(), true
	default:
		return Profile{}, false
	}
}

// Matrix builds the capability view for language from registered operations and its static profile.
// Operation support derives from handler presence; descriptions, commands, and tools derive from defs.
func (r *Registry) Matrix(language backend.LanguageID) (capability.LanguageMatrix, error) {
	profile, ok := profileFor(language)
	if !ok {
		return capability.LanguageMatrix{}, fmt.Errorf("capability matrix for language %q: %w", language, ErrUnsupportedLanguage)
	}
	operations := make(map[string]capability.OpCapability, len(r.byKey))
	for _, entry := range r.All() {
		supported := slices.Contains(entry.Languages, language) || slices.Contains(entry.Languages, backend.LanguageAuto) && len(entry.Languages) == 1
		var languageContract *capability.OperationContract
		contract, hasContract := entry.LanguageContracts[language]
		if !hasContract && supported {
			contract, hasContract = entry.LanguageContracts[backend.LanguageAuto]
		}
		if hasContract {
			languageContract = &contract
		}
		command := ""
		if entry.CLIName != "" {
			command = "semedit " + entry.CLIName
		}
		operations[entry.Key] = capability.OpCapability{
			Supported:        supported,
			Description:      entry.Summary,
			CLICommand:       command,
			MCPTool:          entry.MCPName,
			PlacementKey:     entry.PlacementKey,
			Level:            string(entry.Level),
			ReadOnly:         entry.ReadOnly,
			LanguageContract: languageContract,
		}
	}
	return capability.LanguageMatrix{
		Language:           string(language),
		DisplayName:        profile.DisplayName,
		Maturity:           profile.Maturity,
		SupportedModifiers: profile.SupportedModifiers,
		Operations:         operations,
		Limitations:        profile.Limitations,
	}, nil
}
