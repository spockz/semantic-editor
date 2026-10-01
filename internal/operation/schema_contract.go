// Package operation owns explicit parameter schemas shared by CLI, MCP, and batch ingress.
// Keeping validation beside the registry prevents adapters from accepting different object shapes.
package operation

import "fmt"

// ValidateParameterContracts checks explicit recursive schemas and structured parameter kinds.
func ValidateParameterContracts(params []ParameterContract) error {
	for _, param := range params {
		if param.Schema == nil {
			if param.Type == ParamObject || param.Type == ParamArray {
				return fmt.Errorf("parameter %q of type %q requires an explicit schema: %w", param.JSONName, param.Type, ErrInvalidParams)
			}
			continue
		}
		if len(param.Enums) > 0 {
			return fmt.Errorf("parameter %q cannot combine a recursive schema with legacy enums: %w", param.JSONName, ErrInvalidParams)
		}
		if err := ValidateSchema(*param.Schema); err != nil {
			return fmt.Errorf("parameter %q schema: %w", param.JSONName, err)
		}
		var expected SchemaKind
		switch param.Type {
		case ParamString:
			expected = SchemaString
		case ParamBoolean:
			expected = SchemaBoolean
		case ParamInteger:
			expected = SchemaInteger
		case ParamStringSlice:
			expected = SchemaArray
			if param.Schema.Kind == SchemaArray && param.Schema.Items.Kind != SchemaString {
				return fmt.Errorf("parameter %q requires an array of strings: %w", param.JSONName, ErrInvalidParams)
			}
		case ParamObject:
			expected = SchemaObject
		case ParamArray:
			expected = SchemaArray
		default:
			return fmt.Errorf("parameter %q has unsupported type %q: %w", param.JSONName, param.Type, ErrInvalidParams)
		}
		if param.Schema.Kind != expected {
			return fmt.Errorf("parameter %q schema kind %q does not match type %q: %w", param.JSONName, param.Schema.Kind, param.Type, ErrInvalidParams)
		}
		if param.Default != nil {
			if err := ValidateValue(*param.Schema, param.Default); err != nil {
				return fmt.Errorf("parameter %q default: %w", param.JSONName, err)
			}
		}
	}
	return nil
}

// ValidateParameterValues validates every supplied schema-backed value while preserving missing and null states.
func ValidateParameterValues(raw map[string]any, params []ParameterContract) error {
	if err := ValidateParameterContracts(params); err != nil {
		return err
	}
	return CheckParams(raw, params)
}

// ParameterValueSchema returns the explicit value shape accepted for one parameter.
func ParameterValueSchema(param ParameterContract) (ValueSchema, error) {
	if param.Schema != nil {
		return *param.Schema, nil
	}
	schema := ValueSchema{Nullable: !param.Required}
	switch param.Type {
	case ParamString:
		schema.Kind = SchemaString
		for _, item := range param.Enums {
			schema.Enum = append(schema.Enum, item)
		}
	case ParamBoolean:
		schema.Kind = SchemaBoolean
	case ParamInteger:
		schema.Kind = SchemaInteger
	case ParamStringSlice:
		schema.Kind = SchemaArray
		schema.Items = &ValueSchema{Kind: SchemaString}
	case ParamObject:
		return ValueSchema{}, fmt.Errorf("parameter %q has no explicit object schema: %w", param.JSONName, ErrInvalidParams)
	case ParamArray:
		return ValueSchema{}, fmt.Errorf("parameter %q has no explicit array schema: %w", param.JSONName, ErrInvalidParams)
	default:
		return ValueSchema{}, fmt.Errorf("parameter %q has unsupported type %q: %w", param.JSONName, param.Type, ErrInvalidParams)
	}
	return schema, nil
}
