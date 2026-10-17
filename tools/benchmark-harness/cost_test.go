// cost_test.go verifies the benchmark's cloud token pricing equations and coverage boundaries.
package main

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestEstimateOpenAICost(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		effort    string
		want      float64
		pricingID string
	}{
		{"GPT-6 Astra", "gpt-6-astra", "low", 2.05, "openai-api-gpt-6-astra-standard-short-context"},
		{"GPT-6 Sol", "gpt-6-sol", "medium", 0.41, "openai-api-gpt-6-sol-standard-short-context"},
		{"GPT-6.1 Sol", "gpt-6.1-sol", "low", 0.405, "openai-api-gpt-6.1-sol-standard-short-context"},
		{"GPT-5.6 alias routes to Sol", "gpt-5.6", "high", 0.82, "openai-api-gpt-5.6-sol-standard-short-context"},
		{"GPT-5.6 Terra", "gpt-5.6-terra", "medium", 0.45, "openai-api-gpt-5.6-terra-standard-short-context"},
		{"GPT-5.6 Luna", "gpt-5.6-luna", "low", 0.045, "openai-api-gpt-5.6-luna-standard-short-context"},
		{"GPT-5.6 Cyber", "gpt-5.6-cyber", "low", 2.8125, "openai-api-gpt-5.6-cyber-standard-short-context"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target := Target{Harness: string(HarnessCodex), Model: test.model, Effort: test.effort}
			got := estimateCloudCost(target, 150_000, 50_000, 100_000, 20_000, 5_000)
			if got == nil {
				t.Fatal("estimateCloudCost returned nil")
			}
			if math.Abs(got.TotalUSD-test.want) > 1e-12 {
				t.Errorf("TotalUSD = %.9f, want %.9f", got.TotalUSD, test.want)
			}
			if got.PricingID != test.pricingID {
				t.Errorf("PricingID = %q, want %q", got.PricingID, test.pricingID)
			}
			if got.Currency != "USD" || got.RateCardAsOf != costRateCardAsOf {
				t.Errorf("estimate metadata = currency %q, as-of %q", got.Currency, got.RateCardAsOf)
			}
		})
	}
}

func TestEstimateGPT6LunaRateSharedAcrossEffortLevels(t *testing.T) {
	for _, effort := range []string{"medium", "high"} {
		t.Run(effort, func(t *testing.T) {
			got := estimateCloudCost(Target{Harness: string(HarnessCodex), Model: "gpt-6-luna", Effort: effort}, 150_000, 50_000, 100_000, 20_000, 5_000)
			if got == nil || math.Abs(got.TotalUSD-0.0205) > 1e-12 {
				t.Errorf("estimate = %#v, want USD 0.020500", got)
			}
			if got != nil && got.PricingID != "openai-api-gpt-6-luna-standard-short-context" {
				t.Errorf("PricingID = %q, want GPT-6 Luna rate card", got.PricingID)
			}
		})
	}
}

func TestEstimateGeminiCost(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		want   float64
	}{
		{name: "Gemini 3.7 Flash low includes thinking tokens", target: Target{Harness: string(HarnessAgy), Model: "gemini-3.7-flash-low"}, want: 0.014625},
		{name: "Gemini 3.8 Flash medium", target: Target{Harness: string(HarnessAgy), Model: "gemini-3.8-flash-medium"}, want: 0.014625},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := estimateCloudCost(test.target, 30_000, 20_000, 10_000, 1_000, 500)
			if got == nil || math.Abs(got.TotalUSD-test.want) > 1e-12 {
				t.Errorf("estimate = %#v, want USD %.9f", got, test.want)
			}
		})
	}
}

func TestEstimateCloudCostLeavesUnpricedOrIncompleteUsageUnavailable(t *testing.T) {
	for _, test := range []struct {
		name   string
		target Target
		prompt int
		cache  int
		input  int
	}{
		{name: "local model has user-dependent capex", target: Target{Harness: string(HarnessOpenCode), Model: "local-saluki"}, prompt: 100, input: 100},
		{name: "unknown cloud model", target: Target{Harness: string(HarnessAgy), Model: "gemini-4.0-flash-low"}, prompt: 100, input: 100},
		{name: "usage counters do not reconcile", target: Target{Harness: string(HarnessCodex), Model: "gpt-6-luna"}, prompt: 100, input: 99},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := estimateCloudCost(test.target, test.prompt, test.cache, test.input, 1, 0); got != nil {
				t.Errorf("estimateCloudCost = %#v, want unavailable", got)
			}
		})
	}
}

func TestFormatCloudCostComparisonRow(t *testing.T) {
	baselineCost, semeditCost := 0.02, 0.01
	baseline := &RunResult{EstimatedCostUSD: &baselineCost}
	semedit := &RunResult{EstimatedCostUSD: &semeditCost}
	row := formatCloudCostComparisonRow(baseline, semedit, nil, nil)
	for _, want := range []string{"Estimated Cloud Token Cost", "$0.020000", "$0.010000", "-50.0%"} {
		if !strings.Contains(row, want) {
			t.Errorf("cost row %q does not contain %q", row, want)
		}
	}
}

func TestRunResultSerializesEstablishedCostFieldWithRateMetadata(t *testing.T) {
	cost := 0.00003
	run := RunResult{
		EstimatedCostUSD:          &cost,
		EstimatedCostPricingID:    "google-gemini-api-3.8-flash-global-standard-introductory",
		EstimatedCostRateCardAsOf: "2026-10-17",
		EstimatedCostBasis:        "global Standard text token rates",
	}
	raw, err := json.Marshal(run)
	if err != nil {
		t.Fatalf("marshal RunResult: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("decode RunResult: %v", err)
	}
	if fields["estimated_cost_usd"] != cost || fields["estimated_cost_pricing_id"] != run.EstimatedCostPricingID || fields["estimated_cost_rate_card_as_of"] != run.EstimatedCostRateCardAsOf || fields["estimated_cost_basis"] != run.EstimatedCostBasis {
		t.Errorf("serialized cost fields = %#v", fields)
	}
	if _, ok := fields["cloud_cost_estimate"]; ok {
		t.Fatal("serialized duplicate cloud_cost_estimate object")
	}
}
