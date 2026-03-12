BINARY  := cache-warmer
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-X main.version=$(VERSION) -s -w"

.PHONY: build install clean release test

build:
	go build $(LDFLAGS) -o $(BINARY) .

install:
	go install $(LDFLAGS) .

clean:
	rm -f $(BINARY)
	rm -rf dist/

test:
	go vet ./...

release:
	goreleaser release --clean
