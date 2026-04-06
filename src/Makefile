BINARY  := devkit
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
# Clean version for plugin.json (strip leading v, git suffix — just semver)
SEMVER  := $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0")
LDFLAGS := -s -w -X main.version=$(VERSION)
GOFLAGS := -trimpath
PLUGIN_JSON := $(CURDIR)/../.claude-plugin/plugin.json

.PHONY: build install link clean test vet fmt check all sync-version

all: check build

sync-version:
	@if [ -f "$(PLUGIN_JSON)" ] && command -v jq >/dev/null 2>&1; then \
		CURRENT=$$(jq -r '.version' "$(PLUGIN_JSON)"); \
		if [ "$$CURRENT" != "$(SEMVER)" ]; then \
			jq --arg v "$(SEMVER)" '.version = $$v' "$(PLUGIN_JSON)" > "$(PLUGIN_JSON).tmp" && mv "$(PLUGIN_JSON).tmp" "$(PLUGIN_JSON)"; \
			echo "Synced plugin.json version: $$CURRENT → $(SEMVER)"; \
		fi; \
	fi

build: sync-version
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o bin/$(BINARY) .

install: sync-version
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
