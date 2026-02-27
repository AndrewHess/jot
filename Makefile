BINARY := jot

.PHONY: build test run lint

build:
	go build -o bin/$(BINARY) ./cmd/jot

test:
	GOCACHE=$(PWD)/.gocache go test ./...

run:
	go run ./cmd/jot

lint:
	golangci-lint run
