.PHONY: help test test-race cover lint fuzz bench update-golden tidy tidy-check vuln all

GO ?= go
GOLANGCI_LINT ?= golangci-lint
FUZZTIME ?= 30s
BENCHTIME ?= 3s
BENCHCOUNT ?= 5
COVER_PROFILE ?= coverage.out
MODULES := . examples examples/jsonschema

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: test lint tidy-check vuln ## Run tests, lint, tidy check, and vulnerability scan

test: ## Run all tests in all modules
	for m in $(MODULES); do (cd $$m && $(GO) test ./...) || exit 1; done

test-race: ## Run all tests in all modules with the race detector
	for m in $(MODULES); do (cd $$m && $(GO) test -race ./...) || exit 1; done

cover: ## Run tests with coverage profile
	$(GO) test -race -coverprofile=$(COVER_PROFILE) -covermode=atomic ./...
	$(GO) tool cover -func=$(COVER_PROFILE) | tail -1

lint: ## Run golangci-lint
	$(GOLANGCI_LINT) run ./...

fuzz: ## Run all fuzz targets for FUZZTIME each (default 30s)
	$(GO) test -run=NoMatch -fuzz=FuzzParseQuery -fuzztime=$(FUZZTIME) .
	$(GO) test -run=NoMatch -fuzz=FuzzEdit -fuzztime=$(FUZZTIME) .
	$(GO) test -run=NoMatch -fuzz=FuzzUnmarshalJSON -fuzztime=$(FUZZTIME) .
	$(GO) test -run=NoMatch -fuzz=FuzzUnmarshalYAML -fuzztime=$(FUZZTIME) .

bench: ## Run all benchmarks (BENCHTIME=3s BENCHCOUNT=5 by default)
	$(GO) test -run=NoMatch -bench=. -benchtime=$(BENCHTIME) -count=$(BENCHCOUNT) .

update-golden: ## Refresh cmd/tq golden files from current output
	$(GO) test ./cmd/tq -update

tidy: ## Run go mod tidy in all modules
	for m in $(MODULES); do (cd $$m && $(GO) mod tidy) || exit 1; done

tidy-check: ## Fail if go mod tidy would change any module
	for m in $(MODULES); do (cd $$m && $(GO) mod tidy -diff) || exit 1; done

vuln: ## Run govulncheck against the module
	$(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...
