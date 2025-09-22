# WhoDB Desktop Application

This desktop application uses Tauri to wrap the WhoDB frontend as a native desktop app with an integrated backend.

## Architecture

- **Frontend**: References the main frontend code directly (no duplication)
- **Backend**: Automatically starts the WhoDB core backend on a random available port
- **Desktop**: Tauri wrapper providing native desktop functionality and backend management

## Development

```bash
# Install dependencies for both desktop and frontend
pnpm install
cd ../frontend && pnpm install && cd ../desktop

# Run in development mode
pnpm run dev:combined
```

## Building

```bash
# Build the complete desktop application
pnpm run build:combined

# Build for Windows (from macOS/Linux with proper cross toolchains)
pnpm run build:win

# Build for Linux (x64 and arm64)
pnpm run build:linux

# Build for all supported targets (macOS, Windows, Linux)
pnpm run build:all
```

This will:
1. Install all dependencies
2. Build the frontend (referencing the main frontend code)
3. Build the backend binary
4. Build the Tauri desktop app with the backend bundled

## Backend Integration

The desktop app automatically:
- Starts the WhoDB core backend on a random available port when the app launches
- Provides the backend port to the frontend via Tauri commands
- Handles backend process cleanup when the app closes
- Falls back gracefully if the backend fails to start

## Key Files

- `src/app.tsx` - Desktop wrapper that imports the frontend App
- `src/config/graphql-client.ts` - GraphQL client that dynamically connects to the backend
- `src-tauri/src/main.rs` - Rust code that manages the backend process
- `vite.config.ts` - Configured to reference frontend code via aliases
- `build-combined.sh` - Build script for the complete application

The desktop app reuses the exact same frontend code without duplication, ensuring consistency between web and desktop versions.