# Variables
OUTPUT_DIR = builds
APP_ID = github.com-ashprao-ollamachat

# Platforms and Architectures
PLATFORMS = windows linux darwin
ARCHITECTURES = amd64 arm64

# Default target
all:
	@for platform in $(PLATFORMS); do \
		for arch in $(ARCHITECTURES); do \
			fyne-cross $$platform -arch $$arch --app-id $(APP_ID); \
		done; \
	done

# Platform-specific target
platform:
	@if [ -z "$(PLATFORM)" ]; then \
		echo "Error: PLATFORM variable is not set"; \
		exit 1; \
	fi; \
	for arch in $(ARCHITECTURES); do \
		fyne-cross $(PLATFORM) -arch $$arch --app-id $(APP_ID); \
	done

# Platform-architecture-specific target
platform-arch:
	@if [ -z "$(PLATFORM)" ] || [ -z "$(ARCH)" ]; then \
		echo "Error: PLATFORM and ARCH variables must be set"; \
		exit 1; \
	fi; \
	fyne-cross $(PLATFORM) -arch $(ARCH) --app-id $(APP_ID)

# Clean target
clean:
	rm -rf $(OUTPUT_DIR)

.PHONY: all clean platform platform-arch
