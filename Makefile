BINARY_DIR := bin
FMEM       := $(BINARY_DIR)/fmem
FQUERY     := $(BINARY_DIR)/fquery
MODULE     := github.com/phmotad/firememory
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS    := -ldflags "-s -w -X $(MODULE)/internal/version.Version=$(VERSION)"
TAGS       := onnx

.PHONY: all build build-fmem build-fquery test lint clean release-snapshot help

all: build

## build: build fmem and fquery binaries into ./bin/
build: build-fmem build-fquery

build-fmem:
	@mkdir -p $(BINARY_DIR)
	go build -tags $(TAGS) $(LDFLAGS) -o $(FMEM) ./cmd/fmem

build-fquery:
	@mkdir -p $(BINARY_DIR)
	go build -tags $(TAGS) $(LDFLAGS) -o $(FQUERY) ./cmd/fquery

## test: run all tests (no onnx tag — uses DeterministicEmbedder, offline-safe)
test:
	go test ./...

## test-onnx: run all tests with ONNX backend (requires models in FIREMEMORY_MODELS_DIR)
test-onnx:
	go test -tags onnx ./...

## test-verbose: run all tests with verbose output
test-verbose:
	go test -v ./...

## test-race: run all tests with race detector
test-race:
	go test -race ./...

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## release-snapshot: build release snapshot locally (requires goreleaser + downloads ORT)
release-snapshot:
	@test -f scripts/download-ort.sh && bash scripts/download-ort.sh || true
	goreleaser build --snapshot --clean

## clean: remove build artifacts
clean:
	rm -rf $(BINARY_DIR)

## help: show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //'
