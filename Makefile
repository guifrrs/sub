GO ?= go
APP ?= sub
CMD ?= ./cmd/sub
BIN_DIR ?= ./bin

.PHONY: help run build test fmt vet tidy check clean

help: ## Show available commands
	@printf "Available commands:\n"
	@printf "  make run    - Run the CLI\n"
	@printf "  make build  - Build binary into ./bin\n"
	@printf "  make test   - Run tests\n"
	@printf "  make fmt    - Format Go code\n"
	@printf "  make vet    - Run go vet\n"
	@printf "  make tidy   - Run go mod tidy\n"
	@printf "  make check  - Run fmt + vet + test\n"
	@printf "  make clean  - Remove build artifacts\n"

run: ## Run the CLI
	$(GO) run $(CMD)

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
