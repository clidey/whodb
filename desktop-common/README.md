# WhoDB Desktop Application

This is the WhoDB desktop application built with [Wails](https://wails.io/), providing a native window experience while maintaining full compatibility with the web version. Available in both Community Edition (CE) and Enterprise Edition (EE).

## Prerequisites

1. **Install Wails CLI:**
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   ```

2. **Check dependencies:**
   ```bash
   wails doctor
   ```

3. **Platform requirements:**
   - **Windows:** WebView2 (pre-installed on Windows 10/11)
   - **macOS:** Xcode Command Line Tools (`xcode-select --install`)
   - **Linux:** `gcc`, `libgtk-3`, `libwebkit2gtk-4.1`

## Quick Start

### Running the Desktop Application

Navigate to the edition directory and use make commands:

```bash
# Community Edition
cd desktop-ce
make dev       # Development mode with hot reload
make build     # Build for current platform
make help      # See all available commands

# Enterprise Edition
cd desktop-ee
make dev       # Development mode with EE features
make build     # Build for current platform
make help      # See all available commands
```

## Build Commands

### Development & Basic Builds

```bash
make dev              # Run in development mode with hot reload
make build            # Build for current platform
make build-all        # Build for all platforms
make build-windows    # Windows AMD64 & ARM64
make build-mac        # macOS Universal Binary
make build-mac-arm64  # macOS ARM64 only (Apple Silicon)
make build-mac-amd64  # macOS AMD64 only (Intel)
make build-linux      # Linux AMD64 & ARM64
```

### Production Builds

```bash
make build-prod-windows     # Windows NSIS installer with UPX
make build-prod-mac         # macOS Universal Binary
make build-prod-mac-arm64   # macOS ARM64 only
make build-prod-mac-amd64   # macOS AMD64 only
make build-prod-linux       # Linux binary with UPX compression
```

### macOS Packaging & Signing

```bash
# Create unsigned .pkg
make package-mac

# Sign app and .pkg
make package-mac-signed \
  CODESIGN_ID="Developer ID Application: Your Name (TEAMID)" \
  INSTALLER_ID="Developer ID Installer: Your Name (TEAMID)"

# Notarize for distribution
make notarize-mac \
  NOTARY_PROFILE="WhoDBNotary"
# or
make notarize-mac \
  APPLE_ID="appleid@example.com" \
  TEAM_ID="TEAMID" \
  APPLE_APP_PASSWORD="app-specific-password"

# One-shot GitHub Release (build + sign + notarize)
make release-mac \
  CODESIGN_ID="..." \
  INSTALLER_ID="..." \
  NOTARY_PROFILE="WhoDBNotary"

# Create DMG for drag-and-drop install
make dmg-mac
make release-dmg  # DMG with signing and notarization
```

### Mac App Store Build

```bash
make macstore-mac \
  MAS_CODESIGN_ID="Apple Distribution: Your Name (TEAMID)" \
  MAS_INSTALLER_ID="3rd Party Mac Developer Installer: Your Name (TEAMID)" \
  MAS_PROFILE=/path/to/embedded.provisionprofile \
  MAS_ENTITLEMENTS=/path/to/entitlements.plist
```

## Output Files

Binaries are generated in `build/` organized by platform:

### Community Edition (desktop-ce)
- Windows: `build/windows/[arch]/whodb-ce.exe`
- macOS Universal: `build/darwin/universal/WhoDB.app`
- macOS ARM64: `build/darwin/arm64/WhoDB.app`
- macOS AMD64: `build/darwin/amd64/WhoDB.app`
- Linux: `build/linux/[arch]/whodb-ce`

### Enterprise Edition (desktop-ee)
- Windows: `build/windows/[arch]/whodb-ee.exe`
- macOS Universal: `build/darwin/universal/WhoDB - Enterprise.app`
- macOS ARM64: `build/darwin/arm64/WhoDB - Enterprise.app`
- macOS AMD64: `build/darwin/amd64/WhoDB - Enterprise.app`
- Linux: `build/linux/[arch]/whodb-ee`

**Note:** Architecture-specific macOS builds (`arm64`/`amd64`) are optimized single-architecture builds that will be smaller than the universal binary (~50% size reduction) since they only contain code for one architecture.

## Manual Build (Advanced)

If you need to build manually without Make:

```bash
# Community Edition
cd desktop-ce
GOWORK=$PWD/../go.work.desktop-ce wails build -o whodb-ce

# Enterprise Edition
cd desktop-ee
GOWORK=$PWD/../go.work.desktop-ee wails build -tags ee -o whodb-ee
```

## Architecture

The desktop application:

1. Uses the same Go backend as the web version
2. Embeds the React frontend build
3. Serves through a native window instead of browser
4. Routes all API calls through the same Chi router/GraphQL endpoint
5. Maintains full compatibility with all database plugins
6. EE includes all enterprise features and additional plugins

### Workspace Structure

- **CE workspace** (`go.work.desktop-ce`):
  - `./core` - Core WhoDB functionality
  - `./ee-stub` - EE stub for CE builds
  - `./desktop-ce` - CE desktop module
  - `./desktop-common` - Shared desktop code

- **EE workspace** (`go.work.desktop-ee`):
  - `./core` - Core WhoDB functionality
  - `./ee` - Enterprise Edition modules
  - `./desktop-ee` - EE desktop module
  - `./desktop-common` - Shared desktop code

## Development Tips

- Use `make dev` for hot reload during development
- The desktop app shares the same codebase as the web version
- BrowserRouter works seamlessly with Wails
- GraphQL API calls work identically to the web version
- All database plugins work without modification

## Troubleshooting

### Module Not Found Errors

```bash
# For CE
cd desktop-ce
GOWORK=$PWD/../go.work.desktop-ce go mod tidy

# For EE
cd desktop-ee
GOWORK=$PWD/../go.work.desktop-ee go mod tidy
```

### Platform-Specific Issues

Run `wails doctor` to check dependencies and ensure platform tools are installed.

## Useful Commands

```bash
make clean          # Remove all build artifacts
make doctor         # Check Wails dependencies
make install-wails  # Install/update Wails CLI
make help           # Show all available commands
```

## Versioning

Set version for builds and packages:

```bash
make build-mac VERSION=1.2.3
make package-mac VERSION=1.2.3
```

If VERSION is not specified, a timestamp is used.

## Code Signing Options

### Using Sigstore (Recommended for Open Source)

```bash
# Install cosign
go install github.com/sigstore/cosign/v2/cmd/cosign@latest

# Sign binary
cosign sign-blob build/bin/whodb-ce-windows-amd64.exe \
  --output-signature whodb-ce-windows-amd64.exe.sig \
  --output-certificate whodb-ce-windows-amd64.exe.crt
```

### Platform-Specific Signing

See the macOS packaging section above for Apple Developer ID signing, or use Windows EV certificates with Wails build flags.

## Notes

- The Makefile system automatically handles frontend building and asset preparation
- Both editions share common code via the `desktop-common` module
- The `-obfuscated` flag is currently disabled due to hanging issues
- Linux builds use `webkit2_41` tag for better compatibility