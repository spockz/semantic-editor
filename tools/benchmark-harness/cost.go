// cost.go estimates comparable cloud token spend from provider usage and a dated public rate card.
package main

import (
	"strings"
	"time"
)

const costRateCardAsOf = "2026-10-17"

const tokensPerMillion = 1_000_000

type CloudCostEstimate struct {
	Currency         string  `json:"currency"`
	TotalUSD         float64 `json:"total_usd"`
	UncachedInputUSD float64 `json:"uncached_input_usd"`
	CachedInputUSD   float64 `json:"cached_input_usd"`
	OutputUSD        float64 `json:"output_usd"`
	PricingID        string  `json:"pricing_id"`
	RateCardAsOf     string  `json:"rate_card_as_of"`
	Basis            string  `json:"basis"`
}

type cloudTokenRates struct {
	pricingID               string
	uncachedInputPerMillion float64
	cachedInputPerMillion   float64
	outputPerMillion        float64
	thinkingIsSeparate      bool
	basis                   string
	rateCardAsOf            string
}

func estimateCloudCost(target Target, promptTokens, cachedPromptTokens, uncachedPromptTokens, outputTokens, reasoningTokens int) *CloudCostEstimate {
	return estimateCloudCostAt(target, promptTokens, cachedPromptTokens, uncachedPromptTokens, outputTokens, reasoningTokens, time.Now())
}

func estimateCloudCostAt(target Target, promptTokens, cachedPromptTokens, uncachedPromptTokens, outputTokens, reasoningTokens int, asOf time.Time) *CloudCostEstimate {
	rates, ok := cloudRates(target, asOf)
	if !ok || promptTokens <= 0 || outputTokens < 0 || reasoningTokens < 0 || cachedPromptTokens < 0 || uncachedPromptTokens < 0 {
		return nil
	}
	if cachedPromptTokens+uncachedPromptTokens != promptTokens {
		return nil
	}

	billedOutputTokens := outputTokens
	if rates.thinkingIsSeparate {
		billedOutputTokens += reasoningTokens
	}
	uncachedCost := float64(uncachedPromptTokens) * rates.uncachedInputPerMillion / tokensPerMillion
	cachedCost := float64(cachedPromptTokens) * rates.cachedInputPerMillion / tokensPerMillion
	outputCost := float64(billedOutputTokens) * rates.outputPerMillion / tokensPerMillion
	return &CloudCostEstimate{
		Currency:         "USD",
		TotalUSD:         uncachedCost + cachedCost + outputCost,
		UncachedInputUSD: uncachedCost,
		CachedInputUSD:   cachedCost,
		OutputUSD:        outputCost,
		PricingID:        rates.pricingID,
		RateCardAsOf:     rates.rateCardAsOf,
		Basis:            rates.basis,
	}
}

func cloudRates(target Target, asOf time.Time) (cloudTokenRates, bool) {
	model := strings.ToLower(strings.TrimSpace(target.Model))
	if target.Harness == string(HarnessCodex) {
		var rates cloudTokenRates
		switch model {
		case "gpt-6-astra":
			rates = cloudTokenRates{pricingID: "openai-api-gpt-6-astra-standard-short-context", uncachedInputPerMillion: 10.00, cachedInputPerMillion: 1.00, outputPerMillion: 50.00}
		case "gpt-6-sol":
			rates = cloudTokenRates{pricingID: "openai-api-gpt-6-sol-standard-short-context", uncachedInputPerMillion: 2.00, cachedInputPerMillion: 0.20, outputPerMillion: 10.00}
		case "gpt-6.1-sol":
			rates = cloudTokenRates{pricingID: "openai-api-gpt-6.1-sol-standard-short-context", uncachedInputPerMillion: 2.00, cachedInputPerMillion: 0.10, outputPerMillion: 10.00}
		case "gpt-6-luna":
			rates = cloudTokenRates{pricingID: "openai-api-gpt-6-luna-standard-short-context", uncachedInputPerMillion: 0.10, cachedInputPerMillion: 0.01, outputPerMillion: 0.50}
		case "gpt-5.6", "gpt-5.6-sol":
			rates = cloudTokenRates{pricingID: "openai-api-gpt-5.6-sol-standard-short-context", uncachedInputPerMillion: 4.00, cachedInputPerMillion: 0.40, outputPerMillion: 20.00}
		case "gpt-5.6-terra":
			rates = cloudTokenRates{pricingID: "openai-api-gpt-5.6-terra-standard-short-context", uncachedInputPerMillion: 2.00, cachedInputPerMillion: 0.20, outputPerMillion: 12.00}
		case "gpt-5.6-luna":
			rates = cloudTokenRates{pricingID: "openai-api-gpt-5.6-luna-standard-short-context", uncachedInputPerMillion: 0.20, cachedInputPerMillion: 0.02, outputPerMillion: 1.20}
		case "gpt-5.6-cyber":
			rates = cloudTokenRates{pricingID: "openai-api-gpt-5.6-cyber-standard-short-context", uncachedInputPerMillion: 12.50, cachedInputPerMillion: 1.25, outputPerMillion: 75.00}
		default:
			return cloudTokenRates{}, false
		}
		rates.basis = "OpenAI API Standard short-context text token list rates; reasoning tokens are included in output tokens"
		rates.rateCardAsOf = costRateCardAsOf
		return rates, true
	}
	if target.Harness != string(HarnessAgy) {
		return cloudTokenRates{}, false
	}
	model = strings.TrimSuffix(model, "-low")
	model = strings.TrimSuffix(model, "-medium")
	model = strings.TrimSuffix(model, "-high")
	switch model {
	case "gemini-4-argon":
		if asOf.Before(time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)) {
			return cloudTokenRates{
				pricingID:               "google-gemini-api-4-argon-global-standard-introductory",
				uncachedInputPerMillion: 2.00,
				cachedInputPerMillion:   0.10,
				outputPerMillion:        10.00,
				thinkingIsSeparate:      true,
				basis:                   "Gemini API global Standard introductory text token rates through 2026-11-30; thinking tokens are billed as output",
				rateCardAsOf:            costRateCardAsOf,
			}, true
		}
		return cloudTokenRates{
			pricingID:               "google-gemini-api-4-argon-global-standard",
			uncachedInputPerMillion: 4.00,
			cachedInputPerMillion:   0.20,
			outputPerMillion:        20.00,
			thinkingIsSeparate:      true,
			basis:                   "Gemini API global Standard text token rates from 2026-12-01; thinking tokens are billed as output",
			rateCardAsOf:            "2026-12-01",
		}, true
	case "gemini-3.7-flash", "gemini-3.8-flash":
		if asOf.Before(time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)) {
			return cloudTokenRates{
				pricingID:               "google-gemini-api-3.8-flash-global-standard-introductory",
				uncachedInputPerMillion: 0.75,
				cachedInputPerMillion:   0.075,
				outputPerMillion:        3.75,
				thinkingIsSeparate:      true,
				basis:                   "Gemini API global Standard text token introductory rates through 2026-12-31; thinking tokens are billed as output",
				rateCardAsOf:            costRateCardAsOf,
			}, true
		}
		return cloudTokenRates{
			pricingID:               "google-gemini-api-3.8-flash-global-standard",
			uncachedInputPerMillion: 1.50,
			cachedInputPerMillion:   0.15,
			outputPerMillion:        7.50,
			thinkingIsSeparate:      true,
			basis:                   "Gemini API global Standard text token rates from 2027-01-01; thinking tokens are billed as output",
			rateCardAsOf:            "2027-01-01",
		}, true
	default:
		return cloudTokenRates{}, false
	}
}
