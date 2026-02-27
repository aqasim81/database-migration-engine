# Database Migration Engine — Makefile

BINARY_NAME     := migrate
MAIN_PACKAGE    := ./cmd/migrate
BUILD_DIR       := ./bin
COVERAGE_DIR    := /tmp/migrate-coverage
COVERAGE_FILE   := $(COVERAGE_DIR)/coverage.out
COVERAGE_MIN    := 80

GO              := CGO_ENABLED=1 go
GOTEST          := $(GO) test
GOBUILD         := $(GO) build
GOPATH_BIN      := $(shell go env GOPATH)/bin

GREEN  := \033[0;32m
YELLOW := \033[0;33m
RED    := \033[0;31m
RESET  := \033[0m

.DEFAULT_GOAL := help

# ─────────────────────────────────────────
# HELP
# ─────────────────────────────────────────

.PHONY: help
help: ## Show this help message
	@echo "Usage: make <target>"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

# ─────────────────────────────────────────
# SETUP
# ─────────────────────────────────────────

.PHONY: install-tools
install-tools: ## Install development tools (gitleaks, gofumpt, goimports, golangci-lint)
	go install github.com/zricethezav/gitleaks/v8@latest
	go install mvdan.cc/gofumpt@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "$(GREEN)Tools installed$(RESET)"

# ─────────────────────────────────────────
# DEVELOPMENT
# ─────────────────────────────────────────

.PHONY: build
build: ## Build the binary
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "$(GREEN)Built $(BUILD_DIR)/$(BINARY_NAME)$(RESET)"

.PHONY: run
run: build ## Build and run
	$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: clean
clean: ## Remove build artifacts and coverage files
	rm -rf $(BUILD_DIR) $(COVERAGE_DIR)
	@echo "$(GREEN)Cleaned$(RESET)"

.PHONY: tidy
tidy: ## Tidy go.mod and go.sum
	go mod tidy -v

# ─────────────────────────────────────────
# FORMATTING
# ─────────────────────────────────────────

.PHONY: fmt
fmt: ## Format all Go files with gofumpt + goimports
	gofumpt -l -w .
	$(GOPATH_BIN)/goimports -local github.com/aqasim81/database-migration-engine -w .
	@echo "$(GREEN)Formatted$(RESET)"

.PHONY: fmt-check
fmt-check: ## Check formatting without modifying (CI use)
	@test -z "$$(gofumpt -l .)" || (echo "$(RED)Files need formatting:$(RESET)" && gofumpt -l . && exit 1)
	@echo "$(GREEN)All files formatted correctly$(RESET)"

# ─────────────────────────────────────────
# QUALITY
# ─────────────────────────────────────────

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with autofix
	golangci-lint run --fix ./...

# ─────────────────────────────────────────
# TESTING
# ─────────────────────────────────────────

.PHONY: test
test: ## Run all unit tests with race detection
	$(GOTEST) -v -race -count=1 ./internal/...

.PHONY: test-short
test-short: ## Run short tests only (fast feedback)
	$(GOTEST) -short -race ./internal/...

.PHONY: test-integration
test-integration: ## Run integration tests (requires Docker)
	$(GOTEST) -v -race -count=1 -tags=integration ./integration/...

.PHONY: test-all
test-all: test test-integration ## Run all tests (unit + integration)

# ─────────────────────────────────────────
# COVERAGE
# ─────────────────────────────────────────

COVERAGE_EXCLUDE := /internal/cli/

.PHONY: coverage
coverage: ## Run tests and show coverage breakdown
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -race -coverprofile=$(COVERAGE_DIR)/raw.out -covermode=atomic ./internal/...
	@grep -v '$(COVERAGE_EXCLUDE)' $(COVERAGE_DIR)/raw.out > $(COVERAGE_FILE) || cp $(COVERAGE_DIR)/raw.out $(COVERAGE_FILE)
	$(GO) tool cover -func=$(COVERAGE_FILE)

.PHONY: coverage-html
coverage-html: coverage ## Open coverage report in browser
	$(GO) tool cover -html=$(COVERAGE_FILE)

.PHONY: coverage-check
coverage-check: coverage ## Fail if coverage is below threshold
	@COVERAGE=$$(go tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print $$3}' | tr -d '%'); \
	echo "Total coverage: $${COVERAGE}%  (threshold: $(COVERAGE_MIN)%)"; \
	awk "BEGIN { if ($${COVERAGE}+0 < $(COVERAGE_MIN)) { print \"$(RED)FAIL: coverage $${COVERAGE}%% is below $(COVERAGE_MIN)%%$(RESET)\"; exit 1 } else { print \"$(GREEN)PASS$(RESET)\" } }"

# ─────────────────────────────────────────
# FULL AUDIT (CI gate)
# ─────────────────────────────────────────

.PHONY: audit
audit: fmt-check vet lint test coverage-check ## Full quality gate — format, vet, lint, test, coverage
	go mod tidy -diff
	go mod verify
	@echo "$(GREEN)Audit passed$(RESET)"
