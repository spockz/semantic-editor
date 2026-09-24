# Standard Go Makefile for semedit
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
export DEVELOPER_DIR ?= /Library/Developer/CommandLineTools
HUGO_BASE_URL ?= /
ifeq ($(MAKELEVEL),0)
export PATH := $(CURDIR)/.scratch/go/bin:$(PATH)
endif
# GNU Make 3.81 gives this flag precedence over -jN; set MAKE_JOBS=N to override.
# Set it only in the top-level process so recursive make shares its jobserver.
MAKE_JOBS ?= 8
ifeq ($(MAKELEVEL),0)
MAKEFLAGS += -j$(MAKE_JOBS)
endif

GO_SCRATCH := $(CURDIR)/.scratch/go
export GOENV := $(GO_SCRATCH)/env
export GOCACHE := $(GO_SCRATCH)/build
export GOMODCACHE := $(GO_SCRATCH)/mod
export GOTMPDIR := $(GO_SCRATCH)/tmp
export GOBIN := $(GO_SCRATCH)/bin
export GOPATH := $(GO_SCRATCH)
export GOFLAGS :=
export GOWORK := off
export GOLANGCI_LINT_CACHE := $(CURDIR)/.scratch/golangci-lint-cache

.DEFAULT_GOAL := check

## Go commands use project-local state so caches and temporary files stay under the workspace.
.PHONY: go-env
go-env:
	@mkdir -p "$(GOCACHE)" "$(GOMODCACHE)" "$(GOTMPDIR)" "$(GOBIN)" "$(GOLANGCI_LINT_CACHE)"

## ---------------------------------------------------------
## Single-entry check target (runs formatting, lint [go, markdown, vale], security, tests, and docs)
## ---------------------------------------------------------
.PHONY: check
check: lint vuln test verify-docs ## Run all checks (format, tidy, lint [go, markdown, vale], security, tests, and docs)

## Reader targets wait for source formatting and dependency updates.
lint-go lint-markdown lint-vale vuln test docgen-source build-next: tidy

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
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install mvdan.cc/gofumpt@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	@if command -v pnpm >/dev/null 2>&1; then \
		pnpm install -g markdownlint-cli2; \
	elif command -v npm >/dev/null 2>&1; then \
		npm install -g markdownlint-cli2; \
	fi
	@if command -v brew >/dev/null 2>&1; then \
		brew install vale; \
	else \
		go install github.com/errata-ai/vale/v3/cmd/vale@latest; \
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
lint-fix: go-env fix-markdown ## Apply unambiguous fixes supplied by configured Go and Markdown linters
	@echo "==> Applying Go linter fixes..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix ./...; \
	else \
		echo "golangci-lint not installed. Run 'make tools' or: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

.PHONY: fmt
fmt: fix ## Format Go source code and optimize imports

.PHONY: fmt-go
fmt-go: go-env ## Format Go source code and optimize imports without lint fixes
	@echo "==> Formatting code..."
	@if command -v gofumpt >/dev/null 2>&1; then \
		gofumpt -w .; \
	else \
		go fmt ./...; \
	fi
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w -local $$(go list -m 2>/dev/null || echo "") .; \
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

.PHONY: lint-go
lint-go: go-env ## Run golangci-lint
	@echo "==> Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run 'make tools' or: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

.PHONY: lint-markdown
lint-markdown: ## Run markdownlint-cli2
	@echo "==> Running markdownlint-cli2..."
	@if command -v markdownlint-cli2 >/dev/null 2>&1; then \
		markdownlint-cli2 "**/*.md" "!.agents/"; \
	else \
		echo "markdownlint-cli2 not installed. Run 'make tools' or: pnpm install -g markdownlint-cli2"; \
		exit 1; \
	fi

.PHONY: fix-markdown
fix-markdown: ## Apply automatic markdownlint-cli2 fixes
	@echo "==> Applying Markdown linter fixes..."
	@if command -v markdownlint-cli2 >/dev/null 2>&1; then \
		markdownlint-cli2 --fix "**/*.md"; \
	else \
		echo "markdownlint-cli2 not installed. Run 'make tools' or: pnpm install -g markdownlint-cli2"; \
		exit 1; \
	fi

.PHONY: lint-vale
lint-vale: ## Run vale prose linter
	@echo "==> Running vale..."
	@if command -v vale >/dev/null 2>&1; then \
		find . -path './.scratch' -prune -o -path './.git' -prune -o -path './data' -prune -o -type f -name '*.md' -print0 | xargs -0 vale; \
	else \
		echo "vale not installed. Run 'make tools' or: brew install vale"; \
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
.PHONY: test
test: go-env ## Run unit and race tests
	@echo "==> Running unit tests..."
	go test -race -shuffle=on -cover -v ./...

.PHONY: test-short
test-short: go-env ## Run short test suite without long-running tests
	@echo "==> Running short tests..."
	go test -short ./...

.PHONY: test-property-full
test-property-full: go-env ## Run comprehensive 100-check property tests
	@echo "==> Running full property tests (100 checks)..."
	RAPID_CHECKS=100 go test -race -v -run TestProperty_ ./internal/adapters/golang/...

.PHONY: test-fuzz
test-fuzz: go-env ## Run deep 1000-check metamorphic property fuzzing
	@echo "==> Running deep property fuzzing (1000 checks)..."
	RAPID_CHECKS=1000 go test -race -v -run TestProperty_ ./internal/adapters/golang/...

## ---------------------------------------------------------
## Build and Clean
## ---------------------------------------------------------
.PHONY: build-next
build-next: go-env ## Build the active development binary (bin/semedit-next)
	@echo "==> Building bin/semedit-next..."
	@mkdir -p bin
	go build -o bin/semedit-next .

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
	go run ./cmd/docgen --output-dir .scratch/docgen

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

.PHONY: clean
clean: go-env ## Clean build artifacts and test cache
	@echo "==> Cleaning cache..."
	go clean -testcache
	rm -rf bin/ dist/

.PHONY: help
help: ## Display this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
