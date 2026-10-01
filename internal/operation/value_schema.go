// This file defines explicit recursive input schemas for pure validation before registry and ingress integration.
// It stays in operation so future CLI, MCP, and batch adapters can consume one shared contract.

package operation

import (
	"encoding/json"
	"fmt"
	"math"
	"unicode/utf8"
)

// ValueSchema declares one closed recursive value shape.
type ValueSchema struct {
	Kind       SchemaKind
	Nullable   bool
	Enum       []any
	Properties map[string]ValueSchema
	Required   []string
	Items      *ValueSchema
	MinLength  *int
	MaxLength  *int
	MinItems   *int
	MaxItems   *int
	Minimum    *float64
	Maximum    *float64
}

// SchemaString through SchemaArray enumerate the closed schema value kinds.
const (
	SchemaString  SchemaKind = "string"
	SchemaBoolean SchemaKind = "boolean"
	SchemaInteger SchemaKind = "integer"
	SchemaNumber  SchemaKind = "number"
	SchemaObject  SchemaKind = "object"
	SchemaArray   SchemaKind = "array"
)

// SchemaKind identifies a supported structured value kind.
type SchemaKind string

// ValidateSchema rejects unsupported or internally inconsistent schema declarations.
func ValidateSchema(schema ValueSchema) error {
	switch schema.Kind {
	case SchemaString:
		if schema.Items != nil || schema.Properties != nil || len(schema.Required) > 0 || schema.MinItems != nil || schema.MaxItems != nil || schema.Minimum != nil || schema.Maximum != nil {
			return fmt.Errorf("string schema contains fields for another kind")
		}
		return validateBounds(schema.MinLength, schema.MaxLength, "string length")
	case SchemaBoolean:
		if schema.Items != nil || schema.Properties != nil || len(schema.Required) > 0 || schema.MinLength != nil || schema.MaxLength != nil || schema.MinItems != nil || schema.MaxItems != nil || schema.Minimum != nil || schema.Maximum != nil {
			return fmt.Errorf("boolean schema contains fields for another kind")
		}
	case SchemaInteger, SchemaNumber:
		if schema.Items != nil || schema.Properties != nil || len(schema.Required) > 0 || schema.MinLength != nil || schema.MaxLength != nil || schema.MinItems != nil || schema.MaxItems != nil {
			return fmt.Errorf("%s schema contains fields for another kind", schema.Kind)
		}
		if err := validateNumberBounds(schema.Minimum, schema.Maximum); err != nil {
			return err
		}
	case SchemaObject:
		if schema.Items != nil || schema.MinLength != nil || schema.MaxLength != nil || schema.MinItems != nil || schema.MaxItems != nil || schema.Minimum != nil || schema.Maximum != nil {
			return fmt.Errorf("object schema contains fields for another kind")
		}
		for name, property := range schema.Properties {
			if name == "" {
				return fmt.Errorf("object schema has an empty property name")
			}
			if err := ValidateSchema(property); err != nil {
				return fmt.Errorf("property %q schema: %w", name, err)
			}
		}
		seen := make(map[string]struct{}, len(schema.Required))
		for _, name := range schema.Required {
			if _, ok := schema.Properties[name]; !ok {
				return fmt.Errorf("required property %q has no schema", name)
			}
			if _, ok := seen[name]; ok {
				return fmt.Errorf("required property %q is repeated", name)
			}
			seen[name] = struct{}{}
		}
	case SchemaArray:
		if schema.Properties != nil || len(schema.Required) > 0 || schema.MinLength != nil || schema.MaxLength != nil || schema.Minimum != nil || schema.Maximum != nil {
			return fmt.Errorf("array schema contains fields for another kind")
		}
		if schema.Items == nil {
			return fmt.Errorf("array schema requires an item schema")
		}
		if err := ValidateSchema(*schema.Items); err != nil {
			return fmt.Errorf("array item schema: %w", err)
		}
		if err := validateBounds(schema.MinItems, schema.MaxItems, "array length"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported schema kind %q", schema.Kind)
	}
	for _, value := range schema.Enum {
		if value == nil {
			if !schema.Nullable {
				return fmt.Errorf("null enum value requires a nullable schema")
			}
			continue
		}
		if !valueMatchesKind(schema.Kind, value) || schema.Kind == SchemaObject || schema.Kind == SchemaArray {
			return fmt.Errorf("enum value %T is incompatible with schema kind %q", value, schema.Kind)
		}
	}
	return nil
}

// ValidateValue validates a value while preserving missing and null as distinct states.
func ValidateValue(schema ValueSchema, value any) error {
	if err := ValidateSchema(schema); err != nil {
		return fmt.Errorf("invalid value schema: %w", err)
	}
	return validateValue(schema, value)
}

// ValidateObject validates an object against a closed object schema.
func ValidateObject(schema ValueSchema, value map[string]any) error {
	if schema.Kind != SchemaObject {
		return fmt.Errorf("validate object: schema kind is %q, want %q", schema.Kind, SchemaObject)
	}
	return ValidateValue(schema, value)
}

func validateBounds(minimum, maximum *int, label string) error {
	if minimum != nil && *minimum < 0 {
		return fmt.Errorf("%s minimum must be non-negative", label)
	}
	if minimum != nil && maximum != nil && *minimum > *maximum {
		return fmt.Errorf("%s minimum exceeds maximum", label)
	}
	return nil
}

func validateNumberBounds(minimum, maximum *float64) error {
	if minimum != nil && (math.IsNaN(*minimum) || math.IsInf(*minimum, 0)) {
		return fmt.Errorf("numeric minimum must be finite")
	}
	if maximum != nil && (math.IsNaN(*maximum) || math.IsInf(*maximum, 0)) {
		return fmt.Errorf("numeric maximum must be finite")
	}
	if minimum != nil && maximum != nil && *minimum > *maximum {
		return fmt.Errorf("numeric minimum exceeds maximum")
	}
	return nil
}

func validateValue(schema ValueSchema, value any) error {
	if value == nil {
		if schema.Nullable && (len(schema.Enum) == 0 || enumContains(schema.Enum, nil, schema.Kind)) {
			return nil
		}
		return fmt.Errorf("null is not allowed")
	}
	if !valueMatchesKind(schema.Kind, value) {
		return fmt.Errorf("value has type %T, want %q", value, schema.Kind)
	}
	if len(schema.Enum) > 0 && !enumContains(schema.Enum, value, schema.Kind) {
		return fmt.Errorf("value %v is not in the allowed enum", value)
	}
	switch schema.Kind {
	case SchemaString:
		text, _ := value.(string)
		length := utf8.RuneCountInString(text)
		if schema.MinLength != nil && length < *schema.MinLength {
			return fmt.Errorf("string length %d is below minimum %d", length, *schema.MinLength)
		}
		if schema.MaxLength != nil && length > *schema.MaxLength {
			return fmt.Errorf("string length %d exceeds maximum %d", length, *schema.MaxLength)
		}
	case SchemaInteger, SchemaNumber:
		number, ok := numericValue(value)
		if !ok {
			return fmt.Errorf("numeric value %T is invalid", value)
		}
		if schema.Minimum != nil && number < *schema.Minimum {
			return fmt.Errorf("number %v is below minimum %v", number, *schema.Minimum)
		}
		if schema.Maximum != nil && number > *schema.Maximum {
			return fmt.Errorf("number %v exceeds maximum %v", number, *schema.Maximum)
		}
	case SchemaObject:
		object, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("object value has type %T", value)
		}
		for name := range object {
			if _, ok := schema.Properties[name]; !ok {
				return fmt.Errorf("unknown property %q", name)
			}
		}
		for _, name := range schema.Required {
			if _, ok := object[name]; !ok {
				return fmt.Errorf("required property %q is missing", name)
			}
		}
		for name, propertyValue := range object {
			if err := validateValue(schema.Properties[name], propertyValue); err != nil {
				return fmt.Errorf("property %q: %w", name, err)
			}
		}
	case SchemaArray:
		items, ok := value.([]any)
		if !ok {
			return fmt.Errorf("array value has type %T", value)
		}
		if schema.MinItems != nil && len(items) < *schema.MinItems {
			return fmt.Errorf("array length %d is below minimum %d", len(items), *schema.MinItems)
		}
		if schema.MaxItems != nil && len(items) > *schema.MaxItems {
			return fmt.Errorf("array length %d exceeds maximum %d", len(items), *schema.MaxItems)
		}
		for index, item := range items {
			if err := validateValue(*schema.Items, item); err != nil {
				return fmt.Errorf("array item %d: %w", index, err)
			}
		}

	case SchemaBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("boolean value has type %T", value)
		}
	}
	return nil
}

func valueMatchesKind(kind SchemaKind, value any) bool {
	switch kind {
	case SchemaString:
		_, ok := value.(string)
		return ok
	case SchemaBoolean:
		_, ok := value.(bool)
		return ok
	case SchemaInteger:
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, json.Number:
			number, ok := numericValue(value)
			return ok && math.Trunc(number) == number
		}
	case SchemaNumber:
		_, ok := numericValue(value)
		return ok
	case SchemaObject:
		_, ok := value.(map[string]any)
		return ok
	case SchemaArray:
		_, ok := value.([]any)
		return ok
	}
	return false
}

func numericValue(value any) (float64, bool) {
	var number float64
	var err error
	switch value := value.(type) {
	case int:
		number = float64(value)
	case int8:
		number = float64(value)
	case int16:
		number = float64(value)
	case int32:
		number = float64(value)
	case int64:
		number = float64(value)
	case uint:
		number = float64(value)
	case uint8:
		number = float64(value)
	case uint16:
		number = float64(value)
	case uint32:
		number = float64(value)
	case uint64:
		number = float64(value)
	case float32:
		number = float64(value)
	case float64:
		number = value
	case json.Number:
		number, err = value.Float64()
	default:
		return 0, false
	}
	return number, err == nil && !math.IsNaN(number) && !math.IsInf(number, 0)
}

func enumContains(values []any, value any, kind SchemaKind) bool {
	for _, candidate := range values {
		if candidate == nil && value == nil {
			return true
		}
		if candidate == nil || value == nil {
			continue
		}
		switch kind {
		case SchemaString, SchemaBoolean:
			if candidate == value {
				return true
			}
		case SchemaInteger, SchemaNumber:
			left, leftOK := numericValue(candidate)
			right, rightOK := numericValue(value)
			if leftOK && rightOK && left == right {
				return true
			}

		case SchemaObject, SchemaArray:
			return false
		}
	}
	return false
}
