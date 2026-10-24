// Package java implements native Java inferred-type expansion.
// WHY: This action accepts only JDT LS's eager, single-token native edit.
package java

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/javacst"
)

const javaMakeTypeExplicitOperation backend.Operation = "make_type_explicit"

func selectJavaExplicitTypeEdit(raw json.RawMessage, request backend.CodeActionRequest) (json.RawMessage, error) {
	var actions []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &actions); err != nil {
		return nil, fmt.Errorf("decode Java explicit-type actions: %w", err)
	}
	prefix := "Change type of '" + request.Symbol + "' to '"
	var matches []map[string]json.RawMessage
	for _, action := range actions {
		var title, kind string
		if err := json.Unmarshal(action["title"], &title); err != nil {
			return nil, fmt.Errorf("decode Java explicit-type action title: %w", err)
		}
		if err := json.Unmarshal(action["kind"], &kind); err != nil {
			continue
		}
		if kind == "refactor" && strings.HasPrefix(title, prefix) && strings.HasSuffix(title, "'") {
			matches = append(matches, action)
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("JDT LS explicit-type action requires exactly one matching action, found %d", len(matches))
	}
	action := matches[0]
	for _, key := range []string{"disabled", "command", "data"} {
		if _, present := action[key]; present {
			return nil, fmt.Errorf("JDT LS explicit-type action must be enabled and eager; %s is unsupported", key)
		}
	}
	var title string
	if err := json.Unmarshal(action["title"], &title); err != nil {
		return nil, fmt.Errorf("decode Java explicit-type action title: %w", err)
	}
	typeName := strings.TrimSuffix(strings.TrimPrefix(title, prefix), "'")
	if err := javacst.ValidateJavaExplicitType(typeName); err != nil {
		return nil, err
	}
	if len(action["edit"]) == 0 || string(action["edit"]) == "null" {
		return nil, fmt.Errorf("JDT LS explicit-type action requires a direct workspace edit")
	}
	var edit map[string]json.RawMessage
	if err := json.Unmarshal(action["edit"], &edit); err != nil {
		return nil, fmt.Errorf("decode Java explicit-type workspace edit: %w", err)
	}
	var changes map[string][]map[string]json.RawMessage
	if len(edit) != 1 || json.Unmarshal(edit["changes"], &changes) != nil || len(changes) != 1 {
		return nil, fmt.Errorf("JDT LS explicit-type action requires only one selected-file changes entry")
	}
	for _, edits := range changes {
		if len(edits) != 1 || len(edits[0]) != 2 {
			return nil, fmt.Errorf("JDT LS explicit-type action requires one unannotated text edit")
		}
		var replacement string
		if err := json.Unmarshal(edits[0]["newText"], &replacement); err != nil || replacement != typeName {
			return nil, fmt.Errorf("JDT LS explicit-type replacement must equal its native title type")
		}
		if _, present := edits[0]["range"]; !present {
			return nil, fmt.Errorf("JDT LS explicit-type edit requires a range")
		}
	}
	return action["edit"], nil
}

func validateJavaExplicitTypeEdit(source []byte, uri string, request backend.CodeActionRequest, raw json.RawMessage) error {
	local, err := javacst.JavaInferredLocalAt(source, request.Symbol, request.Range)
	if err != nil {
		return fmt.Errorf("resolve Java inferred local: %w", err)
	}
	var edit struct {
		Changes map[string][]struct {
			Range   backend.Range `json:"range"`
			NewText string        `json:"newText"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(raw, &edit); err != nil {
		return fmt.Errorf("decode Java explicit-type token edit: %w", err)
	}
	edits := edit.Changes[uri]
	if len(edit.Changes) != 1 || len(edits) != 1 || edits[0].Range != local.Type.Range {
		return fmt.Errorf("JDT LS explicit-type edit must replace exactly the selected local's var token")
	}
	return javacst.ValidateJavaExplicitType(edits[0].NewText)
}
