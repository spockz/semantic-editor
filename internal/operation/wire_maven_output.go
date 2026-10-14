// Package operation declares the native Maven result and known-error output contracts.
package operation

import (
	"errors"
	"github.com/spockz/semantic-editor/internal/maven"
)

func mavenResultOutputSchema() *ValueSchema {
	textValue := ValueSchema{Kind: SchemaString}
	integerValue := ValueSchema{Kind: SchemaInteger}
	booleanValue := ValueSchema{Kind: SchemaBoolean}
	command := ValueSchema{Kind: SchemaArray, Nullable: true, Items: &textValue}
	return &ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"goal":             textValue,
		"tool":             textValue,
		"command":          command,
		"exit_code":        integerValue,
		"stdout":           textValue,
		"stderr":           textValue,
		"stdout_truncated": booleanValue,
		"stderr_truncated": booleanValue,
		"log_tail":         textValue,
		"error":            textValue,
		"duration_ms":      integerValue,
	}, Required: []string{"goal", "tool", "command", "exit_code", "duration_ms"}}
}

func projectMavenErrorOutput(err error) (any, bool, error) {
	var runnerError *maven.Error
	if !errors.As(err, &runnerError) || runnerError.Result == nil {
		return nil, false, nil
	}
	return *runnerError.Result, true, nil
}
