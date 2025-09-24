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

## Production Builds (with installers)

```bash
make build-prod-windows  # Windows NSIS installer
make build-prod-mac      # macOS .pkg installer
make build-prod-linux    # Linux AppImage
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

## Output Files

Binaries are generated in `desktop-ee/build/` organized by platform:

- Windows: `build/windows/[arch]/whodb-ee.exe`
- macOS: `build/darwin/universal/whodb-ee.app`
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

2. **Frontend assets missing:**
   ```bash
   cd ../frontend
   pnpm install
   pnpm run build:ee
   ```

3. **Platform-specific issues:**
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