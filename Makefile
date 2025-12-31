SHELL := /bin/bash

.PHONY: build fmt lint test ci tools clean setup

setup:
	@command -v lefthook >/dev/null || (echo "Install lefthook: brew install lefthook" && exit 1)
	lefthook install

TOOLS_DIR := $(CURDIR)/.tools
GOFUMPT := $(TOOLS_DIR)/gofumpt
GOIMPORTS := $(TOOLS_DIR)/goimports
GOLANGCI_LINT := $(TOOLS_DIR)/golangci-lint

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags "-X github.com/salmonumbrella/threads-go/internal/cmd.Version=$(VERSION) -X github.com/salmonumbrella/threads-go/internal/cmd.Commit=$(COMMIT) -X github.com/salmonumbrella/threads-go/internal/cmd.BuildDate=$(BUILD_DATE)"

build:
	go build $(LDFLAGS) -o ./bin/threads ./cmd/threads

tools:
	@mkdir -p $(TOOLS_DIR)
	@GOBIN=$(TOOLS_DIR) go install mvdan.cc/gofumpt@v0.7.0
	@GOBIN=$(TOOLS_DIR) go install golang.org/x/tools/cmd/goimports@v0.28.0
	@GOBIN=$(TOOLS_DIR) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2

fmt: tools
	@$(GOIMPORTS) -w .
	@$(GOFUMPT) -w .

fmt-check: tools
	@$(GOIMPORTS) -w .
	@$(GOFUMPT) -w .
	@git diff --exit-code -- '*.go' go.mod go.sum

lint: tools
	@$(GOLANGCI_LINT) run

test:
	@go test ./...

ci: fmt-check lint test

clean:
	rm -rf bin/ .tools/
