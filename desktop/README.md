# WhoDB Desktop Application

This directory contains the Electron wrapper for WhoDB, enabling it to run as a native desktop application on Windows, macOS, and Linux.

## Architecture

The desktop application consists of:
- **Electron Main Process** (`main.js`): Manages the application lifecycle and spawns the Go backend
- **Preload Script** (`preload.js`): Provides secure context bridge between Electron and the web app
- **Go Backend**: The existing WhoDB backend that serves the React frontend

## Development

### Prerequisites
- Node.js and npm installed
- Go installed (for building the backend)
- All frontend and backend dependencies installed

### Running in Development Mode

From the project root:

```bash
# Install desktop dependencies
npm run desktop:install

# Run desktop app in development mode
npm run desktop:dev
```

This will:
1. Start the Go backend on port 8080
2. Launch Electron pointing to http://localhost:8080

### Building for Production

```bash
# Build for current platform
npm run desktop:build

# Build and package for distribution
npm run desktop:dist

# Platform-specific builds
npm run desktop:dist:win     # Windows
npm run desktop:dist:mac     # macOS
npm run desktop:dist:linux   # Linux
```

## Distribution

Built applications will be available in `desktop/dist/`:
- **Windows**: `WhoDB Setup {version}.exe` (installer)
- **macOS**: `WhoDB-{version}.dmg` (disk image)
- **Linux**: `WhoDB-{version}.AppImage` (portable executable)

## Configuration

### Electron Builder Configuration

The `electron-builder` configuration in `package.json` defines:
- Application metadata (name, ID, version)
- Platform-specific settings
- File inclusion patterns
- Icon locations

### Icons

Place platform-specific icons in the `desktop/` directory:
- `icon.ico` - Windows icon (256x256)
- `icon.icns` - macOS icon (512x512)
- `icon.png` - Linux icon (512x512)

## Security

The desktop app follows Electron security best practices:
- Context isolation enabled
- Node integration disabled
- Preload script with limited API exposure
- No remote content loading

## Troubleshooting

### Backend fails to start
- Ensure the Go backend is built: `npm run desktop:build-backend`
- Check that the backend binary has execute permissions
- Verify no other process is using port 8080

### Application shows blank screen
- The backend may still be starting up - wait a few seconds
- Check the developer console for errors (View → Toggle Developer Tools)
- Ensure the backend is running correctly by visiting http://localhost:8080 in a browser

### Build failures
- Clear the `dist/` directory and try again
- Ensure all dependencies are installed: `npm run desktop:install`
- Check that you have the necessary build tools for your platform