BINARY  := devkit
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)
GOFLAGS := -trimpath

.PHONY: build install clean test vet fmt check all

all: check build

build:
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o bin/$(BINARY) .

install:
	go install $(GOFLAGS) -ldflags '$(LDFLAGS)' .

clean:
	rm -rf bin/

test:
	go test ./... -v

vet:
	go vet ./...

fmt:
	gofmt -w .

check: vet fmt
