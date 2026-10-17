# Share reusable Go caches across linked checkouts while keeping publication, tools, configuration, and analysis state local.
ifndef SEMEDIT_GO_ENVIRONMENT_INCLUDED
SEMEDIT_GO_ENVIRONMENT_INCLUDED := 1

GO_SCRATCH ?= $(REPO_ROOT)/.scratch/go
ifeq ($(origin GO_CACHE_ROOT),undefined)
GO_CACHE_ROOT := $(shell bash "$(REPO_ROOT)/tools/go-cache-root.sh" "$(REPO_ROOT)" || printf '\nSEMEDIT_CACHE_ROOT_ERROR')
ifneq ($(findstring SEMEDIT_CACHE_ROOT_ERROR,$(GO_CACHE_ROOT)),)
$(error cannot resolve shared Go caches; repair Git metadata or set GO_CACHE_ROOT explicitly)
endif
endif
ifeq ($(strip $(GO_CACHE_ROOT)),)
$(error GO_CACHE_ROOT must not be empty)
endif

export GOENV := $(GO_SCRATCH)/env
export GOCACHE ?= $(GO_CACHE_ROOT)/build
export GOMODCACHE ?= $(GO_CACHE_ROOT)/mod
export GOTMPDIR := $(GO_SCRATCH)/tmp
export GOBIN := $(GO_SCRATCH)/bin
export GOPATH := $(GO_SCRATCH)
export GOPLSCACHE := $(GO_SCRATCH)/gopls
export GOFLAGS :=
export GOWORK := off
export GOLANGCI_LINT_CACHE := $(REPO_ROOT)/.scratch/golangci-lint-cache
endif
