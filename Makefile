VERSION_PKG := github.com/macreleaser/macreleaser/pkg/version
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(VERSION_PKG).version=$(VERSION) \
	-X $(VERSION_PKG).commit=$(COMMIT) \
	-X $(VERSION_PKG).date=$(DATE)

.PHONY: build test vet clean install test-coverage

build:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/macreleaser ./cmd/macreleaser

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf bin/ coverage.out coverage.html

install: build
	cp bin/macreleaser /usr/local/bin/

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

run-init:
	go run ./cmd/macreleaser init

run-check:
	go run ./cmd/macreleaser check