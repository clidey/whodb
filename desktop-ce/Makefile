.PHONY: build build-windows build-mac build-linux clean prepare dev

# Prepare frontend assets
prepare:
	@echo "Cleaning previous build artifacts..."
	@rm -rf build/windows build/darwin build/linux build/bin build/assets build/index.html wails.lock frontend/dist
	@echo "Preparing CE build environment..."
	@mkdir -p frontend/dist
	@echo "Building frontend..."
	@cd ../frontend && pnpm run build:ce
	@cp -r ../frontend/build/* frontend/dist/

# Development mode
dev: prepare
	GOWORK=$$PWD/../go.work.desktop-ce ~/go/bin/wails dev

# Build for current platform
build: prepare
	@echo "Building WhoDB CE Desktop for current platform..."
	GOWORK=$$PWD/../go.work.desktop-ce ~/go/bin/wails build -clean -skipbindings -ldflags="-s -w" -o whodb-ce

# Build for all platforms
build-all: build-windows build-mac build-linux

# Windows builds
build-windows: prepare
	@echo "Building WhoDB CE for Windows AMD64..."
	@mkdir -p build/windows/amd64
	GOWORK=$$PWD/../go.work.desktop-ce ~/go/bin/wails build -clean -platform windows/amd64 \
		-windowsconsole=false -skipbindings -ldflags="-s -w" -o whodb-ce.exe
	@mv build/bin/whodb-ce.exe build/windows/amd64/
	@echo "Building WhoDB CE for Windows ARM64..."
	@mkdir -p build/windows/arm64
	GOWORK=$$PWD/../go.work.desktop-ce ~/go/bin/wails build -clean -platform windows/arm64 \
		-windowsconsole=false -skipbindings -ldflags="-s -w" -o whodb-ce.exe
	@mv build/bin/whodb-ce.exe build/windows/arm64/

# macOS builds
build-mac: prepare
	@echo "Building WhoDB CE for macOS Universal..."
	@mkdir -p build/darwin/universal
	GOWORK=$$PWD/../go.work.desktop-ce ~/go/bin/wails build -clean -platform darwin/universal \
		-skipbindings -ldflags="-s -w" -o whodb-ce
	@mv "build/bin/WhoDB CE.app" build/darwin/universal/ 2>/dev/null || mv build/bin/whodb-ce build/darwin/universal/

# Linux builds
build-linux: prepare
	@echo "Building WhoDB CE for Linux (native architecture)..."
	@mkdir -p build/linux/$$(uname -m)
	GOWORK=$$PWD/../go.work.desktop-ce ~/go/bin/wails build -clean \
		-tags webkit2_41 -skipbindings -ldflags="-s -w" -o whodb-ce
	@mv build/bin/whodb-ce build/linux/$$(uname -m)/
	@echo "Note: Cross-compilation for other architectures requires proper toolchain setup"

# Linux AMD64 build (use on AMD64 systems only)
build-linux-amd64: prepare
	@echo "Building WhoDB CE for Linux AMD64..."
	@mkdir -p build/linux/amd64
	GOWORK=$$PWD/../go.work.desktop-ce ~/go/bin/wails build -clean -platform linux/amd64 \
		-tags webkit2_41 -skipbindings -ldflags="-s -w" -o whodb-ce
	@mv build/bin/whodb-ce build/linux/amd64/

# Linux ARM64 build (use on ARM64 systems only)
build-linux-arm64: prepare
	@echo "Building WhoDB CE for Linux ARM64..."
	@mkdir -p build/linux/arm64
	GOWORK=$$PWD/../go.work.desktop-ce ~/go/bin/wails build -clean -platform linux/arm64 \
		-tags webkit2_41 -skipbindings -ldflags="-s -w" -o whodb-ce
	@mv build/bin/whodb-ce build/linux/arm64/

# Production builds with installers
build-prod-windows: prepare
	@echo "Building WhoDB CE Windows installer for AMD64..."
	@mkdir -p build/windows/amd64
	GOWORK=$$PWD/../go.work.desktop-ce ~/go/bin/wails build -clean -platform windows/amd64 \
		-nsis -windowsconsole=false -skipbindings -ldflags="-s -w -H windowsgui" \
		-o whodb-ce-installer.exe
	@mv build/bin/whodb-ce-installer.exe build/windows/amd64/

build-prod-mac: prepare
	@echo "Building WhoDB CE macOS app package..."
	@mkdir -p build/darwin/universal
	GOWORK=$$PWD/../go.work.desktop-ce ~/go/bin/wails build -clean -platform darwin/universal \
		-pkg -skipbindings -ldflags="-s -w" -o whodb-ce
	@mv build/bin/whodb-ce.pkg build/darwin/universal/ 2>/dev/null || mv "build/bin/WhoDB CE.app" build/darwin/universal/

build-prod-linux: prepare
	@echo "Building WhoDB CE Linux AppImage for AMD64..."
	@mkdir -p build/linux/amd64
	GOWORK=$$PWD/../go.work.desktop-ce ~/go/bin/wails build -clean -platform linux/amd64 \
		-tags webkit2_41 -skipbindings -ldflags="-s -w" -o whodb-ce
	@mv build/bin/whodb-ce build/linux/amd64/

# Backup Linux builds (without webkit2_41 tag - for compatibility)
build-linux-compat: prepare
	@echo "Building WhoDB CE for Linux (native architecture, compatibility mode)..."
	@mkdir -p build/linux/$$(uname -m)-compat
	GOWORK=$$PWD/../go.work.desktop-ce ~/go/bin/wails build -clean \
		-skipbindings -ldflags="-s -w" -o whodb-ce-compat
	@mv build/bin/whodb-ce-compat build/linux/$$(uname -m)-compat/

# Clean build artifacts (standalone clean command)
clean:
	@echo "Cleaning all build artifacts..."
	@rm -rf build/windows build/darwin build/linux build/bin build/assets build/index.html wails.lock frontend/dist

# Install Wails CLI
install-wails:
	go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Check Wails dependencies
doctor:
	~/go/bin/wails doctor

# Help target
help:
	@echo "WhoDB CE Desktop Build System"
	@echo "============================="
	@echo ""
	@echo "Targets:"
	@echo "  make build           - Build for current platform"
	@echo "  make build-all       - Build for all platforms"
	@echo "  make build-windows   - Build for Windows (AMD64 & ARM64)"
	@echo "  make build-mac       - Build for macOS (Universal)"
	@echo "  make build-linux     - Build for Linux (AMD64 & ARM64)"
	@echo "  make dev            - Run in development mode"
	@echo ""
	@echo "Production Builds (with installers):"
	@echo "  make build-prod-windows  - Windows installer"
	@echo "  make build-prod-mac      - macOS package"
	@echo "  make build-prod-linux    - Linux AppImage"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make doctor         - Check Wails dependencies"
	@echo "  make help           - Show this help message"