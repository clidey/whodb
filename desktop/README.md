# WhoDB Desktop Application (Tauri)

A lightweight native desktop wrapper for WhoDB using [Tauri](https://tauri.app/), providing a secure and performant desktop experience across Windows, macOS, and Linux.

## Features

- **Lightweight**: Significantly smaller than Electron alternatives (~10MB vs ~150MB)
- **Native Performance**: Uses system WebView for rendering
- **Cross-Platform**: Single codebase for Windows, macOS, and Linux
- **Secure**: Rust-based backend with minimal attack surface
- **Auto-Updates**: Built-in update mechanism (optional)
- **Code Signing**: Scripts included for all platforms

## Architecture

The desktop application consists of:

1. **Tauri Core**: Rust-based application wrapper
2. **Go Backend**: The WhoDB server running as a sidecar process
3. **WebView**: System WebView displaying the WhoDB frontend
4. **IPC Bridge**: Secure communication between Tauri and the Go backend

## Prerequisites

### Development
- [Rust](https://www.rust-lang.org/tools/install) (latest stable)
- [Node.js](https://nodejs.org/) (v18+)
- [Go](https://golang.org/) (v1.21+)
- [pnpm](https://pnpm.io/) (for frontend builds)

### Platform-Specific
- **Windows**: WebView2 (comes with Windows 10/11)
- **macOS**: Xcode Command Line Tools
- **Linux**: webkit2gtk, libayatana-appindicator3-dev

## Quick Start

### Install Dependencies

```bash
# Install Tauri dependencies
cd desktop
npm install

# Linux-specific dependencies
# Ubuntu/Debian:
sudo apt update
sudo apt install libwebkit2gtk-4.0-dev \
    build-essential \
    curl \
    wget \
    libssl-dev \
    libgtk-3-dev \
    libayatana-appindicator3-dev \
    librsvg2-dev

# Fedora:
sudo dnf install webkit2gtk3-devel \
    openssl-devel \
    gtk3-devel \
    libappindicator-gtk3-devel

# macOS (ensure Xcode CLT is installed):
xcode-select --install
```

### Development

```bash
# Build the Go backend first
npm run prebuild:current

# Run in development mode
npm run dev
```

This will:
1. Build the Go backend for your current platform
2. Start the Tauri development server
3. Open the WhoDB desktop app with hot-reload enabled

### Building for Production

#### Current Platform Only
```bash
# Build for your current OS/architecture
npm run build:current
```

#### All Platforms (Cross-Compilation)
```bash
# Build for all supported platforms
npm run build:all
```

This creates distributable packages in `src-tauri/target/release/bundle/`:
- **Windows**: `.msi` and `.exe` installers
- **macOS**: `.dmg` and `.app` bundles  
- **Linux**: `.deb`, `.AppImage`, and `.rpm` packages

## Platform-Specific Builds

### Windows
- Produces MSI installer and portable executable
- Requires WebView2 (auto-installed if missing)
- Supports Windows 10/11 x64

### macOS
- Produces DMG installer and .app bundle
- Supports macOS 10.13+ (Intel and Apple Silicon)
- Requires code signing for distribution

### Linux
- Produces AppImage (portable), .deb, and .rpm packages
- Supports most modern distributions
- AppImage works without installation

## Code Signing

Prevent security warnings by signing your builds:

### Windows
```powershell
.\sign-windows.ps1 -ExePath "path\to\whodb.exe" `
  -CertPath "path\to\certificate.pfx" `
  -CertPassword (Read-Host -AsSecureString)
```

### macOS
```bash
./sign-macos.sh -a WhoDB.app \
  -i "Developer ID Application: Your Name (TEAMID)" \
  -t TEAMID -u your@apple.id -p app-specific-password
```

### Linux
```bash
./sign-linux.sh -f WhoDB.AppImage -k your@email.com -d
```

See [SIGNING.md](SIGNING.md) for detailed instructions.

## Configuration

### Tauri Configuration
Edit `src-tauri/tauri.conf.json` to customize:
- Application metadata (name, version, description)
- Window properties (size, title, resizable)
- Security policies
- Build targets

### Backend Port
The Go backend runs on port 8080 by default. To change:
1. Set the `WHODB_PORT` environment variable
2. Update `get_backend_url()` in `src-tauri/src/main.rs`

## Project Structure

```
desktop/
├── src/                    # Frontend wrapper
│   ├── index.html         # Loading screen
│   └── main.js           # Frontend initialization
├── src-tauri/            # Tauri application
│   ├── src/
│   │   └── main.rs      # Rust backend
│   ├── Cargo.toml       # Rust dependencies
│   ├── tauri.conf.json  # Tauri configuration
│   └── icons/           # Application icons
├── prebuild.sh          # Multi-platform build script
├── prebuild-current.sh  # Current platform build
├── sign-*.sh/ps1       # Platform signing scripts
└── package.json        # Node.js dependencies
```

## Development Tips

### Hot Reload
Frontend changes are automatically reloaded during `npm run dev`.

### Debugging
- **Frontend**: Use browser DevTools (Right-click → Inspect)
- **Rust**: Set `RUST_LOG=debug` for verbose logging
- **Go Backend**: Check terminal output for server logs

### Building Specific Targets
```bash
# Windows only
npm run tauri build -- --target x86_64-pc-windows-msvc

# macOS Intel only
npm run tauri build -- --target x86_64-apple-darwin

# macOS Apple Silicon only  
npm run tauri build -- --target aarch64-apple-darwin

# Linux x64 only
npm run tauri build -- --target x86_64-unknown-linux-gnu
```

## Troubleshooting

### Common Issues

1. **"Go backend failed to start"**
   - Ensure Go is installed and in PATH
   - Check firewall isn't blocking port 8080
   - Verify prebuild script completed successfully

2. **"WebView2 not found" (Windows)**
   - Install from: https://go.microsoft.com/fwlink/p/?LinkId=2124703

3. **Build fails on Linux**
   - Install all required system dependencies (see Prerequisites)
   - Ensure you have a desktop environment with GTK support

4. **Code signing errors**
   - Verify certificates are valid and not expired
   - Check you have the correct permissions/passwords
   - See platform-specific requirements in SIGNING.md

### Debug Mode
Set environment variables for verbose output:
```bash
RUST_LOG=debug npm run dev
```

## Performance Optimization

Tauri provides significant advantages over Electron:
- **Bundle Size**: ~10MB vs ~150MB
- **Memory Usage**: ~50MB vs ~300MB
- **Startup Time**: <1s vs 3-5s
- **CPU Usage**: Minimal overhead using system WebView

## Security

Tauri provides enhanced security through:
- Rust's memory safety
- Minimal API surface
- Content Security Policy (CSP)
- IPC permission system

The app uses secure defaults and isolates the web content from system APIs.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test on all target platforms
5. Submit a pull request

See [CONTRIBUTING.md](../CONTRIBUTING.md) for details.

## License

This project is licensed under the MIT License. See [LICENSE](../LICENSE) for details.