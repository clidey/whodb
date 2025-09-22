# Simple Windows build script
# Run from a regular Windows path (not WSL UNC path)

$ErrorActionPreference = "Stop"

Write-Host ">>> Building WhoDB Desktop for Windows..." -ForegroundColor Green

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

go build -o (Join-Path $BinDir "whodb-core-x86_64-pc-windows-gnu.exe") .
# Also copy with the expected name
Copy-Item (Join-Path $BinDir "whodb-core-x86_64-pc-windows-gnu.exe") (Join-Path $BinDir "whodb-core.exe")

Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue

# Verify the binary exists
Write-Host "Binary created:" -ForegroundColor Cyan
Get-ChildItem $BinDir

# Build Tauri app
Write-Host ">>> Building Tauri app..." -ForegroundColor Yellow
Set-Location $ScriptDir
pnpm run tauri:build -- --target x86_64-pc-windows-gnu

Write-Host ">>> Build complete!" -ForegroundColor Green