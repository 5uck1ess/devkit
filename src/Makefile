BINARY  := devkit
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)
GOFLAGS := -trimpath

.PHONY: build install link clean test vet fmt check all

all: check build

build:
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o bin/$(BINARY) .

install:
	go install $(GOFLAGS) -ldflags '$(LDFLAGS)' .
	@echo "Installed to $$(go env GOPATH)/bin/$(BINARY)"
	@echo "Ensure $$(go env GOPATH)/bin is in your PATH"

link: build
	ln -sf $(CURDIR)/bin/$(BINARY) /usr/local/bin/$(BINARY)
	@echo "Linked bin/$(BINARY) → /usr/local/bin/$(BINARY)"

clean:
	rm -rf bin/

test:
	go test ./... -v

vet:
	go vet ./...

fmt:
	gofmt -w .

check: vet fmt
