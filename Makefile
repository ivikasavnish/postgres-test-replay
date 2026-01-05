.PHONY: all build clean test fmt lint docker-up docker-down install help

# Binary name
BINARY=postgres-test-replay
CMD_PATH=./cmd/postgres-test-replay

# Build variables
BUILD_FLAGS=-ldflags="-s -w"
GO=go

all: fmt build

build: ## Build the application
	$(GO) build $(BUILD_FLAGS) -o $(BINARY) $(CMD_PATH)

clean: ## Clean build artifacts
	rm -f $(BINARY)
	rm -rf waldata/ backups/ sessions/ checkpoints/ *.log

test: ## Run tests
	$(GO) test -v ./...

fmt: ## Format code
	$(GO) fmt ./...

lint: ## Run linter
	golangci-lint run || go vet ./...

install: ## Install dependencies
	$(GO) mod download
	$(GO) mod tidy

docker-up: ## Start Docker Compose services
	docker-compose up -d

docker-down: ## Stop Docker Compose services
	docker-compose down -v

docker-logs: ## Show Docker Compose logs
	docker-compose logs -f

run-listener: build ## Run in listener mode
	./$(BINARY) -mode listener

run-ipc: build ## Run in IPC mode
	./$(BINARY) -mode ipc -addr :8080

run-backup: build ## Run backup
	./$(BINARY) -mode backup

help: ## Display this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'
