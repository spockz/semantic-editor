// Package operation declares the native verification result output contract.
package operation

// NativeVerifyOutputSchema returns the JSON-facing shape of a native verification result, independent of any ingress-specific projection.
func NativeVerifyOutputSchema() *ValueSchema {
	textValue := ValueSchema{Kind: SchemaString}
	integerValue := ValueSchema{Kind: SchemaInteger}
	booleanValue := ValueSchema{Kind: SchemaBoolean}
	textList := ValueSchema{Kind: SchemaArray, Items: &textValue}
	nullableTextList := ValueSchema{Kind: SchemaArray, Nullable: true, Items: &textValue}
	object := func(properties map[string]ValueSchema, required ...string) ValueSchema {
		return ValueSchema{Kind: SchemaObject, Properties: properties, Required: required}
	}
	position := object(map[string]ValueSchema{
		"line":      integerValue,
		"character": integerValue,
	}, "line", "character")
	rangeValue := object(map[string]ValueSchema{
		"start": position,
		"end":   position,
	}, "start", "end")
	location := object(map[string]ValueSchema{
		"uri":   textValue,
		"range": rangeValue,
	}, "uri", "range")
	diagnostic := object(map[string]ValueSchema{
		"message":  textValue,
		"severity": integerValue,
		"location": location,
	}, "message")
	coverage := object(map[string]ValueSchema{
		"language":         textValue,
		"status":           textValue,
		"paths":            textList,
		"module_root":      textValue,
		"package_pattern":  textValue,
		"build_exclusions": textList,
		"detail":           textValue,
	}, "language", "status")
	scope := object(map[string]ValueSchema{
		"Paths": nullableTextList,
	}, "Paths")
	hookResult := object(map[string]ValueSchema{
		"ID":          textValue,
		"Kind":        textValue,
		"Status":      textValue,
		"ExitCode":    integerValue,
		"Stdout":      textValue,
		"Stderr":      textValue,
		"Truncated":   booleanValue,
		"Changed":     nullableTextList,
		"Diagnostics": nullableTextList,
		"StartedAt":   textValue,
		"FinishedAt":  textValue,
		"Error":       textValue,
		"Signal":      textValue,
		"ConfigPath":  textValue,
		"ScopePaths":  nullableTextList,
	}, "ID", "Kind", "Status", "ExitCode", "Stdout", "Stderr", "Truncated", "Changed", "Diagnostics", "StartedAt", "FinishedAt", "Error", "Signal", "ConfigPath", "ScopePaths")
	hookResults := ValueSchema{Kind: SchemaArray, Nullable: true, Items: &hookResult}
	report := object(map[string]ValueSchema{
		"Root":       textValue,
		"ConfigPath": textValue,
		"Phase":      textValue,
		"Scope":      scope,
		"Results":    hookResults,
		"Completed":  booleanValue,
		"Partial":    booleanValue,
	}, "Root", "ConfigPath", "Phase", "Scope", "Results", "Completed", "Partial")
	diagnostics := ValueSchema{Kind: SchemaArray, Items: &diagnostic}
	coverageList := ValueSchema{Kind: SchemaArray, Items: &coverage}
	return &ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"diagnostics": diagnostics,
		"formatted":   booleanValue,
		"normalize":   report,
		"checks":      report,
		"scope":       scope,
		"failure":     textValue,
		"coverage":    coverageList,
	}, Required: []string{"normalize", "checks", "scope"}}
}
