BINARY_NAME=netdiag
GO=go
GOFLAGS=-ldflags="-s -w"
INSTALL_PATH=/usr/local/bin

.PHONY: help
help: ## Display this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME)
	@echo "Build complete: ./$(BINARY_NAME)"

.PHONY: install
install: build ## Build and install to /usr/local/bin with ICMP capabilities
	@echo "Installing to $(INSTALL_PATH)..."
	@sudo mv $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@sudo chmod +x $(INSTALL_PATH)/$(BINARY_NAME)
	@if [ "$$(uname)" = "Linux" ]; then \
		echo "Setting ICMP capabilities..."; \
		sudo setcap cap_net_raw+ep $(INSTALL_PATH)/$(BINARY_NAME); \
	fi
	@echo "Installed to $(INSTALL_PATH)/$(BINARY_NAME)"

.PHONY: test
test: ## Run tests with verbose output
	@echo "Running tests..."
	$(GO) test -v ./...

.PHONY: test-cover
test-cover: ## Run tests with coverage and generate HTML report
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: lint
lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin"; \
		exit 1; \
	fi

.PHONY: fmt
fmt: ## Format code with go fmt and gofmt
	@echo "Formatting code..."
	$(GO) fmt ./...
	gofmt -s -w .
	@echo "Code formatted"

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...

.PHONY: clean
clean: ## Remove binaries and coverage files
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	@rm -f coverage.out coverage.html
	@rm -rf dist/
	@echo "Clean complete"

.PHONY: run-ping
run-ping: build ## Quick test: ping google.com
	./$(BINARY_NAME) ping google.com

.PHONY: run-scan
run-scan: build ## Quick test: scan localhost ports 1-1024
	./$(BINARY_NAME) scan localhost -p 1-1024

.PHONY: run-speedtest
run-speedtest: build ## Quick test: run speedtest
	./$(BINARY_NAME) speedtest

.PHONY: dev
dev: build ## Run with custom command (usage: make dev CMD="ping google.com")
	./$(BINARY_NAME) $(CMD)

.PHONY: deps
deps: ## Download and verify dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download
	@echo "Verifying dependencies..."
	$(GO) mod verify
	@echo "Dependencies ready"

.PHONY: tidy
tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	$(GO) mod tidy
	@echo "Modules tidied"

.PHONY: build-all
build-all: ## Cross-compile for all platforms to dist/ directory
	@echo "Building for all platforms..."
	@mkdir -p dist
	@echo "  Linux amd64..."
	@GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o dist/$(BINARY_NAME)-linux-amd64
	@echo "  Linux arm64..."
	@GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -o dist/$(BINARY_NAME)-linux-arm64
	@echo "  macOS amd64 (Intel)..."
	@GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64
	@echo "  macOS arm64 (Apple Silicon)..."
	@GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64
	@echo "  Windows amd64..."
	@GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe
	@echo "Build complete! Binaries are in ./dist/"

.PHONY: pre-commit
pre-commit: fmt vet lint test ## Run fmt, vet, lint, and test before committing
	@echo "Pre-commit checks complete!"
