// Package cli verifies that structured command-line flags enter the shared typed parameter validator.
package cli

import (
	"encoding/json"
	"semedit/internal/operation"
	"testing"
)

func TestRawParamsDecodesObjectAndArrayJSONFlagsForTypedValidation(t *testing.T) {
	objectSchema := operation.ValueSchema{Kind: operation.SchemaObject, Properties: map[string]operation.ValueSchema{
		"text": {Kind: operation.SchemaString}, "enabled": {Kind: operation.SchemaBoolean},
	}, Required: []string{"text", "enabled"}}
	integerSchema := operation.ValueSchema{Kind: operation.SchemaInteger}
	arraySchema := operation.ValueSchema{Kind: operation.SchemaArray, Items: &integerSchema}
	objectJSON := `{"text":"","enabled":false}`
	arrayJSON := `[0,9007199254740993]`
	entry := operation.Entry{Params: []operation.ParameterContract{
		{Name: "payload", JSONName: "payload", CLIName: "payload", Type: operation.ParamObject, Required: true, Schema: &objectSchema},
		{Name: "indices", JSONName: "indices", CLIName: "indices", Type: operation.ParamArray, Required: true, Schema: &arraySchema},
	}}
	raw, err := rawParams(entry, &flagValues{json: map[string]*string{"payload": &objectJSON, "indices": &arrayJSON}}, func(string) bool { return true })
	if err != nil {
		t.Fatalf("decode structured flags: %v", err)
	}
	if err := operation.ValidateParameterValues(raw, entry.Params); err != nil {
		t.Fatalf("validate decoded structured flags: %v", err)
	}
	object := raw["payload"].(map[string]any)
	if object["text"] != "" || object["enabled"] != false {
		t.Fatalf("empty string or false value was not preserved: %#v", object)
	}
	indices := raw["indices"].([]any)
	if indices[0] != json.Number("0") || indices[1] != json.Number("9007199254740993") {
		t.Fatalf("exact JSON integer tokens were not preserved: %#v", indices)
	}
	invalidJSON := `{`
	if _, err := rawParams(entry, &flagValues{json: map[string]*string{"payload": &invalidJSON, "indices": &arrayJSON}}, func(string) bool { return true }); err == nil {
		t.Error("invalid JSON flag was accepted")
	}
	emptyJSON := ""
	if _, err := rawParams(entry, &flagValues{json: map[string]*string{"payload": &emptyJSON, "indices": &arrayJSON}}, func(name string) bool { return name == "payload" }); err == nil {
		t.Error("explicit empty JSON flag was treated as omitted")
	}
}
