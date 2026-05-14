BINARY := userhunt
PKG    := github.com/nodirsafarov/userhunt
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

.PHONY: build test vet lint install clean release-dry run

build:
	go build -trimpath -ldflags '$(LDFLAGS)' -o bin/$(BINARY) ./cmd/$(BINARY)

install:
	go install -trimpath -ldflags '$(LDFLAGS)' ./cmd/$(BINARY)

test:
	go test -race -cover ./...

vet:
	go vet ./...

lint:
	@which golangci-lint > /dev/null 2>&1 || (echo "install golangci-lint first" && exit 1)
	golangci-lint run ./...

clean:
	rm -rf bin dist coverage.out coverage.html

run: build
	./bin/$(BINARY) $(ARGS)

release-dry:
	@which goreleaser > /dev/null 2>&1 || (echo "install goreleaser first" && exit 1)
	goreleaser release --snapshot --clean
