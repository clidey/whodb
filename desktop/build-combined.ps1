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

# Clean ALL old frontend build artifacts (safe remove)
Write-Host "Cleaning frontend build artifacts..." -ForegroundColor Yellow
if (Test-Path "build") {
    Remove-Item -Recurse -Force "build" -ErrorAction SilentlyContinue
}
if (Test-Path ".cache") {
    Remove-Item -Recurse -Force ".cache" -ErrorAction SilentlyContinue
}
if (Test-Path "node_modules/.cache") {
    Remove-Item -Recurse -Force "node_modules/.cache" -ErrorAction SilentlyContinue
}

# Always install dependencies (frozen lockfile ensures reproducibility)
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


# (similar safe Remove-Item changes applied to desktop + Go cleanup sections)
# ...


# Build Go backend FIRST (before Tauri needs it)
Write-Host "Building backend..." -ForegroundColor Yellow
Set-Location "$ProjectRoot\core"

Write-Host "Cleaning Go build cache..." -ForegroundColor Yellow
go clean -cache -testcache

$BinDir = "$ScriptDir\src-tauri\bin"
if (Test-Path $BinDir) {
    Write-Host "Cleaning old backend binaries..." -ForegroundColor Yellow
    Remove-Item -Recurse -Force $BinDir -ErrorAction SilentlyContinue
}
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

# Explicit binary verification (instead of just count)
if (-not (Test-Path $mainBinary) -or -not (Test-Path $aliasBinary)) {
    Write-Host "ERROR: Backend build failed - expected binaries not found!" -ForegroundColor Red
    exit 1
}

Write-Host "Binary created:" -ForegroundColor Cyan
$binaries = Get-ChildItem $BinDir
foreach ($binary in $binaries) {
    Write-Host "  - $($binary.Name) ($([math]::Round($binary.Length / 1MB, 2)) MB)" -ForegroundColor Cyan
}
Write-Host "✓ Backend build verified" -ForegroundColor Green


# Build Tauri app
Write-Host "Building Tauri app..." -ForegroundColor Yellow
Set-Location $ScriptDir

if ($Debug) {
    try {
        $env:RUST_BACKTRACE = "full"
        $env:RUST_LOG = "debug"
        $env:TAURI_LOG = "true"

        Write-Host "Building with debug logging enabled..." -ForegroundColor Cyan
        pnpm run tauri:build -- --target x86_64-pc-windows-msvc --debug
    }
    finally {
        Remove-Item Env:RUST_BACKTRACE -ErrorAction SilentlyContinue
        Remove-Item Env:RUST_LOG -ErrorAction SilentlyContinue
        Remove-Item Env:TAURI_LOG -ErrorAction SilentlyContinue
    }
} else {
    pnpm run tauri:build -- --target x86_64-pc-windows-msvc
}
