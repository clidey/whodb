# Debug runner for Windows - keeps console window open and shows logs
# Run from PowerShell: .\run-debug-win.ps1

$ErrorActionPreference = "Stop"

Write-Host "=== WhoDB Desktop Debug Runner ===" -ForegroundColor Cyan
Write-Host ""

# Get the directory of this script
$ScriptDir = $PSScriptRoot
$ProjectRoot = Split-Path -Parent $ScriptDir

# Build the app first
Write-Host ">>> Building app in debug mode..." -ForegroundColor Yellow
Set-Location $ScriptDir

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

# Build Go backend with debug output
Write-Host ">>> Building backend with verbose output..." -ForegroundColor Yellow
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

Write-Host "Building Go binary with debug symbols..." -ForegroundColor Cyan
go build -v -o (Join-Path $BinDir "whodb-core-x86_64-pc-windows-msvc.exe") .
Copy-Item (Join-Path $BinDir "whodb-core-x86_64-pc-windows-msvc.exe") (Join-Path $BinDir "whodb-core.exe")

Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue

Write-Host "Binary created:" -ForegroundColor Cyan
Get-ChildItem $BinDir

# Test the backend binary directly
Write-Host ""
Write-Host ">>> Testing backend binary directly..." -ForegroundColor Yellow
Write-Host "Starting backend on port 8081 for 5 seconds..." -ForegroundColor Cyan

$backendProcess = Start-Process -FilePath (Join-Path $BinDir "whodb-core.exe") `
    -ArgumentList "" `
    -EnvironmentVariables @{
        "PORT"="8081"
        "WHODB_ALLOWED_ORIGINS"="*"
    } `
    -PassThru `
    -NoNewWindow

Start-Sleep -Seconds 5

if ($backendProcess.HasExited) {
    Write-Host "Backend exited with code: $($backendProcess.ExitCode)" -ForegroundColor Red
} else {
    Write-Host "Backend is running successfully!" -ForegroundColor Green
    $backendProcess | Stop-Process -Force
}

# Now run the Tauri app in dev mode with console output
Write-Host ""
Write-Host ">>> Starting Tauri app in debug mode..." -ForegroundColor Yellow
Write-Host "This will keep the console window open to show logs" -ForegroundColor Cyan
Write-Host ""

Set-Location $ScriptDir

# Set environment variables for debugging
$env:RUST_BACKTRACE = "full"
$env:RUST_LOG = "debug"
$env:TAURI_LOG = "true"

# Run Tauri in dev mode with visible console
pnpm run tauri:dev

Write-Host ""
Write-Host ">>> Debug session ended" -ForegroundColor Yellow
Write-Host "Press any key to exit..."
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")