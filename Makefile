# Binary name
BINARY_NAME=ragie-cli
VERSION ?= $(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOVET=$(GOCMD) vet
GOGET=$(GOCMD) get

# Build flags
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# Output directory
DIST_DIR=dist

# Supported platforms
PLATFORMS=darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64

# Clean up
.PHONY: clean
clean:
	rm -rf $(DIST_DIR)
	rm -f $(BINARY_NAME)

# Install dependencies
.PHONY: deps
deps:
	$(GOGET) -v ./...
	$(GOMOD) tidy

# Run tests
.PHONY: test
test:
	$(GOTEST) -v ./...

# Run integration tests
.PHONY: integration-test
integration-test:
	INTEGRATION_TEST=true $(GOTEST) -v ./integration_test

# Run linter
.PHONY: lint
lint:
	$(GOVET) ./...

# Build for the current platform
.PHONY: build
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)

# Build all platforms
.PHONY: build-all
build-all: clean
	mkdir -p $(DIST_DIR)
	$(foreach platform,$(PLATFORMS),\
		$(eval GOOS=$(word 1,$(subst /, ,$(platform))))\
		$(eval GOARCH=$(word 2,$(subst /, ,$(platform))))\
		$(eval EXTENSION=$(if $(filter windows,$(GOOS)),.exe))\
		$(eval BINARY=$(DIST_DIR)/$(BINARY_NAME)_$(GOOS)_$(GOARCH)$(EXTENSION))\
		GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) $(LDFLAGS) -o $(BINARY) && \
		shasum -a 256 $(BINARY) > $(BINARY).sha256 ;\
	)

# Create release archives
.PHONY: release
release: build-all
	$(foreach platform,$(PLATFORMS),\
		$(eval GOOS=$(word 1,$(subst /, ,$(platform))))\
		$(eval GOARCH=$(word 2,$(subst /, ,$(platform))))\
		$(eval EXTENSION=$(if $(filter windows,$(GOOS)),.exe))\
		$(eval BINARY=$(DIST_DIR)/$(BINARY_NAME)_$(GOOS)_$(GOARCH)$(EXTENSION))\
		$(eval ARCHIVE=$(DIST_DIR)/$(BINARY_NAME)_$(GOOS)_$(GOARCH).tar.gz)\
		tar -czf $(ARCHIVE) -C $(DIST_DIR) $(notdir $(BINARY)) $(notdir $(BINARY)).sha256 ;\
	)

# Run all checks (test, lint)
.PHONY: check
check: test lint

# Full build process
.PHONY: all
all: deps check build-all release

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build for current platform"
	@echo "  build-all   - Build for all platforms"
	@echo "  release     - Create release archives"
	@echo "  clean       - Remove built files"
	@echo "  deps        - Install dependencies"
	@echo "  test        - Run tests"
	@echo "  integration-test - Run integration tests"
	@echo "  lint        - Run linter"
	@echo "  check       - Run all checks"
	@echo "  all         - Full build process"
	@echo "  help        - Show this help" 