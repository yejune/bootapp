BINARY_NAME=docker-bootapp
VERSION=1.0.1
BUILD_DIR=build

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w -X github.com/yejune/docker-bootapp/cmd.Version=$(VERSION)"

.PHONY: all build clean test deps install uninstall darwin linux

all: deps build

deps:
	$(GOMOD) tidy

build: darwin linux

darwin:
	@echo "Building for macOS (darwin/amd64)..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	@echo "Building for macOS (darwin/arm64)..."
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .

linux:
	@echo "Building for Linux (linux/amd64)..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	@echo "Building for Linux (linux/arm64)..."
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .

# Build for current platform only
local:
	@echo "Building for current platform..."
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

test:
	$(GOTEST) -v ./...

# Install as Docker CLI plugin and standalone binary
install:
	@echo "Installing Docker CLI plugin..."
	@mkdir -p ~/.docker/cli-plugins
	@if [ "$$(uname -s)" = "Darwin" ]; then \
		if [ "$$(uname -m)" = "arm64" ]; then \
			cp $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ~/.docker/cli-plugins/$(BINARY_NAME); \
			sudo cp $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 /usr/local/bin/bootapp; \
		else \
			cp $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ~/.docker/cli-plugins/$(BINARY_NAME); \
			sudo cp $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 /usr/local/bin/bootapp; \
		fi; \
	elif [ "$$(uname -s)" = "Linux" ]; then \
		if [ "$$(uname -m)" = "aarch64" ]; then \
			cp $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ~/.docker/cli-plugins/$(BINARY_NAME); \
			sudo cp $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 /usr/local/bin/bootapp; \
		else \
			cp $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ~/.docker/cli-plugins/$(BINARY_NAME); \
			sudo cp $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 /usr/local/bin/bootapp; \
		fi; \
	fi
	@chmod +x ~/.docker/cli-plugins/$(BINARY_NAME)
	@sudo chmod +x /usr/local/bin/bootapp
	@echo "✓ Installed to ~/.docker/cli-plugins/$(BINARY_NAME)"
	@echo "✓ Installed to /usr/local/bin/bootapp"
	@echo "You can use: 'docker bootapp' or 'bootapp'"

uninstall:
	@echo "Uninstalling Docker CLI plugin and standalone binary..."
	rm -f ~/.docker/cli-plugins/$(BINARY_NAME)
	sudo rm -f /usr/local/bin/bootapp
	@echo "Uninstalled"

# Show help
help:
	@echo "docker-bootapp - Docker CLI Plugin for multi-project networking"
	@echo ""
	@echo "Usage:"
	@echo "  make deps      - Download dependencies"
	@echo "  make build     - Build for all platforms (darwin/linux)"
	@echo "  make darwin    - Build for macOS only"
	@echo "  make linux     - Build for Linux only"
	@echo "  make local     - Build for current platform"
	@echo "  make install   - Install as Docker CLI plugin"
	@echo "  make uninstall - Remove Docker CLI plugin"
	@echo "  make clean     - Clean build artifacts"
	@echo "  make test      - Run tests"
