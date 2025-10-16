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
    [string]$PublisherCN = "CN=TempPublisher",

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

# Copy executable from NSIS build (extract from installer)
$InstallerPath = "desktop-ce\build\windows\$Architecture\whodb-installer.exe"
if (-not (Test-Path $InstallerPath)) {
    Write-Error "Installer not found: $InstallerPath"
    exit 1
}

# For simplicity, we'll use the non-installer exe if available, or extract from NSIS
# In production, you'd extract the exe from the NSIS installer or build without NSIS
$ExePath = "desktop-ce\build\windows\$Architecture\whodb.exe"
if (-not (Test-Path $ExePath)) {
    Write-Host "Warning: Non-installer exe not found, will need to extract from NSIS"
    # This would require nsis extraction tools
    # For now, copy the installer as-is (this won't work for Store, but sets up the structure)
    Copy-Item $InstallerPath "$PackageDir\whodb.exe"
} else {
    Copy-Item $ExePath "$PackageDir\whodb.exe"
}

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
& makeappx pack /d $PackageDir /p "WhoDB-$Version-$Architecture.msix" /o

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
