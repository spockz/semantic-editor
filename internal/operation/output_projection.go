// This file normalizes explicitly projected values before validating declared operation output contracts, making MCP output schemas enforce actual result values.

package operation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
)

const (
	errorContinuationField = "continuations"
	errorRevisionField     = "expected_revision"
)

func validateErrorOutputSchema(schema *ValueSchema) error {
	if schema == nil {
		return fmt.Errorf("error output schema is nil")
	}
	if err := ValidateOutputContract(OperationOutputContract{ResultSchema: schema}); err != nil {
		return err
	}
	return rejectLiveErrorFieldsInSchema(*schema, "result")
}

func rejectLiveErrorFieldsInSchema(schema ValueSchema, path string) error {
	switch schema.Kind {
	case SchemaObject:
		for name, property := range schema.Properties {
			if name == errorContinuationField || name == errorRevisionField {
				return fmt.Errorf("error output schema contains reserved live field %q at %s", name, path)
			}
			if err := rejectLiveErrorFieldsInSchema(property, path+"."+name); err != nil {
				return err
			}
		}
	case SchemaArray:
		return rejectLiveErrorFieldsInSchema(*schema.Items, path+"[]")
	default:
		return nil
	}
	return nil
}

func rejectLiveErrorFields(value any, path string) error {
	switch current := value.(type) {
	case map[string]any:
		for name, nested := range current {
			if name == errorContinuationField || name == errorRevisionField {
				return fmt.Errorf("error output contains reserved live field %q at %s", name, path)
			}
			if err := rejectLiveErrorFields(nested, path+"."+name); err != nil {
				return err
			}
		}
	case []any:
		for index, nested := range current {
			if err := rejectLiveErrorFields(nested, fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
		}
	}
	return nil
}

func normalizeAndValidateOutput(value any, contract *OperationOutputContract) (any, error) {
	if contract == nil || contract.ResultSchema == nil {
		return nil, fmt.Errorf("typed output has no result schema: %w", ErrInvalidDefinition)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode projected result: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var normalized any
	if err := decoder.Decode(&normalized); err != nil {
		return nil, fmt.Errorf("decode projected result: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("projected result has trailing JSON value")
		}
		return nil, fmt.Errorf("check projected result suffix: %w", err)
	}
	if err := ValidateValue(*contract.ResultSchema, normalized); err != nil {
		return nil, fmt.Errorf("projected result violates declared schema: %w", err)
	}
	return normalized, nil
}

// SchemaAtResultPath resolves one declared result-schema path, including array item markers.
func SchemaAtResultPath(schema ValueSchema, path string) (ValueSchema, error) {
	if !strings.HasPrefix(path, "result.") {
		return ValueSchema{}, fmt.Errorf("result continuation path %q must start with result", path)
	}
	remainder := strings.TrimPrefix(path, "result.")
	if remainder == "" {
		return ValueSchema{}, fmt.Errorf("result continuation path %q has no property", path)
	}
	parts := strings.Split(remainder, ".")
	current := schema
	for _, part := range parts {
		if part == "" {
			return ValueSchema{}, fmt.Errorf("result continuation path %q has an empty segment", path)
		}
		arrayItem := strings.HasSuffix(part, "[]")
		name := strings.TrimSuffix(part, "[]")
		if name == "" || strings.ContainsAny(name, "[]") {
			return ValueSchema{}, fmt.Errorf("result continuation path segment %q is malformed", part)
		}
		if current.Kind != SchemaObject {
			return ValueSchema{}, fmt.Errorf("path segment %q traverses non-object schema", part)
		}
		property, ok := current.Properties[name]
		if !ok {
			return ValueSchema{}, fmt.Errorf("path segment %q is absent", part)
		}
		current = property
		if arrayItem {
			if current.Kind != SchemaArray || current.Items == nil {
				return ValueSchema{}, fmt.Errorf("path segment %q does not identify an array", part)
			}
			current = *current.Items
		}
	}
	return current, nil
}

// SchemasEquivalent reports whether two schemas describe the same accepted values.
func SchemasEquivalent(first, second ValueSchema) bool {
	return schemaAssignable(first, second) && schemaAssignable(second, first)
}

func schemaAssignable(source, target ValueSchema) bool {
	if source.Kind != target.Kind || source.Nullable && !target.Nullable {
		return false
	}
	if !schemaEnumAssignable(source.Enum, target.Enum) {
		return false
	}
	switch source.Kind {
	case SchemaBoolean:
		return true
	case SchemaString:
		return integerBoundsAssignable(source.MinLength, source.MaxLength, target.MinLength, target.MaxLength)
	case SchemaInteger, SchemaNumber:
		return numberBoundsAssignable(source, target)
	case SchemaArray:
		return arraySchemaAssignable(source, target)
	case SchemaObject:
		return objectSchemaAssignable(source, target)
	}
	return true
}

func schemaEnumAssignable(source, target []any) bool {
	if len(target) == 0 {
		return true
	}
	if len(source) == 0 {
		return false
	}
	for _, value := range source {
		if !slices.ContainsFunc(target, func(candidate any) bool { return reflect.DeepEqual(value, candidate) }) {
			return false
		}
	}
	return true
}

func integerBoundsAssignable(sourceMin, sourceMax, targetMin, targetMax *int) bool {
	if targetMin != nil && (sourceMin == nil || *sourceMin < *targetMin) {
		return false
	}
	if targetMax != nil && (sourceMax == nil || *sourceMax > *targetMax) {
		return false
	}
	return true
}

func numberBoundsAssignable(source, target ValueSchema) bool {
	if target.Minimum != nil && (source.Minimum == nil || *source.Minimum < *target.Minimum) {
		return false
	}
	if target.Maximum != nil && (source.Maximum == nil || *source.Maximum > *target.Maximum) {
		return false
	}
	return true
}

func arraySchemaAssignable(source, target ValueSchema) bool {
	return source.Items != nil && target.Items != nil &&
		integerBoundsAssignable(source.MinItems, source.MaxItems, target.MinItems, target.MaxItems) &&
		schemaAssignable(*source.Items, *target.Items)
}

func objectSchemaAssignable(source, target ValueSchema) bool {
	if !requiredPropertiesAssignable(source.Required, target.Required) {
		return false
	}
	for name, sourceProperty := range source.Properties {
		targetProperty, ok := target.Properties[name]
		if !ok || !schemaAssignable(sourceProperty, targetProperty) {
			return false
		}
	}
	return true
}

func requiredPropertiesAssignable(source, target []string) bool {
	for _, required := range target {
		if !slices.Contains(source, required) {
			return false
		}
	}
	return true
}

// SchemaAssignable reports whether every value accepted by source is accepted by target.
func SchemaAssignable(source, target ValueSchema) bool {
	if ValidateSchema(source) != nil || ValidateSchema(target) != nil {
		return false
	}
	return schemaAssignable(source, target)
}

// ProjectedContinuationFields extracts and validates declared top-level continuation aliases.
func ProjectedContinuationFields(projected any, contract *OperationOutputContract) (map[string]any, error) {
	fields := make(map[string]any)
	if contract == nil {
		return fields, nil
	}
	object, _ := projected.(map[string]any)
	for _, continuation := range contract.Continuations {
		if continuation.Field == "" {
			continue
		}
		value, exists := object[continuation.Field]
		if !exists {
			if continuation.Optional {
				continue
			}
			return nil, fmt.Errorf("projected result omits declared continuation %q", continuation.Field)
		}
		if err := ValidateValue(continuation.Schema, value); err != nil {
			return nil, fmt.Errorf("projected continuation %q violates its declared schema: %w", continuation.Field, err)
		}
		fields[continuation.Field] = value
	}
	return fields, nil
}
