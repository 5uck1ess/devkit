BINARY         := devkit-engine
RELEASE_BINARY := devkit
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
# Clean semver for plugin.json (strip leading v; falls back to 0.0.0 when untagged)
SEMVER  := $(shell V=$$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//'); echo "$${V:-0.0.0}")
LDFLAGS := -s -w -X main.version=$(VERSION)
GOFLAGS := -trimpath
PLUGIN_JSON  := $(CURDIR)/../.claude-plugin/plugin.json
MCPB_MANIFEST := $(CURDIR)/../mcpb/manifest.json

.PHONY: build build-for build-all install install-plugin link clean test vet fmt check all sync-version

all: check build

# sync-version writes $(SEMVER) into both .claude-plugin/plugin.json and
# mcpb/manifest.json when the git tag is higher than the current value.
# Both files must track the plugin version — plugin.json is the manifest
# Claude Code reads, mcpb/manifest.json is the bundled MCP server descriptor.
sync-version:
	@if ! command -v jq >/dev/null 2>&1; then \
		echo "WARNING: jq not found, skipping version sync"; \
		exit 0; \
	fi; \
	if [ "$(SEMVER)" = "0.0.0" ]; then \
		echo "WARNING: no git tag found, skipping version sync"; \
		exit 0; \
	fi; \
	if ! echo "$(SEMVER)" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+'; then \
		echo "WARNING: git tag is not valid semver ($(SEMVER)), skipping sync"; \
		exit 0; \
	fi; \
	for target in "$(PLUGIN_JSON)" "$(MCPB_MANIFEST)"; do \
		if [ ! -f "$$target" ]; then \
			echo "WARNING: $$target not found, skipping"; \
			continue; \
		fi; \
		CURRENT=$$(jq -r '.version' "$$target" 2>/dev/null) || \
			{ echo "ERROR: $$target is not valid JSON"; exit 1; }; \
		HIGHER=$$(printf '%s\n%s\n' "$$CURRENT" "$(SEMVER)" | sort -V | tail -1); \
		if [ "$$HIGHER" = "$(SEMVER)" ] && [ "$$CURRENT" != "$(SEMVER)" ]; then \
			jq --arg v "$(SEMVER)" '.version = $$v' "$$target" > "$$target.tmp" \
				&& mv "$$target.tmp" "$$target" \
				|| { rm -f "$$target.tmp"; echo "ERROR: failed to sync $$target"; exit 1; }; \
			echo "Synced $$(basename $$target) version: $$CURRENT -> $(SEMVER)"; \
		fi; \
	done

build: | sync-version
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o bin/$(BINARY) .

build-for: | sync-version ## Cross-compile release asset for specified GOOS/GOARCH (outputs bin/devkit-<os>-<arch>)
	@EXT=""; [ "$(GOOS)" = "windows" ] && EXT=".exe"; \
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o bin/$(RELEASE_BINARY)-$(GOOS)-$(GOARCH)$${EXT} . && \
	echo "Built: bin/$(RELEASE_BINARY)-$(GOOS)-$(GOARCH)$${EXT}"

build-all: ## Build all platform binaries
	@$(MAKE) build-for GOOS=linux GOARCH=amd64
	@$(MAKE) build-for GOOS=linux GOARCH=arm64
	@$(MAKE) build-for GOOS=darwin GOARCH=amd64
	@$(MAKE) build-for GOOS=darwin GOARCH=arm64
	@$(MAKE) build-for GOOS=windows GOARCH=amd64
	@$(MAKE) build-for GOOS=windows GOARCH=arm64

install-plugin: | sync-version ## Build and copy binary to plugin bin/ for current platform
	@mkdir -p $(CURDIR)/../bin
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(CURDIR)/../bin/$(BINARY) .
	@echo "Installed to plugin bin/$(BINARY)"

install: | sync-version
	go install $(GOFLAGS) -ldflags '$(LDFLAGS)' .
	@echo "Installed to $$(go env GOPATH)/bin/$(RELEASE_BINARY)"
	@echo "Ensure $$(go env GOPATH)/bin is in your PATH"

link: build
	ln -sf $(CURDIR)/bin/$(BINARY) /usr/local/bin/$(RELEASE_BINARY)
	@echo "Linked bin/$(BINARY) → /usr/local/bin/$(RELEASE_BINARY)"

clean:
	rm -rf bin/

test:
	go test ./... -v

vet:
	go vet ./...

fmt:
	gofmt -w .

check: vet fmt
