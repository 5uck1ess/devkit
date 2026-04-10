BINARY  := devkit
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
# Clean semver for plugin.json (strip leading v; falls back to 0.0.0 when untagged)
SEMVER  := $(shell V=$$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//'); echo "$${V:-0.0.0}")
LDFLAGS := -s -w -X main.version=$(VERSION)
GOFLAGS := -trimpath
PLUGIN_JSON := $(CURDIR)/../.claude-plugin/plugin.json

.PHONY: build build-for build-all install link clean test vet fmt check all sync-version

all: check build

sync-version:
	@if ! command -v jq >/dev/null 2>&1; then \
		echo "WARNING: jq not found, skipping plugin.json version sync"; \
	elif [ ! -f "$(PLUGIN_JSON)" ]; then \
		echo "WARNING: $(PLUGIN_JSON) not found, skipping version sync"; \
	elif [ "$(SEMVER)" = "0.0.0" ]; then \
		echo "WARNING: no git tag found, skipping plugin.json version sync"; \
	elif ! echo "$(SEMVER)" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+'; then \
		echo "WARNING: git tag is not valid semver ($(SEMVER)), skipping sync"; \
	else \
		CURRENT=$$(jq -r '.version' "$(PLUGIN_JSON)" 2>/dev/null) || \
			{ echo "ERROR: plugin.json is not valid JSON"; exit 1; }; \
		if [ "$$CURRENT" != "$(SEMVER)" ]; then \
			jq --arg v "$(SEMVER)" '.version = $$v' "$(PLUGIN_JSON)" > "$(PLUGIN_JSON).tmp" \
				&& mv "$(PLUGIN_JSON).tmp" "$(PLUGIN_JSON)" \
				|| { rm -f "$(PLUGIN_JSON).tmp"; echo "ERROR: failed to sync plugin.json"; exit 1; }; \
			echo "Synced plugin.json version: $$CURRENT -> $(SEMVER)"; \
		fi; \
	fi

build: | sync-version
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o bin/$(BINARY) .

build-for: ## Cross-compile for specified GOOS/GOARCH
	$(eval EXT := $(if $(filter windows,$(GOOS)),.exe,))
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o bin/$(BINARY)-$(GOOS)-$(GOARCH)$(EXT) .
	@echo "Built: bin/$(BINARY)-$(GOOS)-$(GOARCH)$(EXT)"

build-all: ## Build all platform binaries
	@$(MAKE) build-for GOOS=linux GOARCH=amd64
	@$(MAKE) build-for GOOS=linux GOARCH=arm64
	@$(MAKE) build-for GOOS=darwin GOARCH=amd64
	@$(MAKE) build-for GOOS=darwin GOARCH=arm64
	@$(MAKE) build-for GOOS=windows GOARCH=amd64
	@$(MAKE) build-for GOOS=windows GOARCH=arm64

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
