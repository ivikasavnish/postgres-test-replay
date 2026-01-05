.PHONY: all build clean test fmt lint docker-up docker-down install help setup

# Binary name
BINARY=postgres-test-replay
SETUP_BINARY=pgparsers
CMD_PATH=./cmd/postgres-test-replay
SETUP_CMD_PATH=./cmd/pgparsers

# Build variables
BUILD_FLAGS=-ldflags="-s -w"
GO=go

all: fmt build

build: ## Build the application
	$(GO) build $(BUILD_FLAGS) -o $(BINARY) $(CMD_PATH)

build-setup: ## Build the setup tool
	$(GO) build $(BUILD_FLAGS) -o $(SETUP_BINARY) $(SETUP_CMD_PATH)

setup: build-setup ## Run automated Docker Compose setup
	./$(SETUP_BINARY)

clean: ## Clean build artifacts
	rm -f $(BINARY) $(SETUP_BINARY)
	rm -rf waldata/ backups/ sessions/ checkpoints/ *.log

clean-docker: ## Clean Docker data directories
	rm -rf data/ wal/

test: ## Run tests
	$(GO) test -v ./...

fmt: ## Format code
	$(GO) fmt ./...

lint: ## Run linter
	golangci-lint run || go vet ./...

install: ## Install dependencies
	$(GO) mod download
	$(GO) mod tidy

docker-up: ## Start Docker Compose services (manual)
	docker-compose up -d

docker-down: ## Stop Docker Compose services
	docker-compose down -v

docker-logs: ## Show Docker Compose logs
	docker-compose logs -f

run-listener: build ## Run in listener mode
	./$(BINARY) -mode listener

run-ipc: build ## Run in IPC mode (use SERVER_PORT from .env)
	./$(BINARY) -mode ipc

run-backup: build ## Run backup
	./$(BINARY) -mode backup

help: ## Display this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'
