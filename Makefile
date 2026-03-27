BIN     := bv
MODULE  := github.com/may/bad-vibes
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -s -w"

.PHONY: build install clean test lint tidy build-all

build:
	go build $(LDFLAGS) -o $(BIN) .

install:
	go install $(LDFLAGS) .

clean:
	rm -f $(BIN)
	rm -rf dist/

test:
	go test ./...

test-verbose:
	go test -v -race ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

build-all: dist/
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BIN)-darwin-arm64 .
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BIN)-darwin-amd64 .
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BIN)-linux-amd64 .
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BIN)-windows-amd64.exe .

dist/:
	mkdir -p dist
