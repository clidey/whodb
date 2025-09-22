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
    Write-Host ">>> Building WhoDB Desktop for Windows..." -ForegroundColor Green
}

# Get the directory of this script
$ScriptDir = $PSScriptRoot
$ProjectRoot = Split-Path -Parent $ScriptDir

# Install dependencies
Write-Host ">>> Installing dependencies..." -ForegroundColor Yellow
Set-Location $ScriptDir
pnpm install --prefer-offline

# Build frontend
Write-Host ">>> Building frontend..." -ForegroundColor Yellow
pnpm run build

# Create empty build directory for Go embedding (desktop doesn't need frontend)
Write-Host ">>> Creating empty build directory for backend..." -ForegroundColor Yellow
$CoreBuildDir = "$ProjectRoot\core\build"
if (Test-Path $CoreBuildDir) {
    Remove-Item -Recurse -Force $CoreBuildDir
}
New-Item -ItemType Directory -Path $CoreBuildDir | Out-Null
New-Item -ItemType File -Path "$CoreBuildDir\.keep" | Out-Null

# Build Go backend FIRST (before Tauri needs it)
Write-Host ">>> Building backend..." -ForegroundColor Yellow
Set-Location "$ProjectRoot\core"

$BinDir = "$ScriptDir\src-tauri\bin"
if (-not (Test-Path $BinDir)) {
    New-Item -ItemType Directory -Path $BinDir | Out-Null
}

# Clear old binaries
Get-ChildItem $BinDir -File | Remove-Item -Force

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

# Verify the binary exists
Write-Host "Binary created:" -ForegroundColor Cyan
Get-ChildItem $BinDir

if ($Debug) {
    # Test the backend binary directly
    Write-Host ""
    Write-Host ">>> Testing backend binary directly..." -ForegroundColor Yellow
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

# Build Tauri app
Write-Host ">>> Building Tauri app..." -ForegroundColor Yellow
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

Write-Host ">>> Build complete!" -ForegroundColor Green

if ($Debug) {
    Write-Host ""
    Write-Host "Debug build created with console logging enabled." -ForegroundColor Cyan
    Write-Host "Check the console output when running the app for debug messages." -ForegroundColor Cyan
}