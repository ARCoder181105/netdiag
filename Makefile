BINARY_NAME := netdiag
GO := go
INSTALL_PATH := /usr/local/bin

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

.PHONY: all help build install test lint clean deps run dist

all: build

help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) .

install: build
	sudo mv $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@if [ "$$(uname)" = "Linux" ]; then \
		sudo setcap cap_net_raw+ep $(INSTALL_PATH)/$(BINARY_NAME); \
	fi
	@echo "Installed to $(INSTALL_PATH)/$(BINARY_NAME)"

run:
	$(GO) run .

test:
	@if [ "$$($(GO) env GOOS)" = "windows" ]; then \
		$(GO) test -v ./... ; \
	else \
		$(GO) test -v -race ./... ; \
	fi

lint:
	golangci-lint run --timeout=5m --fix

deps:
	$(GO) mod download
	$(GO) mod verify

dist:
	mkdir -p dist
	GOOS=linux   GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux   GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)-linux-arm64 .
	GOOS=darwin  GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)-windows-amd64.exe .

clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/

fmt:
	$(GO) run mvdan.cc/gofumpt@latest -w .
	$(GO) run github.com/daixiang0/gci@latest write -s standard -s default -s "prefix(github.com/ARCoder181105/netdiag)" .

