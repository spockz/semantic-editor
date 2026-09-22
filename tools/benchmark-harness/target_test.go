package main

import (
	"testing"
)

func TestParseTarget(t *testing.T) {
	tests := []struct {
		input   string
		want    Target
		wantErr bool
	}{
		{
			input: "codex",
			want:  Target{Harness: "codex"},
		},
		{
			input: "codex/gpt-5-mini",
			want:  Target{Harness: "codex", Model: "gpt-5-mini"},
		},
		{
			input: "codex/gpt-5-mini/low",
			want:  Target{Harness: "codex", Model: "gpt-5-mini", Effort: "low"},
		},
		{
			input: "agy/gemini-3.8-flash-low",
			want:  Target{Harness: "agy", Model: "gemini-3.8-flash-low"},
		},
		{
			input: "agy/gemini-3.8-flash-low/low",
			want:  Target{Harness: "agy", Model: "gemini-3.8-flash-low", Effort: "low"},
		},
		{
			input: "opencode/amdbeast/qwen36-coder",
			want:  Target{Harness: "opencode", Model: "amdbeast/qwen36-coder"},
		},
		{
			input: "opencode/amdbeast/qwen36-coder/high",
			want:  Target{Harness: "opencode", Model: "amdbeast/qwen36-coder", Effort: "high"},
		},
		{
			input:   "",
			wantErr: true,
		},
		{
			input:   "a/b/c/d",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		got, err := ParseTarget(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseTarget(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ParseTarget(%q) = %+v, want %+v", tt.input, got, tt.want)
		}
	}
}

func TestTargetListFlag(t *testing.T) {
	var tl TargetList
	if err := tl.Set("codex/gpt-5-mini/low,agy/gemini-3.8-flash-low"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if len(tl) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(tl))
	}
	if tl[0].String() != "codex/gpt-5-mini/low" {
		t.Errorf("expected codex/gpt-5-mini/low, got %s", tl[0].String())
	}
	if tl[1].String() != "agy/gemini-3.8-flash-low" {
		t.Errorf("expected agy/gemini-3.8-flash-low, got %s", tl[1].String())
	}
}
