# Windows build script with optional debug mode
# Usage:
#   Normal build: powershell -ExecutionPolicy Bypass -File build-combined.ps1
#   Debug build:  powershell -ExecutionPolicy Bypass -File build-combined.ps1 -Debug

param(
    [switch]$Debug = $false
)

$ErrorActionPreference = "Stop"

if ($Debug) {
    Write-Host "=== WhoDB Desktop Debug Build for Windows ===" -ForegroundColor Cyan
    Write-Host ""
} else {
    Write-Host "Building WhoDB Desktop for Windows..." -ForegroundColor Green
}

# Get the directory of this script
$ScriptDir = $PSScriptRoot
$ProjectRoot = Split-Path -Parent $ScriptDir

# Build main frontend first (desktop app depends on its CSS)
Write-Host "Building main frontend application..." -ForegroundColor Yellow
Set-Location "$ProjectRoot\frontend"

# Clean ALL old frontend build artifacts
Write-Host "Cleaning frontend build artifacts..." -ForegroundColor Yellow
if (Test-Path "build") {
    Remove-Item -Recurse -Force "build"
}
if (Test-Path ".cache") {
    Remove-Item -Recurse -Force ".cache"
}
if (Test-Path "node_modules/.cache") {
    Remove-Item -Recurse -Force "node_modules/.cache"
}

if (-not (Test-Path "node_modules")) {
    Write-Host "Installing frontend dependencies..." -ForegroundColor Yellow
    pnpm install --prefer-offline
}

# Force clean build
$env:NODE_ENV = "production"
pnpm run build
Remove-Item Env:NODE_ENV -ErrorAction SilentlyContinue

# Verify frontend build succeeded
if (-not (Test-Path "build/index.html")) {
    Write-Host "ERROR: Frontend build failed - build/index.html not found!" -ForegroundColor Red
    exit 1
}
$cssFiles = Get-ChildItem "build/assets/*.css" -ErrorAction SilentlyContinue
if ($cssFiles.Count -eq 0) {
    Write-Host "ERROR: Frontend build failed - no CSS files found in build/assets!" -ForegroundColor Red
    exit 1
}
Write-Host "✓ Frontend build verified - found $($cssFiles.Count) CSS file(s)" -ForegroundColor Green

# Install desktop dependencies
Write-Host "Installing desktop dependencies..." -ForegroundColor Yellow
Set-Location $ScriptDir

# Clean ALL old desktop build artifacts
Write-Host "Cleaning desktop build artifacts..." -ForegroundColor Yellow
if (Test-Path "dist") {
    Remove-Item -Recurse -Force "dist"
}
if (Test-Path ".cache") {
    Remove-Item -Recurse -Force ".cache"
}
if (Test-Path "node_modules/.cache") {
    Remove-Item -Recurse -Force "node_modules/.cache"
}
# Clean Tauri build directories
if (Test-Path "src-tauri\target") {
    Write-Host "Cleaning Tauri target directory (this may take a moment)..." -ForegroundColor Yellow
    Remove-Item -Recurse -Force "src-tauri\target"
}

pnpm install --prefer-offline

# Build desktop frontend with clean cache
Write-Host "Building desktop frontend..." -ForegroundColor Yellow
$env:NODE_ENV = "production"
pnpm run build
Remove-Item Env:NODE_ENV -ErrorAction SilentlyContinue

# Verify desktop build succeeded and CSS was copied
if (-not (Test-Path "dist/index.html")) {
    Write-Host "ERROR: Desktop build failed - dist/index.html not found!" -ForegroundColor Red
    exit 1
}
$desktopCssFiles = Get-ChildItem "dist/assets/*.css" -ErrorAction SilentlyContinue
if ($desktopCssFiles.Count -eq 0) {
    Write-Host "ERROR: Desktop build failed - no CSS files found in dist/assets!" -ForegroundColor Red
    Write-Host "This usually means the frontend CSS wasn't copied properly." -ForegroundColor Red
    exit 1
}
Write-Host "✓ Desktop build verified - found $($desktopCssFiles.Count) CSS file(s)" -ForegroundColor Green

# Create empty build directory for Go embedding - desktop does not need frontend
Write-Host "Creating empty build directory for backend..." -ForegroundColor Yellow
$CoreBuildDir = "$ProjectRoot\core\build"
if (Test-Path $CoreBuildDir) {
    Remove-Item -Recurse -Force $CoreBuildDir
}
New-Item -ItemType Directory -Path $CoreBuildDir | Out-Null
New-Item -ItemType File -Path "$CoreBuildDir\.keep" | Out-Null

# Build Go backend FIRST (before Tauri needs it)
Write-Host "Building backend..." -ForegroundColor Yellow
Set-Location "$ProjectRoot\core"

# Clean Go build cache to ensure fresh build (but keep module cache for speed)
Write-Host "Cleaning Go build cache..." -ForegroundColor Yellow
go clean -cache -testcache

# Clean any existing binaries
$BinDir = "$ScriptDir\src-tauri\bin"
if (Test-Path $BinDir) {
    Write-Host "Cleaning old backend binaries..." -ForegroundColor Yellow
    Remove-Item -Recurse -Force $BinDir
}
New-Item -ItemType Directory -Path $BinDir | Out-Null

# Ensure fresh module downloads
Write-Host "Downloading Go modules..." -ForegroundColor Yellow
go mod download
go mod verify

$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

if ($Debug) {
    Write-Host "Building Go binary with verbose output..." -ForegroundColor Cyan
    go build -v -o (Join-Path $BinDir "whodb-core-x86_64-pc-windows-msvc.exe") .
} else {
    go build -o (Join-Path $BinDir "whodb-core-x86_64-pc-windows-msvc.exe") .
}

# Also copy with the expected name
Copy-Item (Join-Path $BinDir "whodb-core-x86_64-pc-windows-msvc.exe") (Join-Path $BinDir "whodb-core.exe")

Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue

# Verify the binaries exist and are fresh
Write-Host "Binary created:" -ForegroundColor Cyan
$binaries = Get-ChildItem $BinDir
if ($binaries.Count -lt 2) {
    Write-Host "ERROR: Backend build failed - expected 2 binaries but found $($binaries.Count)" -ForegroundColor Red
    exit 1
}
foreach ($binary in $binaries) {
    Write-Host "  - $($binary.Name) ($([math]::Round($binary.Length / 1MB, 2)) MB)" -ForegroundColor Cyan
}
Write-Host "✓ Backend build verified" -ForegroundColor Green

if ($Debug) {
    # Test the backend binary directly
    Write-Host ""
    Write-Host "Testing backend binary directly..." -ForegroundColor Yellow
    Write-Host "Starting backend on port 8081 for 5 seconds..." -ForegroundColor Cyan

    $env:PORT = "8081"
    $env:WHODB_ALLOWED_ORIGINS = "*"

    $backendProcess = Start-Process -FilePath (Join-Path $BinDir "whodb-core.exe") `
        -PassThru `
        -NoNewWindow `
        -RedirectStandardOutput "$env:TEMP\whodb-backend-stdout.log" `
        -RedirectStandardError "$env:TEMP\whodb-backend-stderr.log"

    Start-Sleep -Seconds 5

    if ($backendProcess.HasExited) {
        Write-Host "Backend exited with code: $($backendProcess.ExitCode)" -ForegroundColor Red
        Write-Host ""
        Write-Host "Backend stderr output:" -ForegroundColor Yellow
        Get-Content "$env:TEMP\whodb-backend-stderr.log" -ErrorAction SilentlyContinue | ForEach-Object { Write-Host $_ }
        Write-Host ""
        Write-Host "Backend stdout output:" -ForegroundColor Yellow
        Get-Content "$env:TEMP\whodb-backend-stdout.log" -ErrorAction SilentlyContinue | ForEach-Object { Write-Host $_ }
    } else {
        Write-Host "Backend is running successfully!" -ForegroundColor Green
        $backendProcess | Stop-Process -Force
    }

    # Clean up temp files
    Remove-Item "$env:TEMP\whodb-backend-stdout.log" -ErrorAction SilentlyContinue
    Remove-Item "$env:TEMP\whodb-backend-stderr.log" -ErrorAction SilentlyContinue

    # Clean up environment variables
    Remove-Item Env:PORT -ErrorAction SilentlyContinue
    Remove-Item Env:WHODB_ALLOWED_ORIGINS -ErrorAction SilentlyContinue

    Write-Host ""
}

# Clean up any stale binaries in Tauri target directories
Write-Host "Cleaning up stale binaries in target directories..." -ForegroundColor Yellow
$TargetDirs = @(
    "$ScriptDir\src-tauri\target\debug",
    "$ScriptDir\src-tauri\target\release",
    "$ScriptDir\src-tauri\target\x86_64-pc-windows-msvc\debug",
    "$ScriptDir\src-tauri\target\x86_64-pc-windows-msvc\release"
)

foreach ($dir in $TargetDirs) {
    if (Test-Path $dir) {
        $staleFiles = Get-ChildItem -Path $dir -Filter "whodb-core*.exe" -ErrorAction SilentlyContinue
        foreach ($file in $staleFiles) {
            Write-Host "  Removing stale binary: $($file.FullName)" -ForegroundColor Yellow
            Remove-Item $file.FullName -Force -ErrorAction SilentlyContinue
        }
    }
}

# Build Tauri app
Write-Host "Building Tauri app..." -ForegroundColor Yellow
Set-Location $ScriptDir

if ($Debug) {
    # Set environment variables for debugging
    $env:RUST_BACKTRACE = "full"
    $env:RUST_LOG = "debug"
    $env:TAURI_LOG = "true"

    Write-Host "Building with debug logging enabled..." -ForegroundColor Cyan
    pnpm run tauri:build -- --target x86_64-pc-windows-msvc --debug

    # Clean up env vars
    Remove-Item Env:RUST_BACKTRACE -ErrorAction SilentlyContinue
    Remove-Item Env:RUST_LOG -ErrorAction SilentlyContinue
    Remove-Item Env:TAURI_LOG -ErrorAction SilentlyContinue
} else {
    pnpm run tauri:build -- --target x86_64-pc-windows-msvc
}

Write-Host "Build complete!" -ForegroundColor Green

if ($Debug) {
    Write-Host ""
    Write-Host "Debug build created with console logging enabled." -ForegroundColor Cyan
    Write-Host "Check the console output when running the app for debug messages." -ForegroundColor Cyan
}