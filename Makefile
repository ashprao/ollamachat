# Variables
HOST_OUTPUT_DIR = fyne-cross/dist/darwin-arm64
APP_ID = github.com-ashprao-ollamachat

# Platforms and Architectures for cross-compilation
CROSS_PLATFORMS = windows linux
CROSS_ARCHITECTURES = amd64 arm64
DARWIN_CROSS_ARCH = amd64

# Default target - clean and build for all platforms including host
all: clean cross-platform darwin-amd64 darwin-arm64

# Quick development build (local platform only)
build:
	@echo "Building for development (local platform)..."
	go build -o ollamachat cmd/ollamachat/main.go
	@echo "Build complete: ./ollamachat"

# Darwin ARM64 target (local host system)
darwin-arm64:
	@echo "Building for Darwin ARM64 (host system)..."
	@echo "Cleaning Darwin ARM64 build directory..."
	@rm -rf $(HOST_OUTPUT_DIR)
	@mkdir -p $(HOST_OUTPUT_DIR)
	@echo "Building binary..."
	go build -o $(HOST_OUTPUT_DIR)/ollamachat cmd/ollamachat/main.go
	@echo "Packaging app..."
	fyne package -os darwin --app-id $(APP_ID) --exe $(HOST_OUTPUT_DIR)/ollamachat
	@echo "Moving app to correct location..."
	@mv ollamachat.app $(HOST_OUTPUT_DIR)/

# Darwin AMD64 - Skip due to fyne-cross CLI flag incompatibility
darwin-amd64:
	@echo "⚠️  Darwin AMD64 build skipped due to fyne-cross/Fyne CLI compatibility issue"
	@echo "   Problem: fyne-cross v1.6.1 uses -appBuild but Fyne CLI v1.6.1 expects --app-build"
	@echo "   Solutions:"
	@echo "     1. Use 'make darwin-amd64-manual' to attempt manual build"
	@echo "     2. Build on an Intel Mac with 'fyne package -os darwin'"
	@echo "     3. Wait for updated fyne-cross version that fixes CLI compatibility"
	@mkdir -p fyne-cross/dist/darwin-amd64
	@echo "Darwin AMD64 build skipped due to CLI flag incompatibility" > fyne-cross/dist/darwin-amd64/SKIPPED.txt

# Manual Darwin AMD64 build (experimental workaround)
darwin-amd64-manual:
	@echo "⚠️  Attempting manual Darwin AMD64 build..."
	@echo "   This is experimental and may not work due to OpenGL cross-compilation limitations"
	@echo "Cleaning Darwin AMD64 build directory..."
	@rm -rf fyne-cross/dist/darwin-amd64
	@mkdir -p fyne-cross/dist/darwin-amd64
	@echo "Trying direct fyne package with manual flags (may fail)..."
	@if command -v fyne >/dev/null 2>&1; then \
		echo "Building with manual app-build workaround..."; \
		cd fyne-cross/dist/darwin-amd64 && \
		GOOS=darwin GOARCH=amd64 fyne package \
			--os darwin \
			--name ollamachat \
			--app-id $(APP_ID) \
			--app-version "1.0.0" \
			--app-build 1 \
			--icon ../../../Icon.png \
			../../../cmd/ollamachat/main.go || \
		echo "❌ Manual build failed - OpenGL cross-compilation not supported"; \
	else \
		echo "❌ Fyne CLI not found. Install with: go install fyne.io/fyne/v2/cmd/fyne@latest"; \
	fi

# Cross-platform target using fyne-cross (Windows, Linux)
cross-platform:
	@echo "Building cross-platform targets..."
	@for platform in $(CROSS_PLATFORMS); do \
		for arch in $(CROSS_ARCHITECTURES); do \
			echo "Cleaning $$platform/$$arch build directory..."; \
			rm -rf fyne-cross/dist/$$platform-$$arch; \
			echo "Building $$platform/$$arch..."; \
			fyne-cross $$platform -arch $$arch --app-id $(APP_ID) ./cmd/ollamachat; \
		done; \
	done

# Individual platform targets
windows:
	@echo "Building for Windows (all architectures)..."
	@for arch in $(CROSS_ARCHITECTURES); do \
		echo "Cleaning Windows/$$arch build directory..."; \
		rm -rf fyne-cross/dist/windows-$$arch; \
		echo "Building Windows/$$arch..."; \
		fyne-cross windows -arch $$arch --app-id $(APP_ID) ./cmd/ollamachat; \
	done

linux:
	@echo "Building for Linux (all architectures)..."
	@for arch in $(CROSS_ARCHITECTURES); do \
		echo "Cleaning Linux/$$arch build directory..."; \
		rm -rf fyne-cross/dist/linux-$$arch; \
		echo "Building Linux/$$arch..."; \
		fyne-cross linux -arch $$arch --app-id $(APP_ID) ./cmd/ollamachat; \
	done

# Platform-specific target
platform:
	@if [ -z "$(PLATFORM)" ]; then \
		echo "Error: PLATFORM variable is not set"; \
		exit 1; \
	fi; \
	for arch in $(ARCHITECTURES); do \
		fyne-cross $(PLATFORM) -arch $$arch --app-id $(APP_ID) ./cmd/ollamachat; \
	done

# Platform-architecture-specific target
platform-arch:
	@if [ -z "$(PLATFORM)" ] || [ -z "$(ARCH)" ]; then \
		echo "Error: PLATFORM and ARCH variables must be set"; \
		exit 1; \
	fi; \
	fyne-cross $(PLATFORM) -arch $(ARCH) --app-id $(APP_ID) ./cmd/ollamachat

# Clean target
clean:
	rm -rf fyne-cross

.PHONY: all build clean platform platform-arch darwin-arm64 darwin-amd64 darwin-amd64-manual cross-platform windows linux
