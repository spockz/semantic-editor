# Benchmark harness targets for semedit
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

REPO_ROOT ?= $(shell git rev-parse --show-toplevel 2>/dev/null || pwd)
BENCH_DIR ?= $(REPO_ROOT)/testdata/bench
OUT_DIR ?= $(REPO_ROOT)/data/benchmarks/results

TARGET ?= codex/gpt-5.6-luna/medium
VARIANTS ?= small,large
CONCURRENCY ?= 2
MCP_SERVER_INSTRUCTIONS ?= none
# PROVENANCE records technical execution context without creating comparison cells.
PROVENANCE ?=
# CLASSIFIERS is retained as a compatibility alias for PROVENANCE.
CLASSIFIERS ?=
TASK ?= task-01-rename-local

ifndef .DEFAULT_GOAL
.DEFAULT_GOAL := bench-help
endif

.PHONY: bench-help
bench-help: ## Show benchmark harness commands and usage examples
	@echo "Benchmark Harness Commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' "$(REPO_ROOT)/tools/benchmark-harness/Makefile" | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make bench TASK=generate_template_main"
	@echo "  make bench TASK=task-01-rename-local TARGET=codex/gpt-5.6-luna/medium"
	@echo "  make bench-all"
	@echo "  make docgen"

.PHONY: list-benchmarks bench-list
list-benchmarks: ## List all available benchmark tasks and prompt variants
	cd "$(REPO_ROOT)" && go run ./tools/benchmark-harness list --dir "$(BENCH_DIR)"

bench-list: list-benchmarks ## Alias for list-benchmarks

.PHONY: bench-test
bench-test: ## Run unit and oracle tests for benchmark harness
	@echo "==> Testing benchmark harness..."
	cd "$(REPO_ROOT)" && go test -v ./tools/benchmark-harness

.PHONY: bench-build
bench-build: ## Compile standalone benchmark harness binary (.scratch/bin/benchmark-harness)
	@echo "==> Building benchmark harness..."
	@mkdir -p "$(REPO_ROOT)/.scratch/bin"
	cd "$(REPO_ROOT)" && go build -o "$(REPO_ROOT)/.scratch/bin/benchmark-harness" ./tools/benchmark-harness

ifneq ($(CURDIR),$(REPO_ROOT))
.PHONY: test build
test: bench-test ## Run unit tests for benchmark harness
build: bench-build ## Build benchmark harness binary
endif

.PHONY: bench-all
bench-all: ## Run all benchmarks in small and large context with all prompt variants
	@echo "==> Running all benchmarks for target $(TARGET) across variants $(VARIANTS)..."
	cd "$(REPO_ROOT)" && go run ./tools/benchmark-harness \
		--matrix \
		--target "$(TARGET)" \
		--tasks all \
		--variants "$(VARIANTS)" \
		--concurrency "$(CONCURRENCY)" \
		--mcp-server-instructions "$(MCP_SERVER_INSTRUCTIONS)" \
		$(foreach provenance,$(PROVENANCE),--provenance "$(provenance)") \
		$(foreach classifier,$(CLASSIFIERS),--classifier "$(classifier)") \
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
		--mcp-server-instructions "$(MCP_SERVER_INSTRUCTIONS)" \
		$(foreach provenance,$(PROVENANCE),--provenance "$(provenance)") \
		$(foreach classifier,$(CLASSIFIERS),--classifier "$(classifier)") \
		--dir "$(BENCH_DIR)" \
		--out-dir "$(OUT_DIR)"
