set shell := ["sh", "-cu"]

binary := "jot"
gocache := "$PWD/.gocache"
gomodcache := "$PWD/.gomodcache"

default:
  @just --list

help:
  GOCACHE={{gocache}} GOMODCACHE={{gomodcache}} go run ./cmd/jot --help

fmt:
  gofmt -w cmd/jot/main.go internal/jot/*.go

build:
  mkdir -p bin
  GOCACHE={{gocache}} GOMODCACHE={{gomodcache}} go build -o bin/{{binary}} ./cmd/jot

test:
  GOCACHE={{gocache}} GOMODCACHE={{gomodcache}} go test ./...

lint:
  @if ! command -v golangci-lint >/dev/null 2>&1; then \
    echo "golangci-lint is not installed"; \
    exit 1; \
  fi
  GOCACHE={{gocache}} GOMODCACHE={{gomodcache}} GOLANGCI_LINT_CACHE=$PWD/.golangci-cache golangci-lint run

run *args:
  GOCACHE={{gocache}} GOMODCACHE={{gomodcache}} go run ./cmd/jot {{args}}
