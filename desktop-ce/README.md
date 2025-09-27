# WhoDB Desktop Application - Community Edition

This directory contains the Community Edition desktop application build for WhoDB using [Wails](https://wails.io/). The
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

### Community Edition (CE)

```bash
cd desktop-ce

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
cd desktop-ce
GOWORK=$PWD/../go.work.desktop-ce go mod tidy
GOWORK=$PWD/../go.work.desktop-ce wails build -o whodb-ce
```

## Output Files

Binaries are generated in `desktop-ce/build/` organized by platform:

- Windows: `build/windows/[arch]/whodb-ce.exe`
- macOS: `build/darwin/universal/whodb-ce.app`
- Linux: `build/linux/[arch]/whodb-ce`

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
cosign sign-blob build/bin/whodb-ce-windows-amd64.exe \
  --output-signature whodb-ce-windows-amd64.exe.sig \
  --output-certificate whodb-ce-windows-amd64.exe.crt
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

## Workspace Structure

The build system uses Go workspace `go.work.desktop-ce` which includes:

- `./core` - Core WhoDB functionality
- `./ee-stub` - EE stub for CE builds
- `./desktop-ce` - Desktop application module

## Troubleshooting

### Build Issues

1. **Module not found errors:**
   ```bash
   GOWORK=$PWD/../go.work.desktop-ce go mod tidy
   ```

2. **Frontend assets missing:**
   ```bash
   cd ../frontend
   pnpm install
   pnpm run build:ce
   ```

3. **Platform-specific issues:**
    - Run `wails doctor` to check dependencies
    - Ensure you have the correct platform tools installed

### Development Tips

- Use `make dev` for hot reload during development
- The desktop app shares the same codebase as the web version
- No need to modify routing - BrowserRouter works with Wails
- GraphQL API calls work identically to the web version

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