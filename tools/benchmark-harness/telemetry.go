// Shared tool telemetry normalization belongs here because every harness uses the same measurement semantics.

package main

import (
	"encoding/json"
	"strings"
	"time"
)

func appendToolCall(res *RunResult, call ToolCall) int {
	res.ToolCalls = append(res.ToolCalls, call)
	res.ToolsUsed = append(res.ToolsUsed, call.Name)
	return len(res.ToolCalls) - 1
}

func refreshToolCallCounts(res *RunResult) {
	res.ToolCount = len(res.ToolCalls)
	res.FailedToolCallCount = 0
	res.FailedSemanticToolCallCount = 0
	for _, call := range res.ToolCalls {
		failed := call.TransportStatus == ToolCallStatusFailed || call.FunctionalStatus == ToolCallStatusFailed
		if !failed {
			continue
		}
		res.FailedToolCallCount++
		if isSemanticTool(call.Name, call.Server) {
			res.FailedSemanticToolCallCount++
		}
	}
}

func applyMCPStartupMetrics(res *RunResult, metrics *MCPStartupMetrics) {
	if metrics == nil {
		return
	}
	if res.MCPServerStartToInitialize == nil {
		value := time.Duration(metrics.ServerStartToInitializeMS) * time.Millisecond
		res.MCPServerStartToInitialize = &value
	}
	if res.MCPInitializeToFirstSemanticCall == nil {
		value := time.Duration(metrics.InitializeToFirstSemanticCallMS) * time.Millisecond
		res.MCPInitializeToFirstSemanticCall = &value
	}
}

func refreshMCPVerified(res *RunResult) {
	res.MCPVerified = false
	for _, call := range res.ToolCalls {
		if isSemanticTool(call.Name, call.Server) && call.TransportStatus == ToolCallStatusSucceeded {
			res.MCPVerified = true
			return
		}
	}
}

func isSemanticTool(name, server string) bool {
	cleanServer := strings.ToLower(strings.Trim(server, "\""))
	return cleanServer == "semedit" || strings.HasPrefix(semanticToolBaseName(name), "semantic_")
}

func hasJSONValue(raw json.RawMessage) bool {
	value := strings.TrimSpace(string(raw))
	return value != "" && value != "null"
}

func jsonToolFailure(raw json.RawMessage) string {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return compactToolFailure(string(raw))
	}
	return compactToolFailure(toolFailureText(value))
}

func toolFailureText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		for _, item := range typed {
			if text := toolFailureText(item); text != "" {
				return text
			}
		}
	case map[string]any:
		for _, key := range []string{"error", "message", "text", "content", "result"} {
			if item, ok := typed[key]; ok {
				if text := toolFailureText(item); text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func compactToolFailure(message string) string {
	message = strings.Join(strings.Fields(message), " ")
	if len(message) > 512 {
		return message[:512]
	}
	return message
}

func isMutatingTool(name string) bool {
	clean := strings.ToLower(strings.Trim(name, "\""))
	if strings.Contains(clean, "write") || strings.Contains(clean, "replace") ||
		strings.Contains(clean, "edit") || strings.Contains(clean, "patch") ||
		strings.Contains(clean, "insert") || strings.Contains(clean, "delete") ||
		strings.Contains(clean, "rename") || strings.Contains(clean, "semantic_") ||
		clean == codexFileChangeItemType {
		return true
	}
	return false
}

func isMCPDiscoveryTool(name string) bool {
	clean := strings.ToLower(strings.Trim(name, "\""))
	return strings.Contains(clean, "mcp") ||
		strings.Contains(clean, "list_resources") ||
		strings.Contains(clean, "read_resource") ||
		strings.Contains(clean, "get_server_capabilities") ||
		strings.Contains(clean, "describe")
}
