.PHONY: build test vet clean install test-coverage

build:
	@mkdir -p bin
	go build -o bin/macreleaser ./cmd/macreleaser

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