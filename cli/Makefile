# Copyright 2025 Clidey, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# WhoDB CLI Makefile

.PHONY: build clean install test run dev help docker-build docker-run docker-compose-up docker-compose-down

BINARY_NAME=clidey/whodb-cli
BUILD_DIR=build
INSTALL_PATH=/usr/local/bin
DOCKER_IMAGE=clidey/whodb-cli:latest

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BAML_VERSION ?= $(shell grep 'github.com/boundaryml/baml ' ../core/go.mod | awk '{print $$2}' | sed 's/^v//')

# ldflags for version injection
VERSION_PKG=github.com/clidey/whodb/cli/pkg/version
BAML_PKG=github.com/clidey/whodb/cli/internal/baml
LDFLAGS=-ldflags "-s -w -X $(VERSION_PKG).Version=$(VERSION) -X $(VERSION_PKG).Commit=$(COMMIT) -X $(VERSION_PKG).BuildDate=$(BUILD_DATE) -X $(BAML_PKG).BAMLVersion=$(BAML_VERSION)"

help:
	@echo "WhoDB CLI - Makefile targets:"
	@echo "  make build              - Build the CLI binary"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make install            - Install binary to $(INSTALL_PATH)"
	@echo "  make test               - Run tests"
	@echo "  make run                - Build and run in interactive mode"
	@echo "  make dev                - Run in development mode"
	@echo "  make lint               - Run linters"
	@echo ""
	@echo "Docker targets:"
	@echo "  make docker-build       - Build Docker image"
	@echo "  make docker-run         - Run CLI in Docker (interactive)"
	@echo "  make docker-compose-up  - Start CLI and test databases"
	@echo "  make docker-compose-down - Stop all Docker services"
	@echo ""
	@echo "CI build targets (use VERSION=x.x.x):"
	@echo "  make ci-build-linux-amd64   - Build for Linux x86_64"
	@echo "  make ci-build-linux-arm64   - Build for Linux ARM64"
	@echo "  make ci-build-linux-armv7   - Build for Linux ARMv7 (requires cross-compiler)"
	@echo "  make ci-build-darwin-amd64  - Build for macOS Intel"
	@echo "  make ci-build-darwin-arm64  - Build for macOS Apple Silicon"
	@echo "  make ci-build-windows-amd64 - Build for Windows x86_64"
	@echo "  make ci-build-windows-arm64 - Build for Windows ARM64"

build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@go build -trimpath $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_PATH)/
	@echo "Installation complete"

test:
	@echo "Running tests..."
	@go test -v ./...

run: build
	@$(BUILD_DIR)/$(BINARY_NAME)

dev:
	@go run .

lint:
	@echo "Running linters..."
	@golangci-lint run ./...

docker-build:
	@echo "Building Docker image..."
	@cd .. && docker build -t $(DOCKER_IMAGE) -f cli/Dockerfile .
	@echo "Docker image built: $(DOCKER_IMAGE)"

docker-run:
	@echo "Running CLI in Docker..."
	@docker run -it --rm $(DOCKER_IMAGE)

docker-compose-up:
	@echo "Starting CLI and test databases..."
	@docker-compose up -d
	@echo "Services started. Use 'docker-compose exec whodb-cli whodb-cli' to connect"

docker-compose-down:
	@echo "Stopping Docker services..."
	@docker-compose down
	@echo "Services stopped"

# CI build targets for cross-platform builds
# Usage: make ci-build-<platform> VERSION=x.x.x
# Outputs binary to BUILD_DIR with platform suffix

.PHONY: ci-build-linux-amd64 ci-build-linux-arm64 ci-build-linux-armv7
.PHONY: ci-build-darwin-amd64 ci-build-darwin-arm64
.PHONY: ci-build-windows-amd64 ci-build-windows-arm64

CI_LDFLAGS=-trimpath -ldflags "-s -w -X $(VERSION_PKG).Version=$(VERSION) -X $(VERSION_PKG).Commit=$(COMMIT) -X $(VERSION_PKG).BuildDate=$(BUILD_DATE) -X $(BAML_PKG).BAMLVersion=$(BAML_VERSION)"

ci-build-linux-amd64:
	@echo "Building CLI for linux/amd64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build $(CI_LDFLAGS) -o $(BUILD_DIR)/whodb-cli-linux-amd64 .
	@echo "Built: $(BUILD_DIR)/whodb-cli-linux-amd64"

ci-build-linux-arm64:
	@echo "Building CLI for linux/arm64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build $(CI_LDFLAGS) -o $(BUILD_DIR)/whodb-cli-linux-arm64 .
	@echo "Built: $(BUILD_DIR)/whodb-cli-linux-arm64"

ci-build-linux-armv7:
	@echo "Building CLI for linux/armv7..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=7 CC=arm-linux-gnueabihf-gcc go build $(CI_LDFLAGS) -o $(BUILD_DIR)/whodb-cli-linux-armv7 .
	@echo "Built: $(BUILD_DIR)/whodb-cli-linux-armv7"

ci-build-darwin-amd64:
	@echo "Building CLI for darwin/amd64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build $(CI_LDFLAGS) -o $(BUILD_DIR)/whodb-cli-darwin-amd64 .
	@echo "Built: $(BUILD_DIR)/whodb-cli-darwin-amd64"

ci-build-darwin-arm64:
	@echo "Building CLI for darwin/arm64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build $(CI_LDFLAGS) -o $(BUILD_DIR)/whodb-cli-darwin-arm64 .
	@echo "Built: $(BUILD_DIR)/whodb-cli-darwin-arm64"

ci-build-windows-amd64:
	@echo "Building CLI for windows/amd64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build $(CI_LDFLAGS) -o $(BUILD_DIR)/whodb-cli-windows-amd64.exe .
	@echo "Built: $(BUILD_DIR)/whodb-cli-windows-amd64.exe"

ci-build-windows-arm64:
	@echo "Building CLI for windows/arm64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=windows GOARCH=arm64 go build $(CI_LDFLAGS) -o $(BUILD_DIR)/whodb-cli-windows-arm64.exe .
	@echo "Built: $(BUILD_DIR)/whodb-cli-windows-arm64.exe"

.DEFAULT_GOAL := help
