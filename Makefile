.DEFAULT_GOAL := help

BINARY       := clearstack
PKG          := github.com/guilhermejansen/clearstack
VERSION_PKG  := $(PKG)/internal/version
VERSION      := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT       := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE         := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS      := -s -w \
                -X $(VERSION_PKG).Version=$(VERSION) \
                -X $(VERSION_PKG).Commit=$(COMMIT) \
                -X $(VERSION_PKG).Date=$(DATE)

GO_FILES     := $(shell find . -name '*.go' -not -path './vendor/*' 2>/dev/null)

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: tidy
tidy: ## go mod tidy
	go mod tidy

.PHONY: build
build: ## Build binary into ./bin
	@mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o bin/$(BINARY) ./cmd/$(BINARY)

.PHONY: install
install: ## go install
	CGO_ENABLED=0 go install -trimpath -ldflags '$(LDFLAGS)' ./cmd/$(BINARY)

.PHONY: run
run: ## Build and run
	go run ./cmd/$(BINARY)

.PHONY: test
test: ## Run unit tests with race detector
	go test -race -count=1 ./...

.PHONY: test-cover
test-cover: ## Unit tests with coverage
	go test -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

.PHONY: lint
lint: ## golangci-lint
	golangci-lint run ./...

.PHONY: fmt
fmt: ## gofmt + goimports
	gofmt -s -w .
	@command -v goimports >/dev/null 2>&1 && goimports -w . || true

.PHONY: vet
vet: ## go vet
	go vet ./...

.PHONY: check
check: tidy fmt vet lint test ## Full static check + tests

.PHONY: cross
cross: ## Cross-compile sanity (darwin/linux/windows × amd64/arm64)
	@mkdir -p bin
	@for os in darwin linux windows; do \
	  for arch in amd64 arm64; do \
	    ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
	    echo "building $$os/$$arch..."; \
	    CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath \
	      -ldflags '$(LDFLAGS)' \
	      -o bin/$(BINARY)-$$os-$$arch$$ext ./cmd/$(BINARY); \
	  done; \
	done

.PHONY: snapshot
snapshot: ## goreleaser snapshot build
	goreleaser release --snapshot --clean

.PHONY: release
release: ## goreleaser release (requires tag)
	goreleaser release --clean

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf bin dist coverage.out
