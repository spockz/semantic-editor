# Makefile coordinates empirical benchmark execution across targets, context scales, and prompt variants for semedit.
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

REPO_ROOT ?= $(shell git rev-parse --show-toplevel 2>/dev/null || pwd)
BENCH_DIR ?= $(REPO_ROOT)/testdata/bench
OUT_DIR ?= $(REPO_ROOT)/data/benchmarks/results

TARGET ?= agy/gemini-3.8-flash-low
VARIANTS ?= small,large
CONCURRENCY ?= 2
TASK ?= task-01-rename-local

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show available benchmark harness targets
	@echo "Benchmark Harness Commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  # Run a specific benchmark (small & large context, all prompt variants):"
	@echo "  make bench TASK=generate_template_main"
	@echo "  make bench TASK=task-01-rename-local TARGET=agy/gemini-3.8-flash-low"
	@echo ""
	@echo "  # Run all benchmarks across all targets and variants:"
	@echo "  make bench-all"
	@echo "  make bench-all TARGET=agy/gemini-3.8-flash-low CONCURRENCY=4"
	@echo ""
	@echo "  # Regenerate documentation with updated benchmark outcomes:"
	@echo "  make docgen"

.PHONY: list
list: ## List all available benchmark tasks and their prompt variants
	cd "$(REPO_ROOT)" && go run ./tools/benchmark-harness list --dir "$(BENCH_DIR)"

.PHONY: test
test: ## Run unit tests for benchmark harness
	@echo "==> Testing benchmark harness..."
	cd "$(REPO_ROOT)" && go test -v ./tools/benchmark-harness

.PHONY: build
build: ## Build benchmark harness executable
	@echo "==> Building benchmark harness..."
	cd "$(REPO_ROOT)" && go build -o "$(REPO_ROOT)/.scratch/bin/benchmark-harness" ./tools/benchmark-harness

.PHONY: bench-all
bench-all: ## Run all benchmarks in small and large context with all prompt variants
	@echo "==> Running all benchmarks for target $(TARGET) across variants $(VARIANTS)..."
	cd "$(REPO_ROOT)" && go run ./tools/benchmark-harness \
		--matrix \
		--target "$(TARGET)" \
		--tasks all \
		--variants "$(VARIANTS)" \
		--concurrency "$(CONCURRENCY)" \
		--dir "$(BENCH_DIR)" \
		--out-dir "$(OUT_DIR)"

.PHONY: bench
bench: ## Run a specific benchmark in small and large context (e.g. make bench TASK=generate_template_main)
	@if [ -z "$(TASK)" ]; then \
		echo "Error: TASK variable is required (e.g. make bench TASK=task-01-rename-local)"; \
		exit 1; \
	fi
	@echo "==> Running benchmark $(TASK) for target $(TARGET) across variants $(VARIANTS)..."
	cd "$(REPO_ROOT)" && go run ./tools/benchmark-harness \
		--matrix \
		--target "$(TARGET)" \
		--task "$(TASK)" \
		--variants "$(VARIANTS)" \
		--concurrency "$(CONCURRENCY)" \
		--dir "$(BENCH_DIR)" \
		--out-dir "$(OUT_DIR)"

.PHONY: docgen
docgen: ## Regenerate Hugo documentation site with recorded benchmark outcomes
	@echo "==> Regenerating documentation with benchmark outcomes..."
	cd "$(REPO_ROOT)" && make docgen
