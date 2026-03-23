GO ?= go
APP ?= sub
CMD ?= ./cmd/sub
BIN_DIR ?= ./bin
AIR ?= air

.PHONY: help run dev install-air build test fmt vet tidy check clean

help: ## Show available commands
	@printf "Available commands:\n"
	@printf "  make run    - Run the CLI\n"
	@printf "  make dev    - Run with live reload (air)\n"
	@printf "  make install-air - Install air live-reload tool\n"
	@printf "  make build  - Build binary into ./bin\n"
	@printf "  make test   - Run tests\n"
	@printf "  make fmt    - Format Go code\n"
	@printf "  make vet    - Run go vet\n"
	@printf "  make tidy   - Run go mod tidy\n"
	@printf "  make check  - Run fmt + vet + test\n"
	@printf "  make clean  - Remove build artifacts\n"

run: ## Run the CLI
	$(GO) run $(CMD)

dev: ## Run with live reload using air
	@if command -v $(AIR) >/dev/null 2>&1; then \
		$(AIR) -c .air.toml; \
	else \
		printf "air not found. Install with: make install-air\n"; \
		exit 1; \
	fi

install-air: ## Install air live reload tool
	$(GO) install github.com/air-verse/air@latest

build: ## Build binary
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(APP) $(CMD)

test: ## Run tests
	$(GO) test ./...

fmt: ## Format code
	$(GO) fmt ./...

vet: ## Run static checks
	$(GO) vet ./...

tidy: ## Tidy module files
	$(GO) mod tidy

check: fmt vet test ## Run local CI checks

clean: ## Clean build output
	rm -rf $(BIN_DIR)
