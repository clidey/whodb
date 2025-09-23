# Windows build script with optional debug mode
# Usage:
#   Normal build: powershell -ExecutionPolicy Bypass -File build-combined.ps1
#   Debug build:  powershell -ExecutionPolicy Bypass -File build-combined.ps1 -Debug

param(
    [switch]$Debug = $false
)

$ErrorActionPreference = "Stop"

function Clear-DirSafe {
    param([string]$Path)
    if (Test-Path $Path) {
        try {
            Remove-Item -Recurse -Force $Path -ErrorAction SilentlyContinue
            Write-Host "  Cleared: $Path" -ForegroundColor DarkGray
        }
        catch {
            Write-Host "  Warning: Could not fully remove $Path ($_)" -ForegroundColor Yellow
        }
    }
}

if ($Debug) {
    Write-Host "=== WhoDB Desktop Debug Build for Windows ===" -ForegroundColor Cyan
    Write-Host ""
} else {
    Write-Host "Building WhoDB Desktop for Windows..." -ForegroundColor Green
}

# Get the directory of this script
$ScriptDir = $PSScriptRoot
$ProjectRoot = Split-Path -Parent $ScriptDir

# Build main frontend first
Write-Host "Building main frontend application..." -ForegroundColor Yellow
Set-Location "$ProjectRoot\frontend"

# Clean frontend build artifacts
Write-Host "Cleaning frontend build artifacts..." -ForegroundColor Yellow
Clear-DirSafe "build"
Clear-DirSafe ".cache"
Clear-DirSafe "node_modules/.cache"

# Always install dependencies
Write-Host "Installing frontend dependencies..." -ForegroundColor Yellow
pnpm install --prefer-offline --frozen-lockfile

# Force clean build
try {
    $env:NODE_ENV = "production"
    pnpm run build
}
finally {
    Remove-Item Env:NODE_ENV -ErrorAction SilentlyContinue
}

# Verify frontend build
if (-not (Test-Path "build/index.html")) {
    Write-Host "ERROR: Frontend build failed - build/index.html not found!" -ForegroundColor Red
    exit 1
}
$cssFiles = Get-ChildItem "build/assets/*.css" -ErrorAction SilentlyContinue
if ($cssFiles.Count -eq 0) {
    Write-Host "ERROR: Frontend build failed - no CSS files found in build/assets!" -ForegroundColor Red
    exit 1
}
Write-Host "Frontend build verified - found $($cssFiles.Count) CSS file(s)" -ForegroundColor Green

# Install desktop dependencies
Write-Host "Installing desktop dependencies..." -ForegroundColor Yellow
Set-Location $ScriptDir

# Clean desktop build artifacts
Write-Host "Cleaning desktop build artifacts..." -ForegroundColor Yellow
Clear-DirSafe "dist"
Clear-DirSafe ".cache"
Clear-DirSafe "node_modules/.cache"
Clear-DirSafe "src-tauri\target"

pnpm install --prefer-offline --frozen-lockfile

# Build desktop frontend
try {
    $env:NODE_ENV = "production"
    pnpm run build
}
finally {
    Remove-Item Env:NODE_ENV -ErrorAction SilentlyContinue
}

# Verify desktop build
if (-not (Test-Path "dist/index.html")) {
    Write-Host "ERROR: Desktop build failed - dist/index.html not found!" -ForegroundColor Red
    exit 1
}
$desktopCssFiles = Get-ChildItem "dist/assets/*.css" -ErrorAction SilentlyContinue
if ($desktopCssFiles.Count -eq 0) {
    Write-Host "ERROR: Desktop build failed - no CSS files found in dist/assets!" -ForegroundColor Red
    exit 1
}
Write-Host "Desktop build verified - found $($desktopCssFiles.Count) CSS file(s)" -ForegroundColor Green

# Create empty build directory for Go embedding
Write-Host "Creating empty build directory for backend..." -ForegroundColor Yellow
$CoreBuildDir = "$ProjectRoot\core\build"
Clear-DirSafe $CoreBuildDir
New-Item -ItemType Directory -Path $CoreBuildDir | Out-Null
New-Item -ItemType File -Path "$CoreBuildDir\.keep" | Out-Null

# Build Go backend
Write-Host "Building backend..." -ForegroundColor Yellow
Set-Location "$ProjectRoot\core"

Write-Host "Cleaning Go build cache..." -ForegroundColor Yellow
go clean -cache -testcache

$BinDir = "$ScriptDir\src-tauri\bin"
Clear-DirSafe $BinDir
New-Item -ItemType Directory -Path $BinDir | Out-Null

Write-Host "Downloading Go modules..." -ForegroundColor Yellow
go mod download
go mod verify

try {
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"

    if ($Debug) {
        Write-Host "Building Go binary with verbose output..." -ForegroundColor Cyan
        go build -v -o (Join-Path $BinDir "whodb-core-x86_64-pc-windows-msvc.exe") .
    } else {
        go build -o (Join-Path $BinDir "whodb-core-x86_64-pc-windows-msvc.exe") .
    }
}
finally {
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
    Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
}

# Copy to expected name
$mainBinary = Join-Path $BinDir "whodb-core-x86_64-pc-windows-msvc.exe"
$aliasBinary = Join-Path $BinDir "whodb-core.exe"
Copy-Item $mainBinary $aliasBinary -Force

# Verify binaries
if (-not (Test-Path $mainBinary) -or -not (Test-Path $aliasBinary)) {
    Write-Host "ERROR: Backend build failed - expected binaries not found!" -ForegroundColor Red
    exit 1
}

Write-Host "Binary created:" -ForegroundColor Cyan
$binaries = Get-ChildItem $BinDir
foreach ($binary in $binaries) {
    $sizeMB = [math]::Round($binary.Length / 1MB, 2)
    Write-Host "  - $($binary.Name) ($sizeMB MB)" -ForegroundColor Cyan
}
Write-Host "Backend build verified" -ForegroundColor Green

# Test the backend binary if in debug mode
if ($Debug) {
    Write-Host ""
    Write-Host "Testing backend binary directly..." -ForegroundColor Yellow
    Write-Host "Starting backend on port 8081 for 5 seconds..." -ForegroundColor Cyan

    $env:PORT = "8081"
    $env:WHODB_ALLOWED_ORIGINS = "*"

    $backendProcess = Start-Process -FilePath $aliasBinary `
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

# Clean up stale binaries in target directories
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
    Write-Host 'Check the console output when running the app for debug messages.' -ForegroundColor Cyan
}
