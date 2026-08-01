BINARY_NAME := netdiag
GO          := go
VERSION     ?= dev
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null)

# ── Detect OS ────────────────────────────────────────────────────────────────
ifeq ($(OS),Windows_NT)
    DETECTED_OS := windows
    DATE        ?= $(shell git log -1 --format=%cI 2>nul || powershell -NoProfile -Command \
                       "[DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')")
    EXT         := .exe
    RMFILE      = if exist $1 del /f $1
    RMDIR       = if exist $1 rmdir /s /q $1
    MKDIR       = if not exist $1 mkdir $1
    INSTALL_PATH := $(USERPROFILE)\AppData\Local\Microsoft\WindowsApps
else
    UNAME       := $(shell uname -s)
    ifeq ($(UNAME),Darwin)
        DETECTED_OS := darwin
    else
        DETECTED_OS := linux
    endif
    DATE        ?= $(shell git log -1 --format=%cI 2>/dev/null || date -u +%Y-%m-%dT%H:%M:%SZ)
    EXT         :=
    RMFILE      = rm -f $1
    RMDIR       = rm -rf $1
    MKDIR       = mkdir -p $1
    INSTALL_PATH := /usr/local/bin
endif

# ── Tool versions (pinned so `make fmt` is reproducible) ─────────────────────
GOFUMPT_VERSION := v0.7.0
GCI_VERSION     := v0.13.5

# ── Build flags ──────────────────────────────────────────────────────────────
LDFLAGS := -s -w \
    -X main.version=$(VERSION) \
    -X main.commit=$(COMMIT) \
    -X main.date=$(DATE)

# -trimpath keeps local paths out of the binary; CGO is not needed for build/
# install/dist, and disabling it there keeps cross-compilation and static
# linking simple. `test` needs cgo for -race, so CGO_ENABLED is scoped to the
# recipes that build binaries rather than exported globally.
BUILDFLAGS := -trimpath -ldflags "$(LDFLAGS)"

OUTPUT := $(BINARY_NAME)$(EXT)

# ── Targets ──────────────────────────────────────────────────────────────────
.PHONY: all help build install uninstall run test test-cover lint fmt deps tidy dist clean

all: build ## Build the binary (default)

help: ## Show this help message
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command \
	  "Select-String -Path Makefile -Pattern '^[a-zA-Z_-]+:.*## ' | \
	   ForEach-Object { \
	     $$parts = $$_.Line -split ':.*## '; \
	     Write-Host ('  {0,-20} {1}' -f $$parts[0], $$parts[1]) \
	   }"
else
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	 awk 'BEGIN {FS=":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
endif

build: ## Compile binary for current OS
	CGO_ENABLED=0 $(GO) build $(BUILDFLAGS) -o $(OUTPUT) .

run: ## Run the project directly with go run
	$(GO) run .

test: ## Run tests (with -race on non-Windows)
ifeq ($(DETECTED_OS),windows)
	$(GO) test -v ./...
else
	$(GO) test -v -race ./...
endif

test-cover: ## Run tests and report coverage per package
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

lint: ## Run golangci-lint with auto-fix
	golangci-lint run --timeout=5m --fix

fmt: ## Format code with gofumpt and gci
	$(GO) run mvdan.cc/gofumpt@$(GOFUMPT_VERSION) -w .
	$(GO) run github.com/daixiang0/gci@$(GCI_VERSION) write \
	    -s standard -s default \
	    -s "prefix(github.com/ARCoder181105/netdiag)" .

deps: ## Download and verify Go modules
	$(GO) mod download
	$(GO) mod verify

tidy: ## Prune and sync go.mod / go.sum
	$(GO) mod tidy

dist: ## Cross-compile for all platforms into dist/
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "if (-not (Test-Path dist)) { New-Item -ItemType Directory dist | Out-Null }"
	set GOOS=linux&& set GOARCH=amd64&& $(GO) build $(BUILDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	set GOOS=linux&& set GOARCH=arm64&& $(GO) build $(BUILDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 .
	set GOOS=darwin&& set GOARCH=amd64&& $(GO) build $(BUILDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .
	set GOOS=darwin&& set GOARCH=arm64&& $(GO) build $(BUILDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .
	set GOOS=windows&& set GOARCH=amd64&& $(GO) build $(BUILDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe .
else
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 $(GO) build $(BUILDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 $(GO) build $(BUILDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe .
endif

install: build ## Install binary to system PATH
ifeq ($(OS),Windows_NT)
	copy /Y $(OUTPUT) "$(INSTALL_PATH)\$(OUTPUT)"
	@echo Installed to $(INSTALL_PATH)\$(OUTPUT)
else
	sudo mv $(OUTPUT) $(INSTALL_PATH)/$(BINARY_NAME)
ifeq ($(DETECTED_OS),linux)
	sudo setcap cap_net_raw+ep $(INSTALL_PATH)/$(BINARY_NAME)
endif
	@echo "Installed to $(INSTALL_PATH)/$(BINARY_NAME)"
endif

uninstall: ## Remove installed binary from system PATH
ifeq ($(OS),Windows_NT)
	@echo Uninstalling $(BINARY_NAME)...
	if exist "$(INSTALL_PATH)\$(OUTPUT)" del /f "$(INSTALL_PATH)\$(OUTPUT)"
	@echo Done.
else
	@echo "Uninstalling $(BINARY_NAME)..."
	sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Uninstalled from $(INSTALL_PATH)/$(BINARY_NAME)"
endif

clean: ## Remove build artifacts
ifeq ($(OS),Windows_NT)
	if exist $(OUTPUT) del /f $(OUTPUT)
	if exist dist rmdir /s /q dist
else
	rm -f $(BINARY_NAME)
	rm -rf dist/
endif
