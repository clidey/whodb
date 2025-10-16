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
$PossiblePaths = @(
    "desktop-ce\build\windows\$Architecture\whodb.exe",   # Raw exe from workflow build
    "desktop-ce\build\bin\whodb.exe",                     # Alternative location
    "desktop-ce\build\whodb.exe"                          # Fallback location
)

$ExePath = $null
foreach ($Path in $PossiblePaths) {
    if (Test-Path $Path) {
        $ExePath = $Path
        Write-Host "Found executable at: $ExePath"
        break
    }
}

if (-not $ExePath) {
    Write-Host "Could not find executable in expected locations."
    Write-Host "Searching for any .exe file in desktop-ce\build..."

    $FoundExe = Get-ChildItem -Path "desktop-ce\build" -Filter "*.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($FoundExe) {
        $ExePath = $FoundExe.FullName
        Write-Host "Found executable at: $ExePath"
    } else {
        Write-Error "No executable found in desktop-ce\build directory"
        Write-Host "Expected one of:"
        $PossiblePaths | ForEach-Object { Write-Host "  - $_" }
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

# Check if makeappx is available
$makeappxPath = Get-Command makeappx -ErrorAction SilentlyContinue
if (-not $makeappxPath) {
    # Try to find it in Windows SDK locations
    $possiblePaths = @(
        "C:\Program Files (x86)\Windows Kits\10\bin\10.0.22621.0\x64\makeappx.exe",
        "C:\Program Files (x86)\Windows Kits\10\bin\10.0.22000.0\x64\makeappx.exe",
        "C:\Program Files (x86)\Windows Kits\10\bin\10.0.19041.0\x64\makeappx.exe",
        "C:\Program Files (x86)\Windows Kits\10\bin\x64\makeappx.exe"
    )

    foreach ($path in $possiblePaths) {
        if (Test-Path $path) {
            $makeappxPath = $path
            Write-Host "Found makeappx at: $makeappxPath"
            break
        }
    }

    if (-not $makeappxPath) {
        Write-Error "makeappx.exe not found. Please install Windows SDK."
        exit 1
    }

    & $makeappxPath pack /d $PackageDir /p "WhoDB-$Version-$Architecture.msix" /o
} else {
    & makeappx pack /d $PackageDir /p "WhoDB-$Version-$Architecture.msix" /o
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
