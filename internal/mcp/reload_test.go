// Package mcp_test validates live-reload flag behavior, schema advertisement, and reload tool presence/absence.
package mcp_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"semedit/internal/mcp"
)

func TestMCPLiveReloadCapabilities(t *testing.T) {
	t.Parallel()

	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}` + "\n"

	t.Run("default advertisement", func(t *testing.T) {
		t.Parallel()
		inBuf := bytes.NewBufferString(input)
		var outBuf bytes.Buffer

		srv := mcp.NewServer("full", ".", &outBuf)
		if err := srv.Serve(t.Context(), inBuf); err != nil {
			t.Fatalf("Serve failed: %v", err)
		}

		var initResp struct {
			ID     int `json:"id"`
			Result struct {
				Capabilities struct {
					Tools struct {
						ListChanged bool `json:"listChanged"`
					} `json:"tools"`
				} `json:"capabilities"`
			} `json:"result"`
		}
		if err := json.Unmarshal(outBuf.Bytes(), &initResp); err != nil {
			t.Fatalf("unmarshal init response: %v", err)
		}
		if !initResp.Result.Capabilities.Tools.ListChanged {
			t.Errorf("expected capabilities.tools.listChanged to be true, got false")
		}
	})
}

func TestMCPLiveReloadToolPresence(t *testing.T) {
	t.Parallel()

	input := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n"

	t.Run("disabled by default", func(t *testing.T) {
		t.Parallel()
		inBuf := bytes.NewBufferString(input)
		var outBuf bytes.Buffer

		srv := mcp.NewServer("full", ".", &outBuf)
		if err := srv.Serve(t.Context(), inBuf); err != nil {
			t.Fatalf("Serve failed: %v", err)
		}

		var toolsResp struct {
			Result struct {
				Tools []struct {
					Name string `json:"name"`
				} `json:"tools"`
			} `json:"result"`
		}
		if err := json.Unmarshal(outBuf.Bytes(), &toolsResp); err != nil {
			t.Fatalf("unmarshal tools response: %v", err)
		}

		for _, tool := range toolsResp.Result.Tools {
			if tool.Name == "semantic_reload" {
				t.Fatalf("semantic_reload tool should NOT be present when liveReload is disabled")
			}
		}
	})

	t.Run("enabled via option", func(t *testing.T) {
		t.Parallel()
		inBuf := bytes.NewBufferString(input)
		var outBuf bytes.Buffer

		srv := mcp.NewServer("full", ".", &outBuf, mcp.WithLiveReload(true))
		if err := srv.Serve(t.Context(), inBuf); err != nil {
			t.Fatalf("Serve failed: %v", err)
		}

		var toolsResp struct {
			Result struct {
				Tools []struct {
					Name        string         `json:"name"`
					Description string         `json:"description"`
					InputSchema map[string]any `json:"inputSchema"`
				} `json:"tools"`
			} `json:"result"`
		}
		if err := json.Unmarshal(outBuf.Bytes(), &toolsResp); err != nil {
			t.Fatalf("unmarshal tools response: %v", err)
		}

		var found bool
		for _, tool := range toolsResp.Result.Tools {
			if tool.Name == "semantic_reload" {
				found = true
				if tool.InputSchema["type"] != "object" {
					t.Errorf("expected type 'object', got %v", tool.InputSchema["type"])
				}
				if tool.InputSchema["additionalProperties"] != false {
					t.Errorf("expected additionalProperties false, got %v", tool.InputSchema["additionalProperties"])
				}
				break
			}
		}
		if !found {
			t.Fatalf("semantic_reload tool should be present when liveReload is enabled")
		}
	})
}

func TestMCPLiveReloadInitializedNotification(t *testing.T) {
	t.Parallel()

	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}` + "\n" + `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n"

	t.Run("live-reload enabled emits list_changed", func(t *testing.T) {
		t.Parallel()
		inBuf := bytes.NewBufferString(input)
		var outBuf bytes.Buffer

		srv := mcp.NewServer("full", ".", &outBuf, mcp.WithLiveReload(true))
		if err := srv.Serve(t.Context(), inBuf); err != nil {
			t.Fatalf("Serve failed: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(outBuf.String()), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 messages (init response + notification), got %d:\n%s", len(lines), outBuf.String())
		}

		var notif struct {
			JSONRPC string `json:"jsonrpc"`
			Method  string `json:"method"`
		}
		if err := json.Unmarshal([]byte(lines[1]), &notif); err != nil {
			t.Fatalf("unmarshal notification: %v", err)
		}
		if notif.Method != "notifications/tools/list_changed" {
			t.Errorf("expected notifications/tools/list_changed, got %q", notif.Method)
		}
	})

	t.Run("live-reload disabled does not emit list_changed", func(t *testing.T) {
		t.Parallel()
		inBuf := bytes.NewBufferString(input)
		var outBuf bytes.Buffer

		srv := mcp.NewServer("full", ".", &outBuf, mcp.WithLiveReload(false))
		if err := srv.Serve(t.Context(), inBuf); err != nil {
			t.Fatalf("Serve failed: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(outBuf.String()), "\n")
		if len(lines) != 1 {
			t.Fatalf("expected 1 message (init response only), got %d:\n%s", len(lines), outBuf.String())
		}
	})
}

func TestMCPLiveReloadToolCallWhenDisabled(t *testing.T) {
	t.Parallel()

	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"semantic_reload","arguments":{}}}` + "\n"
	inBuf := bytes.NewBufferString(input)
	var outBuf bytes.Buffer

	srv := mcp.NewServer("full", ".", &outBuf, mcp.WithLiveReload(false))
	if err := srv.Serve(t.Context(), inBuf); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	var resp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(outBuf.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal resp: %v", err)
	}
	if resp.Error == nil {
		t.Fatalf("expected error response when semantic_reload is called but liveReload is disabled, got nil")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("expected error code -32601, got %d", resp.Error.Code)
	}
}
