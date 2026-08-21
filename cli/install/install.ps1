# WhoDB CLI Native Installer for Windows
#
# Usage:
#   irm https://raw.githubusercontent.com/clidey/whodb/main/cli/install/install.ps1 | iex
#
# Or with specific version:
#   $env:WHODB_VERSION = "v0.62.0"; irm https://raw.githubusercontent.com/clidey/whodb/main/cli/install/install.ps1 | iex
#
# Copyright 2025 Clidey, Inc.
# Licensed under the Apache License, Version 2.0

$ErrorActionPreference = "Stop"

# Configuration
$Repo = "clidey/whodb"
# Release assets keep the whodb-cli- prefix; the installed command is "whodb".
$AssetName = "whodb-cli"
$InstallName = "whodb"
$InstallDir = if ($env:WHODB_INSTALL_DIR) { $env:WHODB_INSTALL_DIR } else { "$env:LOCALAPPDATA\WhoDB\bin" }

function Write-Step {
    param([string]$Message)
    Write-Host "==> " -ForegroundColor Blue -NoNewline
    Write-Host $Message
}

function Write-Success {
    param([string]$Message)
    Write-Host "==> " -ForegroundColor Green -NoNewline
    Write-Host $Message
}

function Write-Warning {
    param([string]$Message)
    Write-Host "Warning: " -ForegroundColor Yellow -NoNewline
    Write-Host $Message
}

function Write-Error {
    param([string]$Message)
    Write-Host "Error: " -ForegroundColor Red -NoNewline
    Write-Host $Message
}

function Get-Architecture {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        "X64" { return "amd64" }
        "Arm64" { return "arm64" }
        default {
            Write-Error "Unsupported architecture: $arch"
            exit 1
        }
    }
}

function Get-LatestVersion {
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing
        return $response.tag_name
    }
    catch {
        Write-Error "Failed to fetch latest version: $_"
        exit 1
    }
}

function Add-ToPath {
    param([string]$PathToAdd)

    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -notlike "*$PathToAdd*") {
        $newPath = "$PathToAdd;$currentPath"
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        $env:Path = "$PathToAdd;$env:Path"
        return $true
    }
    return $false
}

function Main {
    Write-Step "WhoDB CLI Installer for Windows"
    Write-Host ""

    # Detect architecture
    $arch = Get-Architecture
    Write-Step "Detected architecture: windows/$arch"

    # Get version
    $version = if ($env:WHODB_VERSION) { $env:WHODB_VERSION } else { $null }
    if (-not $version -or $version -eq "latest") {
        Write-Step "Fetching latest version..."
        $version = Get-LatestVersion
    }
    Write-Step "Installing version: $version"

    # Construct download URL
    $binaryFile = "$AssetName-windows-$arch.exe"
    $downloadUrl = "https://github.com/$Repo/releases/download/$version/$binaryFile"

    # Create install directory
    if (-not (Test-Path $InstallDir)) {
        Write-Step "Creating install directory: $InstallDir"
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    $installPath = Join-Path $InstallDir "$InstallName.exe"

    # Download binary
    Write-Step "Downloading $binaryFile..."
    try {
        $ProgressPreference = 'SilentlyContinue'  # Speed up download
        Invoke-WebRequest -Uri $downloadUrl -OutFile $installPath -UseBasicParsing
    }
    catch {
        Write-Error "Download failed: $_"
        Write-Error "URL: $downloadUrl"
        exit 1
    }

    # Verify download
    if (-not (Test-Path $installPath) -or (Get-Item $installPath).Length -eq 0) {
        Write-Error "Download failed or file is empty"
        exit 1
    }

    Write-Success "WhoDB CLI $version installed successfully!"
    Write-Host ""

    # Keep a deprecated "whodb-cli.exe" copy pointing at the new binary so
    # existing scripts keep working. It prints a warning and will be removed
    # in a future release.
    $legacyPath = Join-Path $InstallDir "whodb-cli.exe"
    Copy-Item -Path $installPath -Destination $legacyPath -Force

    # Add to PATH
    $pathAdded = Add-ToPath -PathToAdd $InstallDir
    if ($pathAdded) {
        Write-Success "Added $InstallDir to your PATH"
        Write-Host ""
        Write-Warning "Please restart your terminal for PATH changes to take effect"
        Write-Host ""
    }
    else {
        Write-Step "$InstallDir is already in your PATH"
    }

    # Show usage
    Write-Host "Get started:"
    Write-Host "  $InstallName          # Launch interactive TUI"
    Write-Host "  $InstallName mcp      # Run as MCP server"
    Write-Host "  $InstallName --help   # Show help"
    Write-Host ""
    Write-Host "Documentation: https://docs.whodb.com/cli"
}

# Run main function
Main
