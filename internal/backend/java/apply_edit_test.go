// WHY: Pins Java reverse-RPC acknowledgement to successful application by the active workspace-edit consumer.
package java

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"semedit/internal/backend"
	"semedit/internal/lsp"
)

func TestJavaWorkspaceApplyEditAcknowledgesOnlyAcceptedEdits(t *testing.T) {
	edit := json.RawMessage(`{"changes":{}}`)
	for _, testCase := range []struct {
		name       string
		apply      func(json.RawMessage) (bool, error)
		wantApply  bool
		wantCalled bool
	}{
		{
			name: "accepted",
			apply: func(got json.RawMessage) (bool, error) {
				if string(got) != string(edit) {
					t.Errorf("edit payload = %s, want %s", got, edit)
				}
				return true, nil
			},
			wantApply:  true,
			wantCalled: true,
		},
		{
			name:       "rejected",
			apply:      func(json.RawMessage) (bool, error) { return false, nil },
			wantCalled: true,
		},
		{
			name:       "failed",
			apply:      func(json.RawMessage) (bool, error) { return true, errors.New("stale edit") },
			wantCalled: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			called := false
			session := &javaProcessSession{applyEditHandler: func(raw json.RawMessage) (bool, error) {
				called = true
				return testCase.apply(raw)
			}}
			params, err := json.Marshal(map[string]any{"edit": edit})
			if err != nil {
				t.Fatal(err)
			}
			response, rpcErr := session.handleServerRequest(context.Background(), lsp.Request{Method: "workspace/applyEdit", Params: params})
			if rpcErr != nil {
				t.Fatalf("handle applyEdit: %v", rpcErr)
			}
			var result struct {
				Applied       bool   `json:"applied"`
				FailureReason string `json:"failureReason"`
			}
			if err := json.Unmarshal(response, &result); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if result.Applied != testCase.wantApply || called != testCase.wantCalled {
				t.Fatalf("response applied=%t callback called=%t, want applied=%t called=%t", result.Applied, called, testCase.wantApply, testCase.wantCalled)
			}
			if testCase.name == "failed" && result.FailureReason != "stale edit" {
				t.Fatalf("failure reason = %q, want stale edit", result.FailureReason)
			}
		})
	}
}

func TestJavaWorkspaceApplyEditWithoutConsumerIsRejected(t *testing.T) {
	session := &javaProcessSession{}
	response, rpcErr := session.handleServerRequest(context.Background(), lsp.Request{
		Method: "workspace/applyEdit",
		Params: json.RawMessage(`{"edit":{"changes":{}}}`),
	})
	if rpcErr != nil {
		t.Fatalf("handle applyEdit: %v", rpcErr)
	}
	var result struct {
		Applied bool `json:"applied"`
	}
	if err := json.Unmarshal(response, &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Applied {
		t.Fatal("applyEdit without a consumer was acknowledged")
	}
}

func TestJavaWorkspaceConfigurationReturnsRequestedSettings(t *testing.T) {
	session := &javaProcessSession{settings: javaJDTLSSettings(backend.JavaConfig{ImportMaven: true})}
	response, rpcErr := session.handleServerRequest(context.Background(), lsp.Request{
		Method: "workspace/configuration",
		Params: json.RawMessage(`{"items":[{"section":"java"},{"section":"java.import.maven.enabled"},{"section":"unknown"}]}`),
	})
	if rpcErr != nil {
		t.Fatalf("handle workspace/configuration: %v", rpcErr)
	}
	var values []any
	if err := json.Unmarshal(response, &values); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(values) != 3 || values[0] == nil || values[1] != true || values[2] != nil {
		t.Fatalf("configuration response = %#v", values)
	}
}
