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

# ldflags for version injection
VERSION_PKG=github.com/clidey/whodb/cli/pkg/version
LDFLAGS=-ldflags "-s -w -X $(VERSION_PKG).Version=$(VERSION) -X $(VERSION_PKG).Commit=$(COMMIT) -X $(VERSION_PKG).BuildDate=$(BUILD_DATE)"

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

build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
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

.DEFAULT_GOAL := help
