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
    [string]$PublisherCN = "Test Publisher",

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
# The workflow builds raw exe for MSIX packaging
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
Write-Host "Version parameter: $Version"
Write-Host "Architecture parameter: $Architecture"

# Try to run makeappx directly first (it should be in PATH if SDK is installed properly)
try {
    $msixFileName = "WhoDB-$Version-$Architecture.msix"
    Write-Host "Creating MSIX with filename: $msixFileName"
    Write-Host "Running: makeappx pack /d $PackageDir /p $msixFileName /o"
    makeappx pack /d $PackageDir /p $msixFileName /o

    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ MSIX package created successfully"
    } else {
        throw "makeappx failed with exit code: $LASTEXITCODE"
    }
} catch {
    Write-Host "Failed to run makeappx directly: $_"
    Write-Host ""

    # If direct call fails, try to find and add to PATH
    Write-Host "Searching for makeappx.exe in Windows SDK..."

    $makeappxFound = $false
    $sdkRoot = "C:\Program Files (x86)\Windows Kits\10\bin"

    if (Test-Path $sdkRoot) {
        Write-Host "Found SDK root: $sdkRoot"

        # Get all SDK versions and sort by version number (newest first)
        $sdkVersions = @(Get-ChildItem -Path $sdkRoot -Directory -ErrorAction SilentlyContinue |
            Where-Object { $_.Name -match "^10\.\d+\.\d+\.\d+$" } |
            Sort-Object { [version]$_.Name } -Descending)

        if ($null -eq $sdkVersions -or $sdkVersions.Count -eq 0) {
            Write-Host "No SDK versions found in $sdkRoot"
        } else {
            Write-Host "Found $($sdkVersions.Count) SDK version(s) (sorted newest first):"
            foreach ($v in $sdkVersions) {
                Write-Host "  - $($v.Name) [Type: $($v.GetType().Name)]"
            }
            Write-Host "Will use the newest version that has makeappx.exe"
        }

        foreach ($sdkVersion in $sdkVersions) {
            # Debug what we're getting
            Write-Host "Processing version object: Type=$($sdkVersion.GetType().Name), Name=$($sdkVersion.Name), ToString=$($sdkVersion.ToString())"

            # Construct path more explicitly - sdkVersion.Name contains the version number
            if ($sdkVersion -is [System.IO.DirectoryInfo]) {
                $versionFolder = $sdkVersion.Name
            } else {
                $versionFolder = $sdkVersion.ToString()
            }

            if ([string]::IsNullOrEmpty($versionFolder)) {
                Write-Host "Warning: Version folder name is empty, skipping"
                continue
            }

            $versionPath = Join-Path $sdkRoot $versionFolder
            $x64Path = Join-Path $versionPath "x64"
            $makeappxPath = Join-Path $x64Path "makeappx.exe"
            Write-Host "Checking SDK $versionFolder : $makeappxPath"
            if (Test-Path $makeappxPath) {
                Write-Host "✅ Found makeappx in SDK version $versionFolder"
                Write-Host "Using: $makeappxPath"
                Write-Host "Creating MSIX package..."

                $msixFileName = "WhoDB-$Version-$Architecture.msix"
                Write-Host "Creating MSIX with filename: $msixFileName"
                & $makeappxPath pack /d $PackageDir /p $msixFileName /o

                if ($LASTEXITCODE -eq 0) {
                    Write-Host "✅ MSIX package created successfully using SDK $versionFolder"
                    $makeappxFound = $true
                } else {
                    Write-Error "makeappx failed with exit code: $LASTEXITCODE"
                    exit 1
                }
                break
            } else {
                Write-Host "  Not found"
            }
        }
    } else {
        Write-Host "SDK root not found at: $sdkRoot"
    }

    # If still not found, try alternative search
    if (-not $makeappxFound) {
        Write-Host "Attempting wider search for makeappx.exe..."

        $searchPaths = @(
            "C:\Program Files (x86)\Windows Kits",
            "C:\Program Files\Windows Kits",
            "C:\Program Files (x86)\Microsoft SDKs",
            "C:\Program Files\Microsoft SDKs"
        )

        foreach ($searchPath in $searchPaths) {
            if (Test-Path $searchPath) {
                Write-Host "Searching in: $searchPath"
                $found = Get-ChildItem -Path $searchPath -Filter "makeappx.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
                if ($found) {
                    Write-Host "Found makeappx at: $($found.FullName)"
                    if ($found.FullName -like "*App Certification Kit*") {
                        Write-Host "Note: Using App Certification Kit version (SDK versions not found)"
                    }
                    $msixFileName = "WhoDB-$Version-$Architecture.msix"
                    Write-Host "Creating MSIX with filename: $msixFileName"
                    & $found.FullName pack /d $PackageDir /p $msixFileName /o
                    if ($LASTEXITCODE -eq 0) {
                        Write-Host "MSIX package created successfully"
                        $makeappxFound = $true
                        break
                    }
                }
            }
        }
    }

    if (-not $makeappxFound) {
        Write-Error "makeappx.exe not found. Please install Windows SDK."
        Write-Host ""
        Write-Host "Install Windows SDK from:"
        Write-Host "https://developer.microsoft.com/windows/downloads/windows-sdk/"
        Write-Host ""
        Write-Host "After installation, you may need to:"
        Write-Host "1. Restart PowerShell"
        Write-Host "2. Or manually add to PATH: C:\Program Files (x86)\Windows Kits\10\bin\[version]\x64"
        exit 1
    }
}

# Sign the package if certificate is provided
if (-not $SkipSigning -and $CertPath) {
    if (-not (Test-Path $CertPath)) {
        Write-Error "Certificate file not found: $CertPath"
        exit 1
    }
    Write-Host "Signing MSIX package..."
    & signtool sign /fd SHA256 /a /f $CertPath /p $env:WINDOWS_PFX_PASSWORD "WhoDB-$Version-$Architecture.msix"
    Write-Host "Signed MSIX package created: WhoDB-$Version-$Architecture.msix"
} else {
    Write-Host "Unsigned MSIX package created: WhoDB-$Version-$Architecture.msix"
    Write-Host "Note: This package will be signed by Microsoft when uploaded to Partner Center"
}
