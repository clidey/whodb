.PHONY: build build-windows build-mac build-linux clean prepare dev

# Prepare frontend assets
prepare:
	@echo "Cleaning previous build artifacts..."
	@rm -rf build/windows build/darwin build/linux build/bin build/assets build/index.html wails.lock frontend/dist
	@echo "Preparing EE build environment..."
	@mkdir -p frontend/dist
	@echo "Building frontend (EE)..."
	@cd ../frontend && pnpm run build:ee
	@cp -r ../frontend/build/* frontend/dist/

# Development mode
dev: prepare
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails dev -tags ee

# Build for current platform
build: prepare
	@echo "Building WhoDB EE Desktop for current platform..."
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -tags ee -skipbindings -ldflags="-s -w" -o whodb-ee

# Build for all platforms
build-all: build-windows build-mac build-linux

# Windows builds
build-windows: prepare
	@echo "Building WhoDB EE for Windows AMD64..."
	@mkdir -p build/windows/amd64
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -platform windows/amd64 \
		-tags ee -windowsconsole=false -skipbindings -ldflags="-s -w" -o whodb-ee.exe
	@mv build/bin/whodb-ee.exe build/windows/amd64/
	@echo "Building WhoDB EE for Windows ARM64..."
	@mkdir -p build/windows/arm64
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -platform windows/arm64 \
		-tags ee -windowsconsole=false -skipbindings -ldflags="-s -w" -o whodb-ee.exe
	@mv build/bin/whodb-ee.exe build/windows/arm64/

# macOS builds
build-mac: prepare
	@echo "Building WhoDB EE for macOS Universal..."
	@mkdir -p build/darwin/universal
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -platform darwin/universal \
		-tags ee -skipbindings -ldflags="-s -w" -o whodb-ee
	@mv "build/bin/WhoDB EE.app" build/darwin/universal/ 2>/dev/null || mv build/bin/whodb-ee build/darwin/universal/

# Linux builds
build-linux: prepare
	@echo "Building WhoDB EE for Linux (native architecture)..."
	@mkdir -p build/linux/$$(uname -m)
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean \
		-tags "ee webkit2_41" -skipbindings -ldflags="-s -w" -o whodb-ee
	@mv build/bin/whodb-ee build/linux/$$(uname -m)/
	@echo "Note: Cross-compilation for other architectures requires proper toolchain setup"

# Linux AMD64 build (use on AMD64 systems only)
build-linux-amd64: prepare
	@echo "Building WhoDB EE for Linux AMD64..."
	@mkdir -p build/linux/amd64
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -platform linux/amd64 \
		-tags "ee webkit2_41" -skipbindings -ldflags="-s -w" -o whodb-ee
	@mv build/bin/whodb-ee build/linux/amd64/

# Linux ARM64 build (use on ARM64 systems only)
build-linux-arm64: prepare
	@echo "Building WhoDB EE for Linux ARM64..."
	@mkdir -p build/linux/arm64
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -platform linux/arm64 \
		-tags "ee webkit2_41" -skipbindings -ldflags="-s -w" -o whodb-ee
	@mv build/bin/whodb-ee build/linux/arm64/

# Production builds with installers
build-prod-windows: prepare
	@echo "Building WhoDB EE Windows installer for AMD64..."
	@mkdir -p build/windows/amd64
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -platform windows/amd64 \
		-tags ee -nsis -windowsconsole=false -skipbindings -ldflags="-s -w -H windowsgui" \
		-o whodb-ee-installer.exe
	@mv build/bin/whodb-ee-installer.exe build/windows/amd64/

build-prod-mac: prepare
	@echo "Building WhoDB EE macOS app package..."
	@mkdir -p build/darwin/universal
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -platform darwin/universal \
		-tags ee -pkg -skipbindings -ldflags="-s -w" -o whodb-ee
	@mv build/bin/whodb-ee.pkg build/darwin/universal/ 2>/dev/null || mv "build/bin/WhoDB EE.app" build/darwin/universal/

build-prod-linux: prepare
	@echo "Building WhoDB EE Linux AppImage for AMD64..."
	@mkdir -p build/linux/amd64
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -platform linux/amd64 \
		-tags "ee webkit2_41" -skipbindings -ldflags="-s -w" -o whodb-ee
	@mv build/bin/whodb-ee build/linux/amd64/

# Backup Linux builds (without webkit2_41 tag - for compatibility)
build-linux-compat: prepare
	@echo "Building WhoDB EE for Linux (native architecture, compatibility mode)..."
	@mkdir -p build/linux/$$(uname -m)-compat
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean \
		-tags ee -skipbindings -ldflags="-s -w" -o whodb-ee-compat
	@mv build/bin/whodb-ee-compat build/linux/$$(uname -m)-compat/

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
	@echo "WhoDB EE Desktop Build System"
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