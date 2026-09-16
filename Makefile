# Standard Go Makefile for semedit
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

.DEFAULT_GOAL := check

## ---------------------------------------------------------
## Single-entry check target (runs formatting, lint, security, tests)
## ---------------------------------------------------------
.PHONY: check
check: fmt tidy lint vuln test ## Run all checks (format, tidy, lint, security, test)

## ---------------------------------------------------------
## Dependencies & Tooling
## ---------------------------------------------------------
.PHONY: deps
deps: ## Download and verify module dependencies
	@echo "==> Downloading Go dependencies..."
	go mod download
	go mod verify

.PHONY: tools
tools: ## Install development tools (linters, formatters, scanners)
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

## ---------------------------------------------------------
## Formatting, Refactoring & Imports
## ---------------------------------------------------------
.PHONY: fix
fix: ## Apply automatic Go fixes and standard library modernizations
	@echo "==> Running go fix..."
	go fix ./...

.PHONY: fmt
fmt: fix ## Format Go source code and optimize imports
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
tidy: ## Ensure dependencies match source code
	@echo "==> Tidying go.mod and go.sum..."
	go mod tidy

## ---------------------------------------------------------
## Linting and Static Analysis
## ---------------------------------------------------------
.PHONY: lint
lint: lint-go lint-markdown ## Run all static linters (golangci-lint and markdownlint-cli2)

.PHONY: lint-go
lint-go: ## Run golangci-lint
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
		markdownlint-cli2 "**/*.md"; \
	else \
		echo "markdownlint-cli2 not installed. Run 'make tools' or: pnpm install -g markdownlint-cli2"; \
		exit 1; \
	fi

.PHONY: vuln
vuln: ## Run vulnerability security check
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
test: ## Run unit and race tests
	@echo "==> Running unit tests..."
	go test -race -shuffle=on -cover -v ./...

.PHONY: test-short
test-short: ## Run short test suite without long-running tests
	@echo "==> Running short tests..."
	go test -short ./...

## ---------------------------------------------------------
## Build and Clean
## ---------------------------------------------------------
.PHONY: build
build: ## Build all binaries in the module
	@echo "==> Building binaries..."
	go build ./...

.PHONY: clean
clean: ## Clean build artifacts and test cache
	@echo "==> Cleaning cache..."
	go clean -testcache
	rm -rf bin/ dist/

.PHONY: help
help: ## Display this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
