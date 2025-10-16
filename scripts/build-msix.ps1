# Copyright 2025 Clidey, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

param(
    [Parameter(Mandatory=$true)]
    [string]$Architecture,

    [Parameter(Mandatory=$true)]
    [string]$Version,

    [Parameter(Mandatory=$false)]
    [string]$PublisherCN = "TempPublisher",

    [Parameter(Mandatory=$false)]
    [string]$CertPath,

    [Parameter(Mandatory=$false)]
    [switch]$SkipSigning
)

$ErrorActionPreference = "Stop"

Write-Host "Building MSIX package for $Architecture..."
if ($SkipSigning -or -not $CertPath) {
    Write-Host "Building unsigned package (Microsoft will sign when uploaded to Partner Center)"
} else {
    Write-Host "Building signed package with provided certificate"
}

# Create package directory structure
$PackageDir = "msix-package-$Architecture"
$AssetsDir = "$PackageDir\Assets"
Remove-Item -Path $PackageDir -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Path $PackageDir -Force | Out-Null
New-Item -ItemType Directory -Path $AssetsDir -Force | Out-Null

# Find the executable from Wails build output
# The workflow now builds raw exe (without NSIS) for MSIX packaging
Write-Host "Current directory: $PWD"
Write-Host "Searching for whodb.exe..."

$PossiblePaths = @(
    "desktop-ce\build\windows\$Architecture\whodb.exe",   # Expected location from workflow
    "desktop-ce\build\bin\whodb.exe",                     # Alternative location
    "desktop-ce\build\whodb.exe",                         # Fallback location
    "build\windows\$Architecture\whodb.exe",              # If run from desktop-ce directory
    "build\bin\whodb.exe",                                # If run from desktop-ce directory
    "windows\$Architecture\whodb.exe"                     # Direct path
)

Write-Host "Checking these locations:"
$PossiblePaths | ForEach-Object { Write-Host "  - $_" }

$ExePath = $null
foreach ($Path in $PossiblePaths) {
    if (Test-Path $Path) {
        $ExePath = $Path
        Write-Host "✅ Found executable at: $ExePath"
        $fileInfo = Get-Item $ExePath
        Write-Host "   Size: $($fileInfo.Length) bytes"
        break
    }
}

if (-not $ExePath) {
    Write-Host "Could not find executable in expected locations."
    Write-Host "Searching recursively for any whodb.exe file..."

    # Search more broadly
    $searchPaths = @("desktop-ce", "build", ".")
    foreach ($searchPath in $searchPaths) {
        if (Test-Path $searchPath) {
            Write-Host "Searching in: $searchPath"
            $FoundExe = Get-ChildItem -Path $searchPath -Filter "whodb*.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
            if ($FoundExe) {
                $ExePath = $FoundExe.FullName
                Write-Host "✅ Found executable at: $ExePath"
                break
            }
        }
    }

    if (-not $ExePath) {
        Write-Error "No executable found! Build may have failed."
        Write-Host ""
        Write-Host "Directory listing:"
        Get-ChildItem -Path . -Recurse -ErrorAction SilentlyContinue | Where-Object { -not $_.PSIsContainer } | Select-Object FullName, Length | Format-Table
        exit 1
    }
}

Copy-Item $ExePath "$PackageDir\whodb.exe"
Write-Host "Copied executable to package directory"

# Copy and resize icon for Store assets
$IconPath = "linux\icon.png"
Copy-Item $IconPath "$AssetsDir\StoreLogo.png"
Copy-Item $IconPath "$AssetsDir\Square44x44Logo.png"
Copy-Item $IconPath "$AssetsDir\Square150x150Logo.png"
Copy-Item $IconPath "$AssetsDir\Wide310x150Logo.png"

# Generate AppxManifest.xml from template
$ManifestTemplate = Get-Content "windows\AppxManifest.xml.template" -Raw
$Manifest = $ManifestTemplate `
    -replace '__VERSION__', $Version `
    -replace '__PUBLISHER_CN__', $PublisherCN `
    -replace '__ARCHITECTURE__', $(if ($Architecture -eq "amd64") { "x64" } else { "arm64" })

$Manifest | Out-File -FilePath "$PackageDir\AppxManifest.xml" -Encoding utf8

# Create MSIX package using makeappx
Write-Host "Creating MSIX package..."
Write-Host "Looking for makeappx.exe..."

# First try to find it in Windows SDK locations
$makeappxPath = $null

# Search for any Windows SDK version
$sdkRoot = "C:\Program Files (x86)\Windows Kits\10\bin"
if (Test-Path $sdkRoot) {
    Write-Host "Searching in Windows SDK directory: $sdkRoot"

    # Get all SDK versions and sort by version number (newest first)
    $sdkVersions = Get-ChildItem -Path $sdkRoot -Directory |
        Where-Object { $_.Name -match "^10\.\d+\.\d+\.\d+$" } |
        Sort-Object { [version]($_.Name -replace "^10\.", "") } -Descending

    Write-Host "Found SDK versions: $($sdkVersions.Name -join ', ')"

    foreach ($version in $sdkVersions) {
        $candidatePath = Join-Path $version.FullName "x64\makeappx.exe"
        if (Test-Path $candidatePath) {
            $makeappxPath = $candidatePath
            Write-Host "✅ Found makeappx at: $makeappxPath"
            break
        }
        $candidatePath = Join-Path $version.FullName "x86\makeappx.exe"
        if (Test-Path $candidatePath) {
            $makeappxPath = $candidatePath
            Write-Host "✅ Found makeappx at: $makeappxPath"
            break
        }
    }
}

# If not found, try common locations
if (-not $makeappxPath) {
    $possiblePaths = @(
        "C:\Program Files (x86)\Windows Kits\10\App Certification Kit\makeappx.exe",
        "C:\Program Files\Windows Kits\10\bin\x64\makeappx.exe",
        "C:\Program Files\Windows Kits\10\bin\x86\makeappx.exe"
    )

    foreach ($path in $possiblePaths) {
        if (Test-Path $path) {
            $makeappxPath = $path
            Write-Host "✅ Found makeappx at: $makeappxPath"
            break
        }
    }
}

# Last resort - check if it's in PATH
if (-not $makeappxPath) {
    $cmdPath = Get-Command makeappx -ErrorAction SilentlyContinue
    if ($cmdPath) {
        $makeappxPath = $cmdPath.Source
        Write-Host "✅ Found makeappx in PATH: $makeappxPath"
    }
}

if (-not $makeappxPath) {
    Write-Error "makeappx.exe not found. Please install Windows SDK."
    Write-Host ""
    Write-Host "Searched locations:"
    Write-Host "  - $sdkRoot\*\x64\makeappx.exe"
    Write-Host "  - $sdkRoot\*\x86\makeappx.exe"
    Write-Host "  - C:\Program Files (x86)\Windows Kits\10\App Certification Kit\makeappx.exe"
    Write-Host ""
    Write-Host "Please install Windows SDK from:"
    Write-Host "https://developer.microsoft.com/windows/downloads/windows-sdk/"
    exit 1
}

Write-Host "Using makeappx from: $makeappxPath"
& $makeappxPath pack /d $PackageDir /p "WhoDB-$Version-$Architecture.msix" /o

if ($LASTEXITCODE -ne 0) {
    Write-Error "makeappx failed with exit code: $LASTEXITCODE"
    exit 1
}

# Sign the package if certificate is provided
if (-not $SkipSigning -and $CertPath) {
    if (-not (Test-Path $CertPath)) {
        Write-Error "Certificate file not found: $CertPath"
        exit 1
    }
    Write-Host "Signing MSIX package..."
    & signtool sign /fd SHA256 /a /f $CertPath /p $env:WINDOWS_PFX_PASSWORD "WhoDB-$Version-$Architecture.msix"
    Write-Host "✓ Signed MSIX package created: WhoDB-$Version-$Architecture.msix"
} else {
    Write-Host "✓ Unsigned MSIX package created: WhoDB-$Version-$Architecture.msix"
    Write-Host "ℹ️ This package will be signed by Microsoft when uploaded to Partner Center"
}
