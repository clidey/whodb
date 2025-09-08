# WhoDB Desktop Build Process

This directory contains the build scripts for the WhoDB desktop application.

## Build Scripts

### `prebuild.sh`
Cross-platform prebuild script that builds the Go backend and frontend for all supported platforms and architectures.

**Usage:**
```bash
./prebuild.sh [OPTIONS]
```

**Options:**
- `--platform PLATFORM` - Target platform (darwin, linux, windows)
- `--arch ARCH` - Target architecture (amd64, arm64)
- `--ee` - Build Enterprise Edition
- `--clean` - Clean build directories before building
- `--all` - Build for all platforms and architectures (default)
- `-h, --help` - Show help message

**Examples:**
```bash
./prebuild.sh                    # Build for all platforms
./prebuild.sh --ee               # Build EE for all platforms
./prebuild.sh --platform darwin --arch arm64  # Build for specific platform
```

### `prebuild-current.sh`
Simple prebuild script that builds the Go backend and frontend for the current platform only.

**Usage:**
```bash
./prebuild-current.sh [OPTIONS]
```

**Options:**
- `--ee` - Build Enterprise Edition
- `-h, --help` - Show help message

## NPM Scripts

The following npm scripts are available in `package.json`:

### Prebuild Scripts
- `npm run prebuild` - Run cross-platform prebuild
- `npm run prebuild:current` - Run current platform prebuild
- `npm run prebuild:ee` - Run cross-platform EE prebuild
- `npm run prebuild:current:ee` - Run current platform EE prebuild

### Build Scripts
- `npm run build` - **DEFAULT**: Prebuild for all platforms + electron-builder (macOS, Windows, Linux)
- `npm run build:current` - Prebuild for current platform + electron-builder
- `npm run build:ee` - **DEFAULT**: Prebuild EE for all platforms + electron-builder (macOS, Windows, Linux)
- `npm run build:current:ee` - Prebuild EE for current platform + electron-builder

### Distribution Scripts
- `npm run dist` - **DEFAULT**: Prebuild for all platforms + electron-builder (no publish) - Creates packages for macOS, Windows, and Linux
- `npm run dist:current` - Prebuild for current platform + electron-builder (no publish)
- `npm run dist:ee` - **DEFAULT**: Prebuild EE for all platforms + electron-builder (no publish) - Creates packages for macOS, Windows, and Linux
- `npm run dist:current:ee` - Prebuild EE for current platform + electron-builder (no publish)
- `npm run dist:win` - Prebuild for all platforms + Windows electron-builder
- `npm run dist:mac` - Prebuild for all platforms + macOS electron-builder
- `npm run dist:linux` - Prebuild for all platforms + Linux electron-builder

## Build Output

After running the prebuild scripts, executables for all platforms will be available in `../core/dist/`:

- `whodb-darwin-amd64` - macOS Intel
- `whodb-darwin-arm64` - macOS Apple Silicon
- `whodb-linux-amd64` - Linux Intel (CGO disabled when cross-compiling from macOS)
- `whodb-linux-arm64` - Linux ARM (CGO disabled when cross-compiling from macOS)
- `whodb-windows-amd64.exe` - Windows Intel (CGO disabled when cross-compiling from macOS)
- `whodb` - Current platform executable (when using prebuild-current.sh)

The frontend build will be copied to `../core/build/` and the desktop app will package these resources.

## Cross-Compilation Notes

**CGO Limitations**: When cross-compiling from macOS to Linux or Windows, CGO is automatically disabled due to system call incompatibilities. This means:
- Linux builds from macOS will not have CGO support
- Windows builds from macOS will not have CGO support
- Native builds (macOS from macOS) will have full CGO support

**For CGO-enabled builds**: If you need CGO-enabled Linux or Windows builds, you should build directly on those platforms or use Docker containers with the target platform.

## Workflow

1. **Development**: Use `npm run prebuild:current` for quick builds during development
2. **Testing**: Use `npm run build:current` to test the desktop app on your platform
3. **Release**: Use `npm run build` to create executables and packages for all platforms (macOS, Windows, Linux)
4. **Distribution**: Use `npm run dist` to create distributable packages for all platforms (macOS, Windows, Linux)

**Note**: The default build commands (`npm run build`, `npm run dist`, `npm run build:ee`, `npm run dist:ee`) now build for all platforms by default. Use the `:current` variants only when you need to build for your current platform only.
