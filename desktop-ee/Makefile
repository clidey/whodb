.PHONY: build build-windows build-mac build-linux clean prepare dev

# Package metadata (override via environment if desired)
VERSION ?= $(shell date +%Y.%m.%d.%H%M%S)
PKG_ID ?= com.clidey.whodb.ee
NOTARY_PROFILE ?=
APPLE_ID ?=
TEAM_ID ?=
APPLE_APP_PASSWORD ?=

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
	ENVIRONMENT=dev GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails dev -tags ee

# Build for current platform
build: prepare
	@echo "Building WhoDB EE Desktop for current platform..."
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -tags ee -ldflags="-s -w" -o whodb-ee

# Build for all platforms
build-all: build-windows build-mac build-linux

# Windows builds
build-windows: prepare
	@echo "Building WhoDB EE for Windows AMD64..."
	@mkdir -p build/windows/amd64
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -platform windows/amd64 \
		-tags ee -windowsconsole=false -ldflags="-s -w" -o whodb-ee.exe
	@mv build/bin/whodb-ee.exe build/windows/amd64/
	@echo "Building WhoDB EE for Windows ARM64..."
	@mkdir -p build/windows/arm64
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -platform windows/arm64 \
		-tags ee -windowsconsole=false -ldflags="-s -w" -o whodb-ee.exe
	@mv build/bin/whodb-ee.exe build/windows/arm64/

# macOS builds
build-mac: prepare
	@echo "Building WhoDB EE for macOS Universal..."
	@mkdir -p build/darwin/universal
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -platform darwin/universal \
		-tags ee -ldflags="-s -w" -o whodb-ee
	@mv "build/bin/WhoDB - Enterprise.app" build/darwin/universal/ 2>/dev/null || mv build/bin/whodb-ee build/darwin/universal/
	@if [ -f "build/darwin/universal/WhoDB - Enterprise.app/Contents/Info.plist" ]; then \
		/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $(VERSION)" "build/darwin/universal/WhoDB - Enterprise.app/Contents/Info.plist" || true; \
		/usr/libexec/PlistBuddy -c "Set :CFBundleVersion $(VERSION)" "build/darwin/universal/WhoDB - Enterprise.app/Contents/Info.plist" || true; \
	fi

# Linux builds
build-linux: prepare
	@echo "Building WhoDB EE for Linux (native architecture)..."
	@mkdir -p build/linux/$$(uname -m)
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean \
		-tags "ee webkit2_41" -ldflags="-s -w" -o whodb-ee
	@mv build/bin/whodb-ee build/linux/$$(uname -m)/
	@echo "Note: Cross-compilation for other architectures requires proper toolchain setup"

# Linux AMD64 build (use on AMD64 systems only)
build-linux-amd64: prepare
	@echo "Building WhoDB EE for Linux AMD64..."
	@mkdir -p build/linux/amd64
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -platform linux/amd64 \
		-tags "ee webkit2_41" -ldflags="-s -w" -o whodb-ee
	@mv build/bin/whodb-ee build/linux/amd64/

# Linux ARM64 build (use on ARM64 systems only)
build-linux-arm64: prepare
	@echo "Building WhoDB EE for Linux ARM64..."
	@mkdir -p build/linux/arm64
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -platform linux/arm64 \
		-tags "ee webkit2_41" -ldflags="-s -w" -o whodb-ee
	@mv build/bin/whodb-ee build/linux/arm64/

# Production builds with installers
build-prod-windows: prepare
	@echo "Building WhoDB EE Windows installer for AMD64..."
	@mkdir -p build/windows/amd64
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -platform windows/amd64 \
		-tags ee -nsis -obfuscated -upx -windowsconsole=false -ldflags="-s -w -H windowsgui" \
		-o whodb-ee-installer.exe
	@mv build/bin/whodb-ee-installer.exe build/windows/amd64/

build-prod-mac: prepare
	@echo "Building WhoDB EE macOS app (no pkg flag)..."
	@mkdir -p build/darwin/universal
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -platform darwin/universal \
			-tags ee -ldflags="-s -w" -o whodb-ee
	@mv build/bin/whodb-ee.pkg build/darwin/universal/ 2>/dev/null || mv "build/bin/WhoDB - Enterprise.app" build/darwin/universal/
	@if [ -f "build/darwin/universal/WhoDB - Enterprise.app/Contents/Info.plist" ]; then \
		/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $(VERSION)" "build/darwin/universal/WhoDB - Enterprise.app/Contents/Info.plist" || true; \
		/usr/libexec/PlistBuddy -c "Set :CFBundleVersion $(VERSION)" "build/darwin/universal/WhoDB - Enterprise.app/Contents/Info.plist" || true; \
	fi

# Create an unsigned .pkg that installs the app into /Applications
package-mac: build-prod-mac
	@echo "Packaging WhoDB EE .app into unsigned .pkg (component)…"
	@pkgbuild --component "build/darwin/universal/WhoDB - Enterprise.app" \
		--install-location /Applications \
		--identifier $(PKG_ID) --version $(VERSION) \
		"build/darwin/universal/whodb-ee-unsigned.pkg"
	@echo "Unsigned package created at build/darwin/universal/whodb-ee-unsigned.pkg"

# Optionally sign the app and package if identities are provided
# Env vars:
#   CODESIGN_ID   = "Developer ID Application: Your Name (TEAMID)" (optional)
#   INSTALLER_ID  = "Developer ID Installer: Your Name (TEAMID)" (required to sign .pkg)
package-mac-signed: build-prod-mac
	@echo "Preparing signed package (requires macOS signing identities)..."
	@if [ -z "$$CODESIGN_ID" ]; then \
		echo "ERROR: CODESIGN_ID is required. Set it as an environment variable or pass on the make command: make package-mac-signed CODESIGN_ID=\"Developer ID Application: Name (TEAMID)\" INSTALLER_ID=\"Developer ID Installer: Name (TEAMID)\""; \
		exit 1; \
	fi
	@if [ -z "$$INSTALLER_ID" ]; then \
		echo "ERROR: INSTALLER_ID is required. Set it as an environment variable or pass on the make command: make package-mac-signed INSTALLER_ID=\"Developer ID Installer: Name (TEAMID)\" CODESIGN_ID=\"Developer ID Application: Name (TEAMID)\""; \
		exit 1; \
	fi
	@echo "Codesigning app with '$$CODESIGN_ID'..."
	@codesign --deep --force --options runtime --timestamp --sign "$$CODESIGN_ID" "build/darwin/universal/WhoDB - Enterprise.app"
	@$(MAKE) package-mac VERSION=$(VERSION) PKG_ID=$(PKG_ID)
	@echo "Signing package with '$$INSTALLER_ID'..."
	@productbuild --sign "$$INSTALLER_ID" --package "build/darwin/universal/whodb-ee-unsigned.pkg" \
		"build/darwin/universal/whodb-ee.pkg"
	@echo "Signed package created at build/darwin/universal/whodb-ee.pkg"

# Notarize signed package (requires either NOTARY_PROFILE or APPLE_ID/TEAM_ID/APPLE_APP_PASSWORD)
notarize-mac: package-mac-signed
	@if [ -n "$(NOTARY_PROFILE)" ]; then \
		echo "Submitting for notarization using profile '$(NOTARY_PROFILE)'..."; \
		xcrun notarytool submit "build/darwin/universal/whodb-ee.pkg" --keychain-profile "$(NOTARY_PROFILE)" --wait; \
	else \
		if [ -n "$(APPLE_ID)" ] && [ -n "$(TEAM_ID)" ] && [ -n "$(APPLE_APP_PASSWORD)" ]; then \
			echo "Submitting for notarization using Apple ID credentials..."; \
			xcrun notarytool submit "build/darwin/universal/whodb-ee.pkg" --apple-id "$(APPLE_ID)" --team-id "$(TEAM_ID)" --password "$(APPLE_APP_PASSWORD)" --wait; \
		else \
			echo "ERROR: Provide NOTARY_PROFILE, or APPLE_ID + TEAM_ID + APPLE_APP_PASSWORD"; \
			exit 1; \
		fi; \
	fi
	@echo "Stapling notarization ticket..."
	@xcrun stapler staple "build/darwin/universal/whodb-ee.pkg" || true
	@xcrun stapler staple "build/darwin/universal/WhoDB - Enterprise.app" || true
	@echo "Notarization complete. Package ready for distribution."

# One-shot target for GitHub Release: build, sign, notarize and staple
release-mac: notarize-mac
	@echo "GitHub release artifact ready: build/darwin/universal/whodb-ee.pkg"

# Create a DMG containing the .app (drag-and-drop install UX)
dmg-mac: build-prod-mac
	@echo "Creating DMG for WhoDB EE..."
	@rm -rf build/dmgroot && mkdir -p build/dmgroot
	@cp -R "build/darwin/universal/WhoDB - Enterprise.app" build/dmgroot/
	@ln -sf /Applications build/dmgroot/Applications
	@hdiutil create -volname "WhoDB - Enterprise" -srcfolder build/dmgroot -ov -format UDZO \
		"build/darwin/universal/whodb-ee.dmg"
	@echo "DMG created at build/darwin/universal/whodb-ee.dmg"

# One-shot DMG release: codesign app (if provided), create DMG, notarize & staple DMG
release-dmg: build-prod-mac
	@if [ -z "$$CODESIGN_ID" ]; then \
		echo "INFO: CODESIGN_ID not set; proceeding without app/DMG codesign"; \
	else \
		echo "Codesigning app with '$$CODESIGN_ID'..."; \
		codesign --deep --force --options runtime --timestamp --sign "$$CODESIGN_ID" "build/darwin/universal/WhoDB - Enterprise.app"; \
	fi
	@$(MAKE) dmg-mac VERSION=$(VERSION)
	@if [ -n "$$CODESIGN_ID" ]; then \
		echo "Codesigning DMG with '$$CODESIGN_ID'..."; \
		codesign --force --timestamp --sign "$$CODESIGN_ID" "build/darwin/universal/whodb-ee.dmg"; \
	fi
	@if [ -n "$(NOTARY_PROFILE)" ]; then \
		echo "Submitting DMG for notarization using profile '$(NOTARY_PROFILE)'..."; \
		xcrun notarytool submit "build/darwin/universal/whodb-ee.dmg" --keychain-profile "$(NOTARY_PROFILE)" --wait; \
	elif [ -n "$(APPLE_ID)" ] && [ -n "$(TEAM_ID)" ] && [ -n "$(APPLE_APP_PASSWORD)" ]; then \
		echo "Submitting DMG for notarization using Apple ID credentials..."; \
		xcrun notarytool submit "build/darwin/universal/whodb-ee.dmg" --apple-id "$(APPLE_ID)" --team-id "$(TEAM_ID)" --password "$(APPLE_APP_PASSWORD)" --wait; \
	else \
		echo "ERROR: Provide NOTARY_PROFILE, or APPLE_ID + TEAM_ID + APPLE_APP_PASSWORD"; \
		exit 1; \
	fi
	@echo "Stapling notarization ticket to DMG..."
	@xcrun stapler staple "build/darwin/universal/whodb-ee.dmg" || true
	@echo "DMG release artifact ready: build/darwin/universal/whodb-ee.dmg"

# Build for Mac App Store (MAS) - requires MAS provisioning & entitlements
# Required vars:
#   MAS_CODESIGN_ID    = Apple Distribution: Name (TEAMID)
#   MAS_INSTALLER_ID   = 3rd Party Mac Developer Installer: Name (TEAMID)
#   MAS_PROFILE        = path to .provisionprofile
#   MAS_ENTITLEMENTS   = path to entitlements.plist (with sandbox)
MAS_CODESIGN_ID ?=
MAS_INSTALLER_ID ?=
MAS_PROFILE ?=
MAS_ENTITLEMENTS ?=

macstore-mac: build-prod-mac
	@if [ -z "$(MAS_CODESIGN_ID)" ] || [ -z "$(MAS_INSTALLER_ID)" ] || [ -z "$(MAS_PROFILE)" ] || [ -z "$(MAS_ENTITLEMENTS)" ]; then \
		echo "ERROR: MAS_CODESIGN_ID, MAS_INSTALLER_ID, MAS_PROFILE, MAS_ENTITLEMENTS are required"; \
		exit 1; \
	fi
	@echo "Embedding provisioning profile..."
	@cp "$(MAS_PROFILE)" "build/darwin/universal/WhoDB - Enterprise.app/Contents/embedded.provisionprofile"
	@echo "Codesigning app for Mac App Store..."
	@codesign --deep --force --options runtime --timestamp --entitlements "$(MAS_ENTITLEMENTS)" --sign "$(MAS_CODESIGN_ID)" "build/darwin/universal/WhoDB - Enterprise.app"
	@echo "Creating signed MAS package..."
	@productbuild --component "build/darwin/universal/WhoDB - Enterprise.app" /Applications --sign "$(MAS_INSTALLER_ID)" \
		"build/darwin/universal/whodb-ee-mas.pkg"
	@echo "MAS package created at build/darwin/universal/whodb-ee-mas.pkg (upload via Transporter/Xcode)."

build-prod-linux: prepare
	@echo "Building WhoDB EE Linux AppImage for AMD64..."
	@mkdir -p build/linux/amd64
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean -platform linux/amd64 \
		-tags "ee webkit2_41" -obfuscated -upx -ldflags="-s -w" -o whodb-ee
	@mv build/bin/whodb-ee build/linux/amd64/

# Backup Linux builds (without webkit2_41 tag - for compatibility)
build-linux-compat: prepare
	@echo "Building WhoDB EE for Linux (native architecture, compatibility mode)..."
	@mkdir -p build/linux/$$(uname -m)-compat
	GOWORK=$$PWD/../go.work.desktop-ee ~/go/bin/wails build -clean \
		-tags ee -ldflags="-s -w" -o whodb-ee-compat
	@mv build/bin/whodb-ee-compat build/linux/$$(uname -m)-compat/

# Clean build artifacts (standalone clean command)
clean:
	@echo "Cleaning all build artifacts..."
	@# On macOS, clear immutable/locked flags and ensure write perms before deleting
	@if [ -d build ]; then \
		(command -v chflags >/dev/null 2>&1 && chflags -R -f nouchg,noschg build) || true; \
		chmod -R u+rwX build || true; \
	fi
	@rm -rf build/windows build/darwin build/linux build/bin build/assets build/index.html wails.lock frontend/dist || true

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
	@echo "  make package-mac     - Create unsigned .pkg for macOS"
	@echo "  make package-mac-signed - Sign app/.pkg using macOS identities"
	@echo "  make notarize-mac    - Notarize & staple signed .pkg (uses NOTARY_PROFILE or APPLE_ID/TEAM_ID/APPLE_APP_PASSWORD)"
	@echo "  make release-mac     - Build, sign, notarize (.pkg) for GitHub release"
	@echo "  make dmg-mac         - Create DMG (drag-and-drop)"
	@echo "  make release-dmg     - Build, (codesign), notarize & staple DMG for GitHub release"
	@echo "  make macstore-mac    - Build package for Mac App Store (requires MAS_* vars)"
	@echo ""
	@echo "Production Builds (with installers):"
	@echo "  make build-prod-windows  - Windows installer"
	@echo "  make build-prod-mac      - macOS app (.app)"
	@echo "  make build-prod-linux    - Linux AppImage"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make doctor         - Check Wails dependencies"
	@echo "  make help           - Show this help message"
