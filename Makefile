# maelsink — Makefile
# Version: 0.1.0
# Description: Build and management tasks for maelsink

# --- Configuration ---
BINARY_NAME=maelsink
BIN_DIR=bin
PKG=github.com/0funct0ry/maelsink
MAIN_DIR=.
WEB_DIR=web
WEB_COMPOSE_DIR=web-compose
SITE_DIR=site

# Build-time variables
#
# VERSION is the semver (x.y.z, no leading "v") derived from the nearest
# git tag. `git describe --tags --match 'v*'` walks back to the last
# annotated/lightweight tag reachable from HEAD; on an exact tag it returns
# just that tag, otherwise it appends "-N-gSHA[-dirty]" (standard git
# describe long form) which we treat as a pre-release/build-metadata
# suffix rather than stripping it, so non-release builds remain
# distinguishable. Falls back to "0.0.0" (not "dev") on a shallow clone or
# a repo with no tags yet, so `maelsink version`/`--version` always prints
# a valid semver even outside CI.
VERSION?=$(shell git describe --tags --match 'v*' --always --dirty 2>/dev/null | sed 's/^v//' || echo "0.0.0")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go build flags
LDFLAGS=-ldflags "-X $(PKG)/internal/version.Version=$(VERSION) \
                  -X $(PKG)/internal/version.Commit=$(COMMIT) \
                  -X $(PKG)/internal/version.BuildDate=$(BUILD_DATE) \
                  -s -w"

# --- Colors for help output ---
BLUE=\033[0;34m
GREEN=\033[0;32m
CYAN=\033[0;36m
NC=\033[0m # No Color

.PHONY: all help build build-go build-web build-web-compose build-docker ensure-web-embed ensure-web-compose-embed fmt vet test test-leak check lint clean run dev-go dev-web dev-web-compose site-dev site-build vhs

all: build ## Build everything (frontend and backend)

help: ## Show this help message
	@echo "$(BLUE)maelsink$(NC) build system"
	@echo ""
	@echo "Usage: make $(CYAN)<target>$(NC)"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-15s$(NC) %s\n", $$1, $$2}'

# --- Build Targets ---

build: build-web build-web-compose build-go ## Build frontend assets and then the Go binary

build-go: ensure-web-embed ensure-web-compose-embed ## Build the Go binary
	@echo "$(BLUE)Building Go binary...$(NC)"
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_DIR)/main.go

build-web: ## Build the frontend SPA (React/Vite)
	@echo "$(BLUE)Building frontend assets...$(NC)"
	@if [ -d "$(WEB_DIR)" ]; then \
		cd $(WEB_DIR) && npm ci && npm run build; \
	else \
		echo "$(GREEN)Skip: $(WEB_DIR) directory not found.$(NC)"; \
	fi

build-web-compose: ## Build the compose frontend SPA (React/Vite)
	@echo "$(BLUE)Building compose frontend assets...$(NC)"
	@if [ -d "$(WEB_COMPOSE_DIR)" ]; then \
		cd $(WEB_COMPOSE_DIR) && npm ci && npm run build; \
	else \
		echo "$(GREEN)Skip: $(WEB_COMPOSE_DIR) directory not found.$(NC)"; \
	fi

build-docker: ## Build the multi-stage Docker image
	@echo "$(BLUE)Building Docker image...$(NC)"
	docker build -t maelsink:latest .

ensure-web-embed: ## Build web assets only if missing, so go:embed has something to compile
	@if [ ! -f internal/webui/dist/index.html ]; then \
		$(MAKE) build-web; \
	fi

ensure-web-compose-embed: ## Build compose web assets only if missing, so go:embed has something to compile
	@if [ ! -f internal/compose/dist/index.html ]; then \
		$(MAKE) build-web-compose; \
	fi

# --- Development Targets ---

run: build-go ## Run the application
	./$(BIN_DIR)/$(BINARY_NAME) serve

dev-go: ## Run Go with hot reload (requires air)
	@command -v air > /dev/null || (echo "air not found, install it with: go install github.com/air-verse/air@latest" && exit 1)
	air

dev-web: ## Run frontend dev server (Vite)
	@if [ -d "$(WEB_DIR)" ]; then \
		cd $(WEB_DIR) && npm run dev; \
	else \
		echo "Error: $(WEB_DIR) directory not found."; exit 1; \
	fi

dev-web-compose: ## Run compose frontend dev server (Vite)
	@if [ -d "$(WEB_COMPOSE_DIR)" ]; then \
		cd $(WEB_COMPOSE_DIR) && npm run dev; \
	else \
		echo "Error: $(WEB_COMPOSE_DIR) directory not found."; exit 1; \
	fi

# --- QA Targets ---

fmt: ## Format Go source code
	@echo "$(BLUE)Formatting code...$(NC)"
	go fmt ./...

vet: ensure-web-embed ensure-web-compose-embed ## Run Go vet
	@echo "$(BLUE)Running vet...$(NC)"
	go vet ./...

test: ensure-web-embed ensure-web-compose-embed ## Run Go tests with the race detector (always on, never skip it)
	@echo "$(BLUE)Running tests...$(NC)"
	go test -v -race ./...

test-leak: ensure-web-embed ensure-web-compose-embed ## Run tests with goroutine-leak detection (uber-go/goleak) in addition to -race
	@echo "$(BLUE)Running tests with leak detection...$(NC)"
	go test -v -race -tags leakcheck ./...

check: fmt vet lint test ## Full local QA gate: fmt, vet, lint, race-enabled tests

lint: ## Run golangci-lint
	@echo "$(BLUE)Running linter...$(NC)"
	@command -v golangci-lint > /dev/null || (echo "golangci-lint not found" && exit 1)
	golangci-lint run

# --- Documentation & Marketing Site ---

site-dev: ## Run the Astro documentation site in dev mode
	@if [ -d "$(SITE_DIR)" ]; then \
		cd $(SITE_DIR) && npm run dev; \
	else \
		echo "Error: $(SITE_DIR) directory not found."; exit 1; \
	fi

site-build: ## Build the static documentation site (Astro/Starlight)
	@echo "$(BLUE)Building documentation site...$(NC)"
	@if [ -d "$(SITE_DIR)" ]; then \
		cd $(SITE_DIR) && npm run build; \
	else \
		echo "Error: $(SITE_DIR) directory not found."; exit 1; \
	fi

vhs: ## Regenerate terminal GIFs using VHS
	@echo "$(BLUE)Generating VHS terminal recordings...$(NC)"
	@if [ -d "$(SITE_DIR)/vhs" ]; then \
		cd $(SITE_DIR)/vhs && vhs *.tape; \
	else \
		echo "Error: $(SITE_DIR)/vhs directory not found."; exit 1; \
	fi

# --- Cleanup ---

clean: ## Remove build artifacts and temporary files
	@echo "$(BLUE)Cleaning up...$(NC)"
	rm -rf $(BIN_DIR)
	rm -rf internal/webui/dist
	rm -rf internal/compose/dist
	rm -rf $(SITE_DIR)/dist
	go clean
