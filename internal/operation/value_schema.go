// This file defines explicit recursive input schemas for pure validation before registry and ingress integration.
// It stays in operation so future CLI, MCP, and batch adapters can consume one shared contract.

package operation

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
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

// JSONSchema converts an explicit recursive schema to a closed JSON Schema object.
func (schema ValueSchema) JSONSchema() (map[string]any, error) {
	if err := ValidateSchema(schema); err != nil {
		return nil, fmt.Errorf("convert value schema: %w", err)
	}
	return valueSchemaJSON(schema)
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
		if err := validateStringSchema(schema); err != nil {
			return err
		}
	case SchemaBoolean:
		if schemaHasNonBooleanFields(schema) {
			return fmt.Errorf("boolean schema contains fields for another kind")
		}
	case SchemaInteger, SchemaNumber:
		if err := validateNumericSchema(schema); err != nil {
			return err
		}
	case SchemaObject:
		if err := validateObjectSchema(schema); err != nil {
			return err
		}
	case SchemaArray:
		if err := validateArraySchema(schema); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported schema kind %q", schema.Kind)
	}
	return validateSchemaEnum(schema)
}

func validateStringSchema(schema ValueSchema) error {
	if schemaHasNonStringFields(schema) {
		return fmt.Errorf("string schema contains fields for another kind")
	}
	return validateBounds(schema.MinLength, schema.MaxLength, "string length")
}

func schemaHasNonStringFields(schema ValueSchema) bool {
	return schema.Items != nil || schema.Properties != nil || len(schema.Required) > 0 || schema.MinItems != nil || schema.MaxItems != nil || schema.Minimum != nil || schema.Maximum != nil
}

func schemaHasNonBooleanFields(schema ValueSchema) bool {
	return schema.Items != nil || schema.Properties != nil || len(schema.Required) > 0 || schema.MinLength != nil || schema.MaxLength != nil || schema.MinItems != nil || schema.MaxItems != nil || schema.Minimum != nil || schema.Maximum != nil
}

func validateNumericSchema(schema ValueSchema) error {
	if schema.Items != nil || schema.Properties != nil || len(schema.Required) > 0 || schema.MinLength != nil || schema.MaxLength != nil || schema.MinItems != nil || schema.MaxItems != nil {
		return fmt.Errorf("%s schema contains fields for another kind", schema.Kind)
	}
	return validateNumberBounds(schema.Minimum, schema.Maximum)
}

func validateObjectSchema(schema ValueSchema) error {
	if schema.Items != nil || schema.MinLength != nil || schema.MaxLength != nil || schema.MinItems != nil || schema.MaxItems != nil || schema.Minimum != nil || schema.Maximum != nil {
		return fmt.Errorf("object schema contains fields for another kind")
	}
	if err := validateObjectProperties(schema.Properties); err != nil {
		return err
	}
	return validateRequiredProperties(schema.Properties, schema.Required)
}

func validateObjectProperties(properties map[string]ValueSchema) error {
	for name, property := range properties {
		if name == "" {
			return fmt.Errorf("object schema has an empty property name")
		}
		if err := ValidateSchema(property); err != nil {
			return fmt.Errorf("property %q schema: %w", name, err)
		}
	}
	return nil
}

func validateRequiredProperties(properties map[string]ValueSchema, required []string) error {
	seen := make(map[string]struct{}, len(required))
	for _, name := range required {
		if _, ok := properties[name]; !ok {
			return fmt.Errorf("required property %q has no schema", name)
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("required property %q is repeated", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

func validateArraySchema(schema ValueSchema) error {
	if schema.Properties != nil || len(schema.Required) > 0 || schema.MinLength != nil || schema.MaxLength != nil || schema.Minimum != nil || schema.Maximum != nil {
		return fmt.Errorf("array schema contains fields for another kind")
	}
	if schema.Items == nil {
		return fmt.Errorf("array schema requires an item schema")
	}
	if err := ValidateSchema(*schema.Items); err != nil {
		return fmt.Errorf("array item schema: %w", err)
	}
	return validateBounds(schema.MinItems, schema.MaxItems, "array length")
}

func validateSchemaEnum(schema ValueSchema) error {
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
	if maximum != nil && *maximum < 0 {
		return fmt.Errorf("%s maximum must be non-negative", label)
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
	return validateValueKind(schema, value)
}

func validateValueKind(schema ValueSchema, value any) error {
	switch schema.Kind {
	case SchemaString:
		return validateStringValue(schema, value)
	case SchemaInteger, SchemaNumber:
		return validateNumberValue(schema, value)
	case SchemaBoolean:
		return validateBooleanValue(value)
	case SchemaObject:
		return validateObjectValue(schema, value)
	case SchemaArray:
		return validateArrayValue(schema, value)
	default:
		return fmt.Errorf("unsupported schema kind %q", schema.Kind)
	}
}

func validateStringValue(schema ValueSchema, value any) error {
	text, _ := value.(string)
	length := utf8.RuneCountInString(text)
	if schema.MinLength != nil && length < *schema.MinLength {
		return fmt.Errorf("string length %d is below minimum %d", length, *schema.MinLength)
	}
	if schema.MaxLength != nil && length > *schema.MaxLength {
		return fmt.Errorf("string length %d exceeds maximum %d", length, *schema.MaxLength)
	}
	return nil
}

func validateNumberValue(schema ValueSchema, value any) error {
	number, ok := exactNumericValue(value)
	if !ok {
		return fmt.Errorf("numeric value %T is invalid", value)
	}
	if schema.Minimum != nil {
		minimum := new(big.Rat).SetFloat64(*schema.Minimum)
		if number.Cmp(minimum) < 0 {
			return fmt.Errorf("number %s is below minimum %v", number.RatString(), *schema.Minimum)
		}
	}
	if schema.Maximum != nil {
		maximum := new(big.Rat).SetFloat64(*schema.Maximum)
		if number.Cmp(maximum) > 0 {
			return fmt.Errorf("number %s exceeds maximum %v", number.RatString(), *schema.Maximum)
		}
	}
	return nil
}

func validateBooleanValue(value any) error {
	if _, ok := value.(bool); !ok {
		return fmt.Errorf("boolean value has type %T", value)
	}
	return nil
}

func validateObjectValue(schema ValueSchema, value any) error {
	object, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("object value has type %T", value)
	}
	if err := validateObjectKeys(schema.Properties, object); err != nil {
		return err
	}
	if err := validateObjectRequired(schema.Required, object); err != nil {
		return err
	}
	for name, propertyValue := range object {
		if err := validateValue(schema.Properties[name], propertyValue); err != nil {
			return fmt.Errorf("property %q: %w", name, err)
		}
	}
	return nil
}

func validateObjectKeys(properties map[string]ValueSchema, object map[string]any) error {
	for name := range object {
		if _, ok := properties[name]; !ok {
			return fmt.Errorf("unknown property %q", name)
		}
	}
	return nil
}

func validateObjectRequired(required []string, object map[string]any) error {
	for _, name := range required {
		if _, ok := object[name]; !ok {
			return fmt.Errorf("required property %q is missing", name)
		}
	}
	return nil
}

func validateArrayValue(schema ValueSchema, value any) error {
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
		_, ok := exactIntegerValue(value)
		return ok
	case SchemaNumber:
		_, ok := exactNumericValue(value)
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
		case SchemaString:
			if candidate == value {
				return true
			}
		case SchemaBoolean:
			if candidate == value {
				return true
			}
		case SchemaInteger:
			left, leftOK := exactIntegerValue(candidate)
			right, rightOK := exactIntegerValue(value)
			if leftOK && rightOK && left.Cmp(right) == 0 {
				return true
			}
		case SchemaNumber:
			left, leftOK := exactNumericValue(candidate)
			right, rightOK := exactNumericValue(value)
			if leftOK && rightOK && left.Cmp(right) == 0 {
				return true
			}

		case SchemaObject, SchemaArray:
			return false
		}
	}
	return false
}

func exactNumericValue(value any) (*big.Rat, bool) {
	switch value := value.(type) {
	case int:
		return new(big.Rat).SetInt64(int64(value)), true
	case int8:
		return new(big.Rat).SetInt64(int64(value)), true
	case int16:
		return new(big.Rat).SetInt64(int64(value)), true
	case int32:
		return new(big.Rat).SetInt64(int64(value)), true
	case int64:
		return new(big.Rat).SetInt64(value), true
	case uint:
		return new(big.Rat).SetInt(new(big.Int).SetUint64(uint64(value))), true
	case uint8:
		return new(big.Rat).SetInt(new(big.Int).SetUint64(uint64(value))), true
	case uint16:
		return new(big.Rat).SetInt(new(big.Int).SetUint64(uint64(value))), true
	case uint32:
		return new(big.Rat).SetInt(new(big.Int).SetUint64(uint64(value))), true
	case uint64:
		return new(big.Rat).SetInt(new(big.Int).SetUint64(value)), true
	case float32:
		number, ok := numericValue(value)
		if !ok {
			return nil, false
		}
		return new(big.Rat).SetFloat64(number), true
	case float64:
		number, ok := numericValue(value)
		if !ok {
			return nil, false
		}
		return new(big.Rat).SetFloat64(number), true
	case json.Number:
		return boundedJSONNumber(value)
	default:
		return nil, false
	}
}

func exactIntegerValue(value any) (*big.Int, bool) {
	if floating, ok := value.(float64); ok && math.Abs(floating) > 9007199254740991 {
		return nil, false
	}
	if floating, ok := value.(float32); ok && math.Abs(float64(floating)) > 9007199254740991 {
		return nil, false
	}
	number, ok := exactNumericValue(value)
	if !ok || !number.IsInt() {
		return nil, false
	}
	return new(big.Int).Set(number.Num()), true
}

func valueSchemaJSON(schema ValueSchema) (map[string]any, error) {
	properties := make(map[string]any, len(schema.Properties))
	for name, property := range schema.Properties {
		converted, err := valueSchemaJSON(property)
		if err != nil {
			return nil, fmt.Errorf("convert property %q schema: %w", name, err)
		}
		properties[name] = converted
	}
	result := map[string]any{"type": string(schema.Kind)}
	if schema.Nullable {
		result["type"] = []string{string(schema.Kind), "null"}
	}
	if len(schema.Enum) > 0 {
		result["enum"] = append([]any(nil), schema.Enum...)
	}
	if schema.Kind == SchemaObject {
		result["properties"] = properties
		result["additionalProperties"] = false
		if len(schema.Required) > 0 {
			result["required"] = append([]string(nil), schema.Required...)
		}
	}
	if schema.Kind == SchemaArray {
		items, err := valueSchemaJSON(*schema.Items)
		if err != nil {
			return nil, fmt.Errorf("convert array item schema: %w", err)
		}
		result["items"] = items
	}
	if schema.MinLength != nil {
		result["minLength"] = *schema.MinLength
	}
	if schema.MaxLength != nil {
		result["maxLength"] = *schema.MaxLength
	}
	if schema.MinItems != nil {
		result["minItems"] = *schema.MinItems
	}
	if schema.MaxItems != nil {
		result["maxItems"] = *schema.MaxItems
	}
	if schema.Minimum != nil {
		result["minimum"] = *schema.Minimum
	}
	if schema.Maximum != nil {
		result["maximum"] = *schema.Maximum
	}
	return result, nil
}

func boundedJSONNumber(value json.Number) (*big.Rat, bool) {
	raw := value.String()
	if len(raw) == 0 || len(raw) > 1024 || !json.Valid([]byte(raw)) {
		return nil, false
	}
	exponentAt := len(raw)
	for index, character := range raw {
		if character == 101 || character == 69 {
			exponentAt = index
			break
		}
	}
	allZero := true
	for _, character := range raw[:exponentAt] {
		if character != 45 && character != 46 && character != 48 {
			allZero = false
			break
		}
	}
	if allZero {
		return new(big.Rat), true
	}
	if exponentAt < len(raw) {
		exponent, err := strconv.ParseInt(raw[exponentAt+1:], 10, 16)
		if err != nil || exponent > 1024 || exponent < -1024 {
			return nil, false
		}
	}
	number, ok := new(big.Rat).SetString(raw)
	return number, ok
}
