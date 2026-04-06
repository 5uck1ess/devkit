BINARY  := devkit
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
# Clean semver for plugin.json (strip leading v — fallback only if no tags exist)
SEMVER  := $(shell V=$$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//'); echo "$${V:-0.0.0}")
LDFLAGS := -s -w -X main.version=$(VERSION)
GOFLAGS := -trimpath
PLUGIN_JSON := $(CURDIR)/../.claude-plugin/plugin.json

.PHONY: build install link clean test vet fmt check all sync-version

all: check build

sync-version:
	@if ! command -v jq >/dev/null 2>&1; then \
		echo "WARNING: jq not found, skipping plugin.json version sync"; \
	elif [ ! -f "$(PLUGIN_JSON)" ]; then \
		echo "WARNING: $(PLUGIN_JSON) not found, skipping version sync"; \
	elif [ "$(SEMVER)" = "0.0.0" ]; then \
		echo "WARNING: no git tag found, skipping plugin.json version sync"; \
	else \
		CURRENT=$$(jq -r '.version' "$(PLUGIN_JSON)"); \
		if [ "$$CURRENT" != "$(SEMVER)" ]; then \
			jq --arg v "$(SEMVER)" '.version = $$v' "$(PLUGIN_JSON)" > "$(PLUGIN_JSON).tmp" \
				&& mv "$(PLUGIN_JSON).tmp" "$(PLUGIN_JSON)" \
				|| { rm -f "$(PLUGIN_JSON).tmp"; echo "ERROR: failed to sync plugin.json"; exit 1; }; \
			echo "Synced plugin.json version: $$CURRENT → $(SEMVER)"; \
		fi; \
	fi

build: | sync-version
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o bin/$(BINARY) .

install: | sync-version
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
