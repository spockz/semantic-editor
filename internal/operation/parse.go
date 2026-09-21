// Package operation coerces ingress-neutral parameter maps into typed operation requests.
package operation

import (
	"errors"
	"fmt"
	"slices"
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
	for _, param := range params {
		value, present := rawLookup(raw, param.JSONName, param.CLIName)
		if !present || value == nil {
			if param.Required {
				return fmt.Errorf("param %q is required: %w", param.JSONName, ErrInvalidParams)
			}
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			if param.Required {
				return fmt.Errorf("param %q is required: %w", param.JSONName, ErrInvalidParams)
			}
			continue
		}
		if len(param.Enums) == 0 {
			continue
		}
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("param %q must be a string: %w", param.JSONName, ErrInvalidParams)
		}
		matched := slices.Contains(param.Enums, text)
		if !matched {
			return fmt.Errorf("param %q value %q must be one of [%s]: %w",
				param.JSONName, text, strings.Join(param.Enums, ", "), ErrInvalidParams)
		}
	}
	return nil
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
