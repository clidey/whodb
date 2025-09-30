# WhoDB Desktop Application - Enterprise Edition

This directory contains the Enterprise Edition desktop application build for WhoDB using [Wails](https://wails.io/). The
desktop app provides a native window experience while maintaining the same functionality as the web version.

## Prerequisites

1. **Install Wails CLI:**
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   ```

2. **Check dependencies:**
   ```bash
   wails doctor
   ```

3. **Platform-specific requirements:**
    - **Windows:** WebView2 (usually pre-installed on Windows 10/11)
    - **macOS:** Xcode Command Line Tools (`xcode-select --install`)
    - **Linux:** `gcc`, `libgtk-3`, `libwebkit2gtk-4.1`

## Quick Build Commands

### Enterprise Edition (EE)

```bash
cd desktop-ee

# Build for current platform
make build

# Build for specific platforms
make build-windows   # Windows AMD64 & ARM64
make build-mac       # macOS Universal Binary
make build-linux     # Linux AMD64 & ARM64

# Build for all platforms
make build-all

# Development mode with hot reload
make dev
```

## Production Builds and Packaging

```bash
# Build production binaries
make build-prod-windows  # Windows NSIS installer
make build-prod-mac      # macOS app (.app)
make build-prod-linux    # Linux AppImage

# Package macOS app into a .pkg (unsigned)
make package-mac

# Package & sign macOS app and .pkg (GitHub Releases)
# You can pass identities on the command or via environment variables.
make package-mac-signed \
  VERSION=1.2.3 \
  CODESIGN_ID="Developer ID Application: Your Name (TEAMID)" \
  INSTALLER_ID="Developer ID Installer: Your Name (TEAMID)"

# Alternatively, set env vars and run
export CODESIGN_ID="Developer ID Application: Your Name (TEAMID)"
export INSTALLER_ID="Developer ID Installer: Your Name (TEAMID)"
make package-mac-signed VERSION=1.2.3

# Notarize (required for Gatekeeper to trust downloads)
make notarize-mac \
  NOTARY_PROFILE="WhoDBNotary" \
  VERSION=1.2.3
# or
make notarize-mac \
  APPLE_ID="appleid@example.com" TEAM_ID="TEAMID" APPLE_APP_PASSWORD="app-specific-password" \
  VERSION=1.2.3

# One-shot GitHub Release artifact (build+sign+notarize)
make release-mac VERSION=1.2.3 CODESIGN_ID="…" INSTALLER_ID="…" NOTARY_PROFILE="WhoDBNotary"
```

## Alternative: Using Shell Scripts

```bash
# Unix/Linux/macOS
./build.sh

# Windows
build-windows.bat
```

## Manual Build (if needed)

```bash
cd desktop-ee
GOWORK=$PWD/../go.work.desktop-ee go mod tidy
GOWORK=$PWD/../go.work.desktop-ee wails build -tags ee -o whodb-ee
```

## Versioning

- You can set a version for macOS builds and packages via `VERSION`. If not set, a timestamp is used.

```bash
# Build the macOS app with a specific version embedded in Info.plist
make build-mac VERSION=1.2.3

# Package to .pkg with a specific version
make package-mac VERSION=1.2.3

## Mac App Store build (MAS)

```bash
# Required:
#   MAS_CODESIGN_ID   = Apple Distribution: Your Name (TEAMID)
#   MAS_INSTALLER_ID  = 3rd Party Mac Developer Installer: Your Name (TEAMID)
#   MAS_PROFILE       = path to .provisionprofile
#   MAS_ENTITLEMENTS  = path to entitlements.plist (with sandbox)

make macstore-mac \
  VERSION=1.2.3 \
  MAS_CODESIGN_ID="Apple Distribution: Your Name (TEAMID)" \
  MAS_INSTALLER_ID="3rd Party Mac Developer Installer: Your Name (TEAMID)" \
  MAS_PROFILE=/path/to/embedded.provisionprofile \
  MAS_ENTITLEMENTS=/path/to/entitlements.plist

# Upload the resulting whodb-ee-mas.pkg via Transporter/Xcode to App Store Connect
```
```

## Output Files

Binaries are generated in `desktop-ee/build/` organized by platform:

- Windows: `build/windows/[arch]/whodb-ee.exe`
- macOS: `build/darwin/universal/WhoDB - Enterprise.app`
- Linux: `build/linux/[arch]/whodb-ee`

## Other Useful Commands

```bash
# Check Wails dependencies
make doctor

# Clean build artifacts
make clean

# Install Wails CLI (one-time setup)
make install-wails

# Show all available commands
make help
```

## Code Signing

### Using Sigstore (Recommended for Open Source)

```bash
# Install cosign
go install github.com/sigstore/cosign/v2/cmd/cosign@latest

# Sign binary
cosign sign-blob build/bin/whodb-ee-windows-amd64.exe \
  --output-signature whodb-ee-windows-amd64.exe.sig \
  --output-certificate whodb-ee-windows-amd64.exe.crt
```

### Platform-Specific Signing

#### Windows (with EV Certificate)

```bash
wails build -windowsconsole=false \
  -windowscertificate="cert.pfx" \
  -windowscertificatepassword="password"
```

#### macOS (with Developer ID)

```bash
wails build -platform darwin/universal \
  -codesign="Developer ID Application: Your Company (TEAMID)" \
  -appid="com.yourcompany.whodb" \
  -notarize
```

## Architecture

The desktop application:

1. Uses the same Go backend as the web version
2. Embeds the React frontend build
3. Serves the app through a native window instead of a browser
4. Routes all API calls through the same Chi router/GraphQL endpoint
5. Maintains full compatibility with all database plugins
6. Includes all Enterprise Edition features and plugins

## Workspace Structure

The build system uses Go workspace `go.work.desktop-ee` which includes:

- `./core` - Core WhoDB functionality
- `./ee` - Enterprise Edition modules and plugins
- `./desktop-ee` - Desktop application module

## Troubleshooting

### Build Issues

1. **Module not found errors:**
   ```bash
   GOWORK=$PWD/../go.work.desktop-ee go mod tidy
   ```

2. **Platform-specific issues:**
   - Run `wails doctor` to check dependencies
   - Ensure you have the correct platform tools installed

### Development Tips

- Use `make dev` for hot reload during development
- The desktop app shares the same codebase as the web version
- No need to modify routing - BrowserRouter works with Wails
- GraphQL API calls work identically to the web version
- Enterprise features are available when building with the `ee` tag

## Clean Build

To remove all build artifacts:

```bash
make clean
```

## Help

For a complete list of available commands:

```bash
make help
```
