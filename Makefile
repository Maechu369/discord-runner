BINARY := discord-runner
CONFIG ?= config.json

.PHONY: all build build-arm64 test race vet fmt fmtcheck check run init flatten clean help

all: check build

## build: Build the binary
build:
	go build -o $(BINARY) .

## build-arm64: Build an arm64 binary
build-arm64:
	CGO_ENABLED=0 GOARCH=arm64 go build -o $(BINARY)-arm64 .

## test: Run all tests
test:
	go test ./...

## race: Run tests with the race detector
race:
	go test -race ./...

## vet: Run go vet
vet:
	go vet ./...

## fmt: Format Go sources
fmt:
	gofmt -w .

## fmtcheck: Fail if any Go file is not gofmt-formatted
fmtcheck:
	@FILES=$$(gofmt -l .); \
	if [ -n "$$FILES" ]; then \
		echo "gofmt required (run 'make fmt'):"; echo "$$FILES"; \
		exit 1; \
	fi

## check: Format check, vet, and tests
check: fmtcheck vet test

## run: Run the Discord bot from config.json
run:
	go run . start $(CONFIG)

## init: Generate config.json via `config init`
init:
	./$(BINARY) config init $(CONFIG)

## flatten: Print config.json as key=value lines via `config flatten`
flatten:
	./$(BINARY) config flatten $(CONFIG)

## clean: Remove the binary
clean:
	rm -f $(BINARY)

## help: Show this help
help:
	@C=$$(printf '\033[36m'); R=$$(printf '\033[0m'); \
	grep -hE '^## [a-zA-Z0-9_-]+' $(MAKEFILE_LIST) \
		| sort \
		| awk -v c="$$C" -v r="$$R" 'BEGIN{FS=": "}{n=$$1; sub(/^## /,"",n); printf "  %s%-10s%s  %s\n", c, n, r, $$2}'
