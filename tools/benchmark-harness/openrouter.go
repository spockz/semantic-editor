package main

// OpenRouterFreeModel records one pinned catalog entry from the OpenRouter free-model collection.
type OpenRouterFreeModel struct {
	Rank             int
	ID               string
	Name             string
	ProgrammingRank  int
	SupportsToolCall bool
	SourceURL        string
}

// openRouterFreeCatalog is a dated, coding-oriented snapshot. Explicit model IDs remain overridable with --target.
var openRouterFreeCatalog = []OpenRouterFreeModel{
	{Rank: 1, ID: "nvidia/nemotron-3-ultra-550b-a55b:free", Name: "NVIDIA Nemotron 3 Ultra", ProgrammingRank: 18, SupportsToolCall: true, SourceURL: "https://openrouter.ai/nvidia/nemotron-3-ultra-550b-a55b:free"},
	{Rank: 2, ID: "poolside/laguna-s-2.1:free", Name: "Poolside Laguna S 2.1", ProgrammingRank: 45, SupportsToolCall: true, SourceURL: "https://openrouter.ai/poolside/laguna-s-2.1:free"},
	{Rank: 3, ID: "inclusionai/ling-3.0-flash-fin:free", Name: "InclusionAI Ling 3.0 Flash Fin", ProgrammingRank: 32, SupportsToolCall: true, SourceURL: "https://openrouter.ai/inclusionai/ling-3.0-flash-fin:free"},
	{Rank: 4, ID: "dots-studio/dots-3-note-preview:free", Name: "Dots3 Note Preview", ProgrammingRank: 44, SupportsToolCall: true, SourceURL: "https://openrouter.ai/dots-studio/dots-3-note-preview:free"},
	{Rank: 5, ID: "nvidia/nemotron-3.5-lightning:free", Name: "NVIDIA Nemotron 3.5 Lightning", ProgrammingRank: 39, SupportsToolCall: true, SourceURL: "https://openrouter.ai/nvidia/nemotron-3.5-lightning:free"},
	{Rank: 6, ID: "inclusionai/ling-3.0-flash-vl:free", Name: "InclusionAI Ling 3.0 Flash VL", ProgrammingRank: 47, SupportsToolCall: true, SourceURL: "https://openrouter.ai/inclusionai/ling-3.0-flash-vl:free"},
	{Rank: 7, ID: "nex-agi/nex-n2.5-pro:free", Name: "Nex-N2.5-Pro", SupportsToolCall: true, SourceURL: "https://openrouter.ai/nex-agi/nex-n2.5-pro:free"},
	{Rank: 8, ID: "thinkingmachines/inkling:free", Name: "Thinking Machines Inkling", SupportsToolCall: true, SourceURL: "https://openrouter.ai/thinkingmachines/inkling:free"},
	{Rank: 9, ID: "nvidia/nemotron-3-super-120b-a12b:free", Name: "NVIDIA Nemotron 3 Super", SupportsToolCall: true, SourceURL: "https://openrouter.ai/nvidia/nemotron-3-super-120b-a12b:free"},
	{Rank: 10, ID: "cohere/north-mini-code:free", Name: "Cohere North Mini Code", SupportsToolCall: true, SourceURL: "https://openrouter.ai/cohere/north-mini-code:free"},
}

const openRouterFreeCatalogAsOf = "2026-09-23"
const openRouterFreeCatalogURL = "https://openrouter.ai/collections/free-models/"

func openRouterTopTargets() []Target {
	targets := make([]Target, 0, len(openRouterFreeCatalog))
	for _, model := range openRouterFreeCatalog {
		targets = append(targets, Target{Harness: string(HarnessOpenCode), Model: "openrouter/" + model.ID})
	}
	return targets
}
