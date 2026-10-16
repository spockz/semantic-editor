# Standard Go Makefile for semedit
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
ifeq ($(shell uname -s),Darwin)
export DEVELOPER_DIR ?= /Library/Developer/CommandLineTools
endif
HUGO_BASE_URL ?= /
DOCGEN_ARGS ?=
GO_SCRATCH := $(CURDIR)/.scratch/go
export GOENV := $(GO_SCRATCH)/env
export GOCACHE := $(GO_SCRATCH)/build
export GOMODCACHE := $(GO_SCRATCH)/mod
export GOTMPDIR := $(GO_SCRATCH)/tmp
export GOBIN := $(GO_SCRATCH)/bin
export GOPATH := $(GO_SCRATCH)
export GOPLSCACHE := $(GO_SCRATCH)/gopls
export GOFLAGS :=
export GOWORK := off
export GOLANGCI_LINT_CACHE := $(CURDIR)/.scratch/golangci-lint-cache
export HUGO_CACHEDIR := $(CURDIR)/.scratch/hugo-cache
GOLANGCI_LINT_VERSION ?= v2.14.0
GOLANGCI_GO_VERSION := $(shell GOMODCACHE="$(GOMODCACHE)" go env GOVERSION)
GOLANGCI_LINT = $(GOBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)-$(GOLANGCI_GO_VERSION)
PACKAGE_CLOSURE_PLUGIN = $(GOBIN)/packageclosure.so
MARKDOWNLINT_CLI2_VERSION ?= 0.23.2
VALE_VERSION ?= v3.22.0
GOPLS_VERSION ?= v0.23.0
# GNU Make 3.81 gives this flag precedence over -jN; set MAKE_JOBS=N to override.
# Set it only in the top-level process so recursive make shares its jobserver.
MAKE_JOBS ?= 12
ifeq ($(MAKELEVEL),0)
MAKEFLAGS += -j$(MAKE_JOBS)
endif

ifeq ($(MAKELEVEL),0)
export PATH := $(CURDIR)/.scratch/tools/node_modules/.bin:$(GOBIN):$(PATH)
endif

.DEFAULT_GOAL := check

## Go commands use project-local state so caches and temporary files stay under the workspace.
.PHONY: go-env
go-env:
	@mkdir -p "$(GOCACHE)" "$(GOMODCACHE)" "$(GOTMPDIR)" "$(GOBIN)" "$(GOPLSCACHE)" "$(GOLANGCI_LINT_CACHE)" "$(HUGO_CACHEDIR)" "$(CURDIR)/.scratch/tools"

## ---------------------------------------------------------
## Check targets
## ---------------------------------------------------------
.PHONY: check
check: lint-check test-core ## Run fast iterative checks (linting and core engine tests)

.PHONY: check-ci
check-ci: lint vuln test-all verify-docs ## Run comprehensive checks (format, tidy, full linters, security, all tests, docs)

## Go source reader targets wait for source formatting and dependency updates; markdown and vale linting run independently.
lint-go vuln test test-core test-harness test-all build-next: tidy

## ---------------------------------------------------------
## Dependencies & Tooling
## ---------------------------------------------------------
.PHONY: deps
deps: go-env ## Download and verify module dependencies
	@echo "==> Downloading Go dependencies..."
	go mod download
	go mod verify

.PHONY: tools
tools: go-env ## Install development tools (linters, formatters, scanners)
	@echo "==> Installing development tools..."
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install mvdan.cc/gofumpt@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install golang.org/x/tools/gopls@$(GOPLS_VERSION)
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/vale-cli/vale/v3/cmd/vale@$(VALE_VERSION)
	@if command -v npm >/dev/null 2>&1; then \
		npm install --prefix "$(CURDIR)/.scratch/tools" --no-save --package-lock=false --no-audit --no-fund markdownlint-cli2@$(MARKDOWNLINT_CLI2_VERSION); \
	else \
		echo "npm not installed. Install Node.js to install markdownlint-cli2 locally."; \
		exit 1; \
	fi

## ---------------------------------------------------------
## Formatting, Refactoring & Imports
## ---------------------------------------------------------
.PHONY: fix
fix: lint-fix ## Apply automatic Go and Markdown fixes and standard library modernizations
	@echo "==> Running go fix..."
	go fix ./...
	@$(MAKE) fmt-go

.PHONY: lint-fix
lint-fix: go-env fix-markdown build-golangci-lint ## Apply unambiguous fixes supplied by configured Go and Markdown linters
	@echo "==> Applying Go linter fixes..."
	"$(GOLANGCI_LINT)" run --enable-only packageclosure --issues-exit-code=0 ./...
	"$(GOLANGCI_LINT)" run --disable=packageclosure --fix ./...

.PHONY: fmt
fmt: fix ## Format Go source code and optimize imports

.PHONY: fmt-go
fmt-go: go-env ## Format Go source code and optimize imports without lint fixes
	@echo "==> Formatting code..."
	@if command -v gofumpt >/dev/null 2>&1; then \
		find . -type d \( -path ./data/benchmarks -o -name .scratch -o -name .git -o -name vendor -o -name node_modules -o -name testdata \) -prune -o -type f -name '*.go' -print0 | xargs -0 gofumpt -w; \
	else \
		go fmt ./...; \
	fi
	@if command -v goimports >/dev/null 2>&1; then \
		if ! module_path=$$(go list -m); then \
			echo "Unable to determine the Go module for goimports." >&2; \
			exit 1; \
		fi; \
		if [ -z "$$module_path" ]; then \
			echo "Go module lookup returned an empty module path." >&2; \
			exit 1; \
		fi; \
		find . -type d \( -path ./data/benchmarks -o -name .scratch -o -name .git -o -name vendor -o -name node_modules \) -prune -o -type f -name '*.go' -execdir goimports -w -local "$$module_path" {} +; \
	fi

.PHONY: tidy
tidy: fmt ## Ensure dependencies match source code
	@echo "==> Tidying go.mod and go.sum..."
	go mod tidy

## ---------------------------------------------------------
## Linting and Static Analysis
## ---------------------------------------------------------
.PHONY: lint
lint: lint-go lint-markdown lint-vale ## Run all static linters (golangci-lint, markdownlint-cli2, vale)

MARKDOWN_GLOBS := "docs/**/*.md" "skills/**/*.md" "tools/**/*.md" "*.md"
VALE_TARGETS := docs skills tools *.md

.PHONY: lint-check
lint-check: go-env build-golangci-lint ## Run configured Go, Markdown, and Vale linters with automated fixes
	@echo "==> Reporting package closure candidates (advisory)..."
	"$(GOLANGCI_LINT)" run --enable-only packageclosure --issues-exit-code=0 ./...
	@echo "==> Running blocking Go lint with fixes..."
	"$(GOLANGCI_LINT)" run --disable=packageclosure --fix ./...
	@echo "==> Running Markdown lint with fixes..."
	@if command -v markdownlint-cli2 >/dev/null 2>&1; then \
		changed_md=$$(git status --porcelain=v1 -- '*.md' 2>/dev/null | awk '$$1 !~ /^D/ {print $$2}' | while read -r f; do \
			if [[ "$$f" != data/benchmarks/results/* && "$$f" != data/benchmarks/archive/* && "$$f" != data/benchmarks/fixture-validation/* ]]; then [ -f "$$f" ] && echo "$$f"; fi; \
		done || true); \
		if [ -n "$$changed_md" ] && [ "$${CI:-}" != "true" ]; then \
			echo "$$changed_md" | xargs markdownlint-cli2 --fix; \
		else \
			markdownlint-cli2 --fix $(MARKDOWN_GLOBS); \
		fi; \
	else \
		echo "markdownlint-cli2 not installed. Run 'make tools'."; \
		exit 1; \
	fi
	@echo "==> Running read-only Vale lint..."
	@if command -v vale >/dev/null 2>&1; then \
		changed_md=$$(git status --porcelain=v1 -- '*.md' 2>/dev/null | awk '$$1 !~ /^D/ {print $$2}' | while read -r f; do \
			if [[ "$$f" != data/benchmarks/results/* && "$$f" != data/benchmarks/archive/* && "$$f" != data/benchmarks/fixture-validation/* ]]; then [ -f "$$f" ] && echo "$$f"; fi; \
		done || true); \
		if [ -n "$$changed_md" ] && [ "$${CI:-}" != "true" ]; then \
			echo "$$changed_md" | xargs vale; \
		else \
			vale $(VALE_TARGETS); \
		fi; \
	else \
		echo "vale not installed. Run 'make tools'."; \
		exit 1; \
	fi

.PHONY: check-clean
check-clean: check ## Run checks and fail if any tracked, staged, or untracked files remain
	@status=$$(git status --porcelain=v1 --untracked-files=all -- 2>&1) || { \
		status_code=$$?; \
		printf 'git status failed (%s): %s\n' "$$status_code" "$$status" >&2; \
		exit "$$status_code"; \
	}; \
	if [[ -n "$$status" ]]; then \
		printf 'working tree is not clean after checks:\n%s\n' "$$status" >&2; \
		exit 1; \
	fi

.PHONY: lint-go
lint-go: go-env build-golangci-lint ## Build and run pinned golangci-lint with the native package-closure plugin
	@echo "==> Reporting package closure candidates (advisory)..."
	"$(GOLANGCI_LINT)" run --enable-only packageclosure --issues-exit-code=0 ./...
	@echo "==> Running golangci-lint..."
	"$(GOLANGCI_LINT)" run --disable=packageclosure ./...

PACKAGE_CLOSURE_SOURCES := $(filter-out %_test.go,$(wildcard tools/package-closure/*.go tools/package-closure/plugin/*.go tools/package-closure/go.mod tools/package-closure/go.sum))

.PHONY: build-golangci-lint
build-golangci-lint: go-env $(GOLANGCI_LINT) $(PACKAGE_CLOSURE_PLUGIN) ## Build golangci-lint and its native package-closure plugin

$(GOLANGCI_LINT):
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	mv "$(GOBIN)/golangci-lint" "$@"

$(PACKAGE_CLOSURE_PLUGIN): $(GOLANGCI_LINT) $(PACKAGE_CLOSURE_SOURCES)
	cd tools/package-closure && go build -buildmode=plugin -o "$(PACKAGE_CLOSURE_PLUGIN)" ./plugin

bin/fatalorder: tools/linters/fatalorder/*.go tools/linters/fatalorder/cmd/fatalorder/*.go | go-env
	@mkdir -p bin
	go build -o bin/fatalorder ./tools/linters/fatalorder/cmd/fatalorder

.PHONY: lint-fatalorder
lint-fatalorder: bin/fatalorder go-env ## Run fatalorder static analysis on Go test packages
	@echo "==> Running fatalorder static analysis..."
	go vet -vettool=./bin/fatalorder ./...

.PHONY: lint-markdown
lint-markdown: ## Run markdownlint-cli2
	@echo "==> Running markdownlint-cli2..."
	@if command -v markdownlint-cli2 >/dev/null 2>&1; then \
		markdownlint-cli2 $(MARKDOWN_GLOBS); \
	else \
		echo "markdownlint-cli2 not installed. Run 'make tools'."; \
		exit 1; \
	fi

.PHONY: fix-markdown
fix-markdown: ## Apply automatic markdownlint-cli2 fixes
	@echo "==> Applying Markdown linter fixes..."
	@if command -v markdownlint-cli2 >/dev/null 2>&1; then \
		markdownlint-cli2 --fix $(MARKDOWN_GLOBS); \
	else \
		echo "markdownlint-cli2 not installed. Run 'make tools'."; \
		exit 1; \
	fi

.PHONY: lint-vale
lint-vale: ## Run vale prose linter
	@echo "==> Running vale..."
	@if command -v vale >/dev/null 2>&1; then \
		vale $(VALE_TARGETS); \
	else \
		echo "vale not installed. Run 'make tools'."; \
		exit 1; \
	fi

.PHONY: vuln
vuln: go-env ## Run vulnerability security check
	@echo "==> Running govulncheck..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not installed (optional). Run 'make tools' or: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi

## ---------------------------------------------------------
## Testing
## ---------------------------------------------------------
.PHONY: test test-core
test: test-core ## Alias for test-core

test-core: go-env test-package-closure ## Run core engine unit tests excluding benchmark-harness
	@echo "==> Running core engine tests..."
	@CORE_PACKAGES=$$(go list ./... | grep -v 'github.com/spockz/semantic-editor/tools/benchmark-harness'); \
	go test $$CORE_PACKAGES

.PHONY: test-package-closure
test-package-closure: go-env ## Test the custom package-closure linter module
	cd tools/package-closure && go test ./...

.PHONY: test-race
test-race: go-env ## Run core engine unit tests with race detector
	@echo "==> Running core engine tests with race detector..."
	@CORE_PACKAGES=$$(go list ./... | grep -v 'github.com/spockz/semantic-editor/tools/benchmark-harness'); \
	go test -race -shuffle=on -cover -v $$CORE_PACKAGES

.PHONY: test-harness
test-harness: go-env ## Run benchmark harness tests
	@echo "==> Running benchmark harness tests..."
	go test -shuffle=on -cover ./tools/benchmark-harness/...

.PHONY: test-all
test-all: go-env ## Run all unit and race tests including benchmark-harness
	@echo "==> Running full test suite with race detector..."
	go test -race -shuffle=on -cover -v ./...

.PHONY: bench-mcp-tools
COLD_SAMPLES ?= 100
HOT_SAMPLES ?= 1000
TOOL_FILTER ?=
ENFORCE_BUDGETS ?= false
TOOL_SMOKE ?= false
CORPUS ?= 20
bench-mcp-tools: go-env ## Report cold first-call and retained-session p99 by semantic MCP tool
	@case "$(ENFORCE_BUDGETS)" in true|false) ;; *) echo "ENFORCE_BUDGETS must be true or false" >&2; exit 2;; esac
	@case "$(TOOL_SMOKE)" in true|false) ;; *) echo "TOOL_SMOKE must be true or false" >&2; exit 2;; esac
	@case "$(CORPUS)" in 20|repository) ;; *) echo "CORPUS must be 20 or repository" >&2; exit 2;; esac
	@echo "==> Cold/hot MCP samples: cold=$(COLD_SAMPLES), hot=$(HOT_SAMPLES), filter=$(TOOL_FILTER)"
	SEMEDIT_MCP_TOOL_COSTS=1 SEMEDIT_MCP_COST_CORPUS="$(CORPUS)" SEMEDIT_MCP_COLD_SAMPLES="$(COLD_SAMPLES)" SEMEDIT_MCP_HOT_SAMPLES="$(HOT_SAMPLES)" SEMEDIT_MCP_TOOL_FILTER="$(TOOL_FILTER)" SEMEDIT_MCP_COST_ENFORCE_BUDGETS="$(if $(filter true,$(ENFORCE_BUDGETS)),1,0)" SEMEDIT_MCP_TOOL_SMOKE="$(if $(filter true,$(TOOL_SMOKE)),1,0)" go test -v ./internal/mcp -run '^$$' -bench '^BenchmarkMCPColdHot(ToolCosts|FileEdits)$$' -benchtime=1x -count=1 -timeout=0

.PHONY: bench-mcp-gopls
GOPLS_CALL_SAMPLES ?= 10
GOPLS_TRACE ?=
GOPLS_CPU_PROFILE ?=
bench-mcp-gopls: go-env ## Attribute warm edits on the synthetic and repository corpora
	go test -v ./internal/mcp -run '^$$' -bench '^BenchmarkMCPGoplsCallAttribution$$' -benchtime="$(GOPLS_CALL_SAMPLES)x" -count=1 -timeout=10m $(if $(strip $(GOPLS_TRACE)),-trace="$(GOPLS_TRACE)") $(if $(strip $(GOPLS_CPU_PROFILE)),-cpuprofile="$(GOPLS_CPU_PROFILE)" -o .scratch/mcp-gopls-profile.test)

.PHONY: test-short
test-short: go-env ## Run short test suite without long-running tests
	@echo "==> Running short tests..."
	go test -short ./...

.PHONY: test-property-full
test-property-full: go-env ## Run comprehensive 100-check property tests
	@echo "==> Running full property tests (100 checks)..."
	RAPID_CHECKS=100 go test -race -shuffle=on -v -run TestProperty_ ./internal/adapters/golang/...

.PHONY: test-fuzz
test-fuzz: go-env ## Run deep 1000-check metamorphic property fuzzing
	@echo "==> Running deep property fuzzing (1000 checks)..."
	RAPID_CHECKS=1000 go test -race -shuffle=on -v -run TestProperty_ ./internal/adapters/golang/...

## ---------------------------------------------------------
## Build and Clean
## ---------------------------------------------------------
.PHONY: build-next
build-next: go-env ## Build the active development binary (bin/semedit-next)
	@echo "==> Building bin/semedit-next..."
	@mkdir -p bin
	go build -o bin/semedit-next ./cmd/semedit

.PHONY: promote
promote: check build-next ## Verify checks and promote bin/semedit-next to healthy bin/semedit
	@echo "==> Promoting semedit-next to semedit..."
	@mkdir -p bin
	cp bin/semedit-next bin/semedit.tmp && mv -f bin/semedit.tmp bin/semedit
	@echo "Successfully promoted healthy semedit binary"

.PHONY: build
build: build-next ## Build development and stable binaries
	@echo "==> Ensuring bin/semedit exists..."
	@if [ ! -f bin/semedit ]; then \
		rm -f bin/semedit; \
		cp bin/semedit-next bin/semedit; \
	fi

.PHONY: install-mcp
install-mcp: build ## Install development MCP configs with absolute binary paths
	go run ./tools/mcp-config --workspace "$(CURDIR)"

.PHONY: check-hugo-version
check-hugo-version: ## Verify installed Hugo matches .hugo-version
	@if command -v hugo >/dev/null 2>&1; then \
		EXPECTED=$$(tr -d ' \t\r\n' < .hugo-version 2>/dev/null || echo ""); \
		INSTALLED=$$(hugo version 2>/dev/null | sed -E 's/.*v([0-9]+\.[0-9]+\.[0-9]+).*/\1/'); \
		if [ -n "$$EXPECTED" ] && [ "$$EXPECTED" != "$$INSTALLED" ]; then \
			echo "Hugo version mismatch: installed $$INSTALLED, expected $$EXPECTED from .hugo-version"; \
			echo "Run 'make sync-hugo-version' to update .hugo-version or align your installed Hugo."; \
			exit 1; \
		fi; \
	else \
		echo "Hugo is not installed. Install Hugo Extended to build documentation."; \
		exit 1; \
	fi

.PHONY: sync-hugo-version
sync-hugo-version: ## Update .hugo-version to match installed Hugo version
	@if command -v hugo >/dev/null 2>&1; then \
		INSTALLED=$$(hugo version 2>/dev/null | sed -E 's/.*v([0-9]+\.[0-9]+\.[0-9]+).*/\1/'); \
		echo "$$INSTALLED" > .hugo-version; \
		echo "Synced .hugo-version to $$INSTALLED"; \
	else \
		echo "Hugo is not installed."; \
		exit 1; \
	fi

.PHONY: docgen
docgen: check-hugo-version docgen-source ## Generate the documentation site into dist/docs
	@echo "==> Building Hugo documentation to dist/docs..."
	@hugo --source .scratch/docgen --destination "$(CURDIR)/dist/docs" --baseURL "$(HUGO_BASE_URL)" --cleanDestinationDir --minify
	@touch dist/docs/.nojekyll

.PHONY: docgen-source
docgen-source: go-env ## Generate temporary Hugo Markdown and module source in .scratch/docgen
	@echo "==> Generating Hugo documentation source in .scratch/docgen..."
	@mkdir -p .scratch/docgen
	go run ./cmd/docgen --output-dir .scratch/docgen $(DOCGEN_ARGS)

.PHONY: docs
docs: docgen ## Alias for docgen

.PHONY: verify-docs
verify-docs: docgen ## Generate the site and assert its published files exist
	@echo "==> Verifying generated documentation output..."
	@test -f "$(CURDIR)/dist/docs/index.html"
	@test -f "$(CURDIR)/dist/docs/docs/index.html"
	@test -f "$(CURDIR)/dist/docs/docs/getting-started/index.html"
	@test -f "$(CURDIR)/dist/docs/docs/reference/index.html"
	@test -f "$(CURDIR)/dist/docs/docs/benchmarks/index.html"
	@test -f "$(CURDIR)/dist/docs/docs/benchmarks/aggregates/index.html"
	@test -f "$(CURDIR)/dist/docs/.nojekyll"

## ---------------------------------------------------------
## Benchmark Harness
## ---------------------------------------------------------
-include tools/benchmark-harness/Makefile

.PHONY: list-clean-worktrees clean-clean-worktrees
list-clean-worktrees: go-env ## List worktrees that are clean and integrated into local main
	@go run ./cmd/worktreecheck

clean-clean-worktrees: go-env ## Remove clean worktrees integrated into local main
	@go run ./cmd/worktreecheck --remove-done

.PHONY: clean
clean: go-env ## Clean build artifacts and test cache
	@echo "==> Cleaning cache..."
	go clean -testcache
	rm -rf bin/ dist/

.PHONY: help
help: ## Display this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
