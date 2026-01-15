BINARY_NAME=netdiag
GO=go
GOFLAGS=-ldflags="-s -w"
INSTALL_PATH=/usr/local/bin

.PHONY: all build test clean run deps lint help install

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) main.go

install: build ## Build and install locally
	sudo mv $(BINARY_NAME) $(INSTALL_PATH)
	@if [ "$$(uname)" = "Linux" ]; then sudo setcap cap_net_raw+ep $(INSTALL_PATH)/$(BINARY_NAME); fi
	@echo "Installed to $(INSTALL_PATH)/$(BINARY_NAME)"

test: ## Run tests
	$(GO) test -v -race ./...

lint: ## Run linter
	golangci-lint run

clean: ## Remove binaries
	rm -f $(BINARY_NAME)
	rm -rf dist/

deps: ## Download dependencies
	$(GO) mod download
	$(GO) mod verify