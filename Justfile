set shell := ["sh", "-cu"]

binary := "jot"
gocache := "$PWD/.gocache"
gomodcache := "$PWD/.gomodcache"

default:
  @just --list

fmt:
  gofmt -w cmd/jot/main.go internal/jot/*.go

build:
  mkdir -p bin
  GOCACHE={{gocache}} GOMODCACHE={{gomodcache}} go build -o bin/{{binary}} ./cmd/jot

test:
  GOCACHE={{gocache}} GOMODCACHE={{gomodcache}} go test ./...

lint:
  golangci-lint run

run *args:
  GOCACHE={{gocache}} GOMODCACHE={{gomodcache}} go run ./cmd/jot {{args}}

smoke:
  tmpdir=$(mktemp -d /tmp/jot-smoke.XXXXXX)
  cd "$tmpdir"
  {{justfile_directory()}}/bin/jot init
  {{justfile_directory()}}/bin/jot add "first"
  {{justfile_directory()}}/bin/jot add -c "checkbox"
  {{justfile_directory()}}/bin/jot done 2
  {{justfile_directory()}}/bin/jot show
  echo "SMOKE_DIR=$tmpdir"
