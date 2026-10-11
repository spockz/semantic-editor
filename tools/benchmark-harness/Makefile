# Benchmark harness targets for semedit
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

REPO_ROOT ?= $(shell git rev-parse --show-toplevel 2>/dev/null)
ifeq ($(strip $(REPO_ROOT)),)
$(error cannot determine repository root; run in a Git checkout or set REPO_ROOT)
endif
GO_SCRATCH ?= $(REPO_ROOT)/.scratch/go
export GOENV := $(GO_SCRATCH)/env
export GOCACHE := $(GO_SCRATCH)/build
export GOMODCACHE := $(GO_SCRATCH)/mod
export GOTMPDIR := $(GO_SCRATCH)/tmp
export GOBIN := $(GO_SCRATCH)/bin
export GOPATH := $(GO_SCRATCH)
export GOPLSCACHE := $(GO_SCRATCH)/gopls
export GOFLAGS :=
export GOWORK := off
export GOLANGCI_LINT_CACHE := $(REPO_ROOT)/.scratch/golangci-lint-cache
ifneq ($(CURDIR),$(REPO_ROOT))
ifeq ($(MAKELEVEL),0)
export PATH := $(REPO_ROOT)/.scratch/tools/node_modules/.bin:$(GOBIN):$(PATH)
endif
endif
BENCH_DIR ?= $(REPO_ROOT)/testdata/bench
OUT_DIR ?= $(REPO_ROOT)/data/benchmarks/results
BENCH_RUN_STAMP := $(shell date -u '+%Y%m%dT%H%M%SZ')
RUN_ID ?= run-$(BENCH_RUN_STAMP)
RESUME ?= false
SEMEDIT_SKILL ?= false

TARGET ?= codex/gpt-5.6-luna/medium
VARIANTS ?= small,large
CONCURRENCY ?= 2
REPEATS ?= 1
TIMEOUT ?= 5m
MCP_SERVER_INSTRUCTIONS ?= prescriptive
# MCP_SERVER_INSTRUCTIONS accepts a comma-separated subset of none, descriptive, and prescriptive.
SEMEDIT_ARM_RESTRICT ?= write
SEMEDIT_PREWARM_VERIFY ?= true
# PROVENANCE records technical execution context without creating comparison cells.
PROVENANCE ?=
# CLASSIFIERS is retained as a compatibility alias for PROVENANCE.
CLASSIFIERS ?=
TASK ?=
TASKS ?=

ifndef .DEFAULT_GOAL
.DEFAULT_GOAL := bench-help
endif

.PHONY: bench-go-env
bench-go-env:
	@mkdir -p "$(GOCACHE)" "$(GOMODCACHE)" "$(GOTMPDIR)" "$(GOBIN)" "$(GOPLSCACHE)" "$(GOLANGCI_LINT_CACHE)"

.PHONY: bench-help
bench-help: ## Show benchmark harness commands and usage examples
	@echo "Benchmark Harness Commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' "$(REPO_ROOT)/tools/benchmark-harness/Makefile" | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make bench TASK=generate_template_main"
	@echo "  make bench TASK=task-01-rename-local TARGET=codex/gpt-5.6-luna/medium"
	@echo "  make bench TASK=task-01-rename-local TARGET=opencode/openrouter/free-top10"
	@echo "  make bench TASK=task-01-rename-local TARGET=opencode/openrouter/cohere/north-mini-code:free REPEATS=3"
	@echo "  make bench-all TARGET=opencode/openrouter/free-top10 REPEATS=2"
	@echo "  make bench-all RUN_ID=my-run RESUME=true"
	@echo "  make bench-smoke-hard"
	@echo "  make bench-full-hard REPEATS=10"
	@echo "  make bench-smoke-swe"
	@echo "  make bench-full-swe REPEATS=10"
	@echo "  make bench-smoke-saluki"
	@echo "  make bench-full-saluki REPEATS=10"
	@echo "  make bench TASK=task-01-rename-local SEMEDIT_ARM_RESTRICT=readwrite"
	@echo "  make bench TASK=grpc__grpc-go-2951 SEMEDIT_SKILL=true"
	@echo "  make bench-smoke-swe SEMEDIT_SKILL=true"
	@echo "  make bench-smoke-swe SEMEDIT_SKILL=true TASKS=grpc__grpc-go-3361"
	@echo "  SEMEDIT_ARM_RESTRICT accepts read, write (default), or readwrite"
	@echo "  SEMEDIT_SKILL controls optional skill delivery to both semedit arms; baseline is unchanged"
	@echo "  MCP_SERVER_INSTRUCTIONS accepts a comma-separated subset of none, descriptive, and prescriptive"
	@echo "  go run ./tools/benchmark-harness --list-openrouter-free"
	@echo "  make bench-all"
	@echo "  make docgen"

.PHONY: list-benchmarks bench-list
list-benchmarks: bench-go-env ## List all available benchmark tasks and prompt variants
	cd "$(REPO_ROOT)" && go run ./tools/benchmark-harness --list --dir "$(BENCH_DIR)"

bench-list: list-benchmarks ## Alias for list-benchmarks

.PHONY: bench-test
bench-test: bench-go-env ## Run unit and oracle tests for benchmark harness
	@echo "==> Testing benchmark harness..."
	cd "$(REPO_ROOT)" && go test -v ./tools/benchmark-harness

.PHONY: bench-build
bench-build: bench-go-env ## Compile standalone benchmark harness binary (.scratch/bin/benchmark-harness)
	@echo "==> Building benchmark harness..."
	@mkdir -p "$(REPO_ROOT)/.scratch/bin"
	cd "$(REPO_ROOT)" && go build -o "$(REPO_ROOT)/.scratch/bin/benchmark-harness" ./tools/benchmark-harness

ifneq ($(CURDIR),$(REPO_ROOT))
.PHONY: test build
test: bench-test ## Run unit tests for benchmark harness
build: bench-build ## Build benchmark harness binary
endif

.PHONY: bench-all
bench-all: bench-go-env ## Run all benchmarks in small and large context with all prompt variants
	@echo "==> Running all benchmarks for target $(TARGET) across variants $(VARIANTS)..."
	cd "$(REPO_ROOT)" && go run ./tools/benchmark-harness \
		--target "$(TARGET)" \
		--tasks all \
		--variants "$(VARIANTS)" \
		--concurrency "$(CONCURRENCY)" \
		--repeats "$(REPEATS)" \
		--timeout "$(TIMEOUT)" \
		--mcp-server-instructions "$(MCP_SERVER_INSTRUCTIONS)" \
		--semedit-arm-restrict "$(SEMEDIT_ARM_RESTRICT)" \
		--semedit-skill="$(SEMEDIT_SKILL)" \
		--semedit-prewarm-verify="$(SEMEDIT_PREWARM_VERIFY)" \
		$(foreach provenance,$(PROVENANCE),--provenance "$(provenance)") \
		$(foreach classifier,$(CLASSIFIERS),--classifier "$(classifier)") \
		--dir "$(BENCH_DIR)" \
		--out-dir "$(OUT_DIR)" \
		--run-id "$(RUN_ID)" \
		$(if $(filter true,$(RESUME)),--resume)

.PHONY: bench bench-smoke-hard bench-full-hard bench-smoke-swe bench-full-swe bench-smoke-saluki bench-full-saluki
bench: ## Run one or more selected benchmarks (e.g. make bench TASK=generate_template_main)
bench-smoke-hard: ## Run the hard first-party smoke suite
bench-full-hard: ## Run the full hard first-party suite
bench-smoke-swe: ## Run the Multi-SWE smoke suite
bench-full-swe: ## Run the full Multi-SWE suite
bench-smoke-saluki: ## Run the Saluki local-model smoke suite
bench-full-saluki: ## Run the full Saluki local-model suite

bench bench-smoke-hard bench-full-hard bench-smoke-swe bench-full-swe bench-smoke-saluki bench-full-saluki: bench-go-env
	@echo "==> Running benchmark $(if $(TASKS),$(TASKS),$(TASK)) for target $(TARGET) across variants $(VARIANTS)..."
	cd "$(REPO_ROOT)" && go run ./tools/benchmark-harness \
		--target "$(TARGET)" \
		$(if $(TASKS),--tasks "$(TASKS)",$(if $(TASK),--task "$(TASK)")) \
		--variants "$(VARIANTS)" \
		--concurrency "$(CONCURRENCY)" \
		--repeats "$(REPEATS)" \
		--timeout "$(TIMEOUT)" \
		--mcp-server-instructions "$(MCP_SERVER_INSTRUCTIONS)" \
		--semedit-arm-restrict "$(SEMEDIT_ARM_RESTRICT)" \
		--semedit-skill="$(SEMEDIT_SKILL)" \
		--semedit-prewarm-verify="$(SEMEDIT_PREWARM_VERIFY)" \
		$(foreach provenance,$(PROVENANCE),--provenance "$(provenance)") \
		$(foreach classifier,$(CLASSIFIERS),--classifier "$(classifier)") \
		--dir "$(BENCH_DIR)" \
		--out-dir "$(OUT_DIR)" \
		--run-id "$(RUN_ID)" \
		$(if $(filter true,$(RESUME)),--resume)

HARD_TASKS := task-09-composite-refactor,task-11-mixed-sink-api-migration,task-12-request-identity-migration
SWE_TASKS := grpc__grpc-go-2735,grpc__grpc-go-2951,grpc__grpc-go-3361,zeromicro__go-zero-2463,zeromicro__go-zero-964
SMOKE_TARGETS := codex/gpt-6-luna/medium
FULL_TARGETS := codex/gpt-6-luna/medium,codex/gpt-6-luna/high,agy/gemini-3.7-flash-low,agy/gemini-3.7-flash-medium
SALUKI_TARGET ?= opencode/amdbeast/Kujira/Underdog-Saluki-27B-1.0-MTP-GGUF:latest
SALUKI_TASKS ?= task-01-rename-local

bench-smoke-hard: TARGET := $(SMOKE_TARGETS)
bench-smoke-hard: VARIANTS := small:explicit
bench-smoke-hard: REPEATS := 1
bench-smoke-hard: TASKS := $(HARD_TASKS)
bench-smoke-hard: RUN_ID = $@-$(BENCH_RUN_STAMP)

bench-full-hard: TARGET := $(FULL_TARGETS)
bench-full-hard: VARIANTS := small:explicit
bench-full-hard: REPEATS := 10
bench-full-hard: TASKS := $(HARD_TASKS)
bench-full-hard: RUN_ID = $@-$(BENCH_RUN_STAMP)

bench-smoke-swe: TARGET := $(SMOKE_TARGETS)
bench-smoke-swe: VARIANTS := small:default
bench-smoke-swe: REPEATS := 1
bench-smoke-swe: TASKS := $(SWE_TASKS)
bench-smoke-swe: RUN_ID = $@-$(BENCH_RUN_STAMP)

bench-full-swe: TARGET := $(FULL_TARGETS)
bench-full-swe: VARIANTS := small:default
bench-full-swe: REPEATS := 10
bench-full-swe: TASKS := $(SWE_TASKS)
bench-full-swe: RUN_ID = $@-$(BENCH_RUN_STAMP)
bench-smoke-saluki: TARGET := $(SALUKI_TARGET)
bench-smoke-saluki: TASKS := $(SALUKI_TASKS)
bench-smoke-saluki: VARIANTS := small:explicit
bench-smoke-saluki: CONCURRENCY := 1
bench-smoke-saluki: REPEATS := 1
bench-smoke-saluki: RUN_ID = $@-$(BENCH_RUN_STAMP)

bench-full-saluki: TARGET := $(SALUKI_TARGET)
bench-full-saluki: TASKS := $(SALUKI_TASKS)
bench-full-saluki: VARIANTS := small:explicit
bench-full-saluki: CONCURRENCY := 1
bench-full-saluki: REPEATS := 10
bench-full-saluki: RUN_ID = $@-$(BENCH_RUN_STAMP)
