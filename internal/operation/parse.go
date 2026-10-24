// Package operation coerces ingress-neutral parameter maps into typed operation requests.
package operation

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strconv"
	"strings"
)

// ErrInvalidParams indicates a raw parameter map violates its parameter contracts.
var ErrInvalidParams = errors.New("invalid operation parameters")

func rawLookup(raw map[string]any, jsonName, cliName string) (any, bool) {
	if raw == nil {
		return nil, false
	}
	if value, ok := raw[jsonName]; ok {
		return value, true
	}
	if cliName != "" && cliName != jsonName {
		if value, ok := raw[cliName]; ok {
			return value, true
		}
	}
	return nil, false
}

// CheckParams validates required presence and enum membership for every contract.
// Empty strings count as absent so optional enum parameters accept the zero value.
func CheckParams(raw map[string]any, params []ParameterContract) error {
	if err := rejectUnknownParams(raw, params); err != nil {
		return err
	}
	for _, param := range params {
		if err := validateParameter(raw, param); err != nil {
			return err
		}
	}
	return nil
}

func parameterKeys(params []ParameterContract) map[string]struct{} {
	allowed := make(map[string]struct{}, len(params)*2)
	for _, param := range params {
		if param.JSONName != "" {
			allowed[param.JSONName] = struct{}{}
		}
		if param.CLIName != "" {
			allowed[param.CLIName] = struct{}{}
		}
	}
	return allowed
}

func rejectUnknownParams(raw map[string]any, params []ParameterContract) error {
	allowed := parameterKeys(params)
	unknown := make([]string, 0)
	for key := range raw {
		if _, ok := allowed[key]; !ok {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	slices.Sort(unknown)
	supported := make([]string, 0, len(allowed))
	for key := range allowed {
		supported = append(supported, key)
	}
	slices.Sort(supported)

	message := fmt.Sprintf("unknown parameter %q; supported keys are [%s]", unknown[0], strings.Join(supported, ", "))
	required := make([]string, 0, len(params))
	for _, param := range params {
		value, present := rawLookup(raw, param.JSONName, param.CLIName)
		if param.Required && param.JSONName != "" && (!present || value == nil || !nonEmptyInputValue(value)) {
			required = append(required, param.JSONName)
		}
	}
	slices.Sort(required)
	if len(required) > 0 {
		message += fmt.Sprintf(". Required parameters: [%s]", strings.Join(required, ", "))
	}
	return fmt.Errorf("%s: %w", message, ErrInvalidParams)
}
func validateParameter(raw map[string]any, param ParameterContract) error {
	value, present := rawLookup(raw, param.JSONName, param.CLIName)
	if param.Schema != nil {
		return validateSchemaParameter(param, value, present)
	}
	if !present || value == nil {
		return requiredParameterError(param, param.Required)
	}
	if param.Type == ParamInteger {
		if _, err := exactNativeInteger(value); err != nil {
			return fmt.Errorf("param %q must be an integer: %w", param.JSONName, ErrInvalidParams)
		}
	}
	text, isText := value.(string)
	if isText && strings.TrimSpace(text) == "" {
		return requiredParameterError(param, param.Required)
	}
	return validateEnumParameter(param, text, isText)
}

func validateSchemaParameter(param ParameterContract, value any, present bool) error {
	if !present {
		return requiredParameterError(param, param.Required)
	}
	if err := ValidateValue(*param.Schema, value); err != nil {
		return fmt.Errorf("param %q: %w: %w", param.JSONName, err, ErrInvalidParams)
	}
	return nil
}

func requiredParameterError(param ParameterContract, required bool) error {
	if required {
		return fmt.Errorf("param %q is required: %w", param.JSONName, ErrInvalidParams)
	}
	return nil
}

func validateEnumParameter(param ParameterContract, text string, isText bool) error {
	if len(param.Enums) == 0 {
		return nil
	}
	if !isText {
		return fmt.Errorf("param %q must be a string: %w", param.JSONName, ErrInvalidParams)
	}
	if slices.Contains(param.Enums, text) {
		return nil
	}
	return fmt.Errorf("param %q value %q must be one of [%s]: %w", param.JSONName, text, strings.Join(param.Enums, ", "), ErrInvalidParams)
}

// ParseString extracts an optional or required string parameter.
func ParseString(raw map[string]any, jsonName, cliName string, required bool) (string, error) {
	value, present := rawLookup(raw, jsonName, cliName)
	if !present || value == nil {
		if required {
			return "", fmt.Errorf("param %q is required: %w", jsonName, ErrInvalidParams)
		}
		return "", nil
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("param %q must be a string: %w", jsonName, ErrInvalidParams)
	}
	if required && strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("param %q is required: %w", jsonName, ErrInvalidParams)
	}
	return text, nil
}

// ParseStringDefault extracts an optional string parameter, substituting def for absent or blank values.
func ParseStringDefault(raw map[string]any, jsonName, cliName, def string) (string, error) {
	text, err := ParseString(raw, jsonName, cliName, false)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(text) == "" {
		return def, nil
	}
	return text, nil
}

// ParseBool extracts a boolean parameter, accepting JSON booleans, numeric flags, and CLI strings.
func ParseBool(raw map[string]any, jsonName, cliName string, def bool) (bool, error) {
	value, present := rawLookup(raw, jsonName, cliName)
	if !present || value == nil {
		return def, nil
	}
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes":
			return true, nil
		case "false", "0", "no", "":
			return false, nil
		default:
			return false, fmt.Errorf("param %q value %q is not a boolean: %w", jsonName, typed, ErrInvalidParams)
		}
	case float64:
		switch typed {
		case 1:
			return true, nil
		case 0:
			return false, nil
		default:
			return false, fmt.Errorf("param %q value %v is not a boolean: %w", jsonName, typed, ErrInvalidParams)
		}
	default:
		return false, fmt.Errorf("param %q must be a boolean: %w", jsonName, ErrInvalidParams)
	}
}

// ParseInteger extracts a required JSON or CLI integer parameter.
func ParseInteger(raw map[string]any, jsonName, cliName string, required bool) (int, error) {
	value, present := rawLookup(raw, jsonName, cliName)
	if !present || value == nil {
		if required {
			return 0, fmt.Errorf("param %q is required: %w", jsonName, ErrInvalidParams)
		}
		return 0, nil
	}
	parsed, err := exactNativeInteger(value)
	if err != nil {
		return 0, fmt.Errorf("param %q must be an integer: %w", jsonName, ErrInvalidParams)
	}
	return parsed, nil
}

// ParseStringSlice extracts a string slice, accepting JSON arrays, repeated values, and single strings.
func ParseStringSlice(raw map[string]any, jsonName, cliName string) ([]string, error) {
	value, present := rawLookup(raw, jsonName, cliName)
	if !present || value == nil {
		return nil, nil
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), nil
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("param %q must be an array of strings: %w", jsonName, ErrInvalidParams)
			}
			result = append(result, text)
		}
		return result, nil
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil, nil
		}
		parts := strings.Split(typed, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("param %q must be an array of strings: %w", jsonName, ErrInvalidParams)
	}
}

// ParseEnum extracts a string parameter constrained to an allowed set, substituting def when absent.
func ParseEnum(raw map[string]any, jsonName, cliName string, allowed []string, required bool, def string) (string, error) {
	text, err := ParseString(raw, jsonName, cliName, required)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(text) == "" {
		if required && def == "" {
			return "", fmt.Errorf("param %q is required: %w", jsonName, ErrInvalidParams)
		}
		return def, nil
	}
	if slices.Contains(allowed, text) {
		return text, nil
	}
	return "", fmt.Errorf("param %q value %q must be one of [%s]: %w",
		jsonName, text, strings.Join(allowed, ", "), ErrInvalidParams)
}

func exactNativeInteger(value any) (int, error) {
	switch typed := value.(type) {
	case int:
		return typed, nil
	case int64:
		if int64(int(typed)) != typed {
			return 0, fmt.Errorf("integer is outside native int range")
		}
		return int(typed), nil
	case float64:
		rational := new(big.Rat).SetFloat64(typed)
		if rational == nil || !rational.IsInt() {
			return 0, fmt.Errorf("number is not an exact integer")
		}
		parsed, err := strconv.ParseInt(rational.Num().String(), 10, strconv.IntSize)
		if err != nil {
			return 0, fmt.Errorf("integer is outside native int range: %w", err)
		}
		return int(parsed), nil
	case json.Number:
		rational, ok := boundedJSONNumber(typed)
		if !ok || !rational.IsInt() {
			return 0, fmt.Errorf("number is not an exact integer")
		}
		parsed, err := strconv.ParseInt(rational.Num().String(), 10, strconv.IntSize)
		if err != nil {
			return 0, fmt.Errorf("integer is outside native int range: %w", err)
		}
		return int(parsed), nil
	default:
		return 0, fmt.Errorf("value %T is not numeric", value)
	}
}
