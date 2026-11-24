NAME := deepseek-search-mcp
BIN_STDIO := bin/$(NAME)
BIN_HTTP := bin/$(NAME)-http
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
DIST := dist
RELEASE_PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: build build-stdio build-http run run-stdio run-http tidy clean release release-platform checksums clean-dist

build: build-stdio build-http

build-stdio:
	go build -o $(BIN_STDIO) ./cmd/stdio

build-http:
	go build -o $(BIN_HTTP) ./cmd/http

run: run-stdio

run-stdio: build-stdio
	$(BIN_STDIO)

run-http: build-http
	$(BIN_HTTP)

tidy:
	go mod tidy

clean:
	rm -rf bin logs $(DIST)

clean-dist:
	rm -rf $(DIST)

# Package both binaries for one GOOS/GOARCH into dist/.
release-platform:
	@set -eu; \
	os="$(GOOS)"; arch="$(GOARCH)"; \
	test -n "$$os"; test -n "$$arch"; \
	ext=""; \
	if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
	bundle="$(NAME)_$(VERSION)_$${os}_$${arch}"; \
	stage="$(DIST)/$$bundle"; \
	mkdir -p "$$stage"; \
	CGO_ENABLED=0 GOOS="$$os" GOARCH="$$arch" go build -trimpath -ldflags="-s -w" \
		-o "$$stage/$(NAME)$$ext" ./cmd/stdio; \
	CGO_ENABLED=0 GOOS="$$os" GOARCH="$$arch" go build -trimpath -ldflags="-s -w" \
		-o "$$stage/$(NAME)-http$$ext" ./cmd/http; \
	cp README.md .env.example "$$stage/"; \
	if [ "$$os" = "windows" ]; then \
		( cd "$(DIST)" && zip -qr "$$bundle.zip" "$$bundle" ); \
	else \
		tar -C "$(DIST)" -czf "$(DIST)/$$bundle.tar.gz" "$$bundle"; \
	fi; \
	rm -rf "$$stage"

# Cross-compile every release platform and write dist/checksums.txt.
release: clean-dist
	@set -eu; \
	for platform in $(RELEASE_PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		$(MAKE) --no-print-directory release-platform GOOS="$$os" GOARCH="$$arch" VERSION="$(VERSION)"; \
	done
	$(MAKE) --no-print-directory checksums VERSION="$(VERSION)"

checksums:
	cd $(DIST) && sha256sum *.tar.gz *.zip > checksums.txt
