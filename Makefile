.PHONY: all build test test-coverage clean lint run

BINARY_NAME=bin/toolcraft
VERSION?=1.0.0
COMMIT?=$(shell git rev-parse --short HEAD 2>nul || echo "dev")
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>nul || echo "unknown")
LDFLAGS=-ldflags "-s -w -X github.com/alexandrmotologa/toolcraft/internal/version.Version=$(VERSION) -X github.com/alexandrmotologa/toolcraft/internal/version.Commit=$(COMMIT) -X github.com/alexandrmotologa/toolcraft/internal/version.BuildDate=$(BUILD_DATE)"

all: build

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/toolcraft

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -html=coverage.txt -o coverage.html

clean:
	rm -rf bin/ dist/ coverage.txt coverage.html

run: build
	./$(BINARY_NAME) --help
