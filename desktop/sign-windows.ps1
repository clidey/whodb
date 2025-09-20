# Windows Code Signing Script for WhoDB
# This script signs the Windows executable to prevent virus warnings
# Prerequisites:
# - Windows SDK with signtool.exe
# - Valid code signing certificate (.pfx file)
# - Certificate password

param(
    [Parameter(Mandatory=$true)]
    [string]$ExePath,
    
    [Parameter(Mandatory=$true)]
    [string]$CertPath,
    
    [Parameter(Mandatory=$true)]
    [SecureString]$CertPassword,
    
    [string]$TimestampServer = "http://timestamp.digicert.com"
)

# Check if signtool exists
$signtool = Get-Command signtool.exe -ErrorAction SilentlyContinue
if (-not $signtool) {
    # Try to find signtool in Windows SDK
    $sdkPath = "${env:ProgramFiles(x86)}\Windows Kits\10\bin"
    $signtoolPath = Get-ChildItem -Path $sdkPath -Recurse -Filter "signtool.exe" -ErrorAction SilentlyContinue | Select-Object -First 1
    
    if ($signtoolPath) {
        $signtool = $signtoolPath.FullName
    } else {
        Write-Error "signtool.exe not found. Please install Windows SDK."
        exit 1
    }
} else {
    $signtool = $signtool.Path
}

# Check if certificate exists
if (-not (Test-Path $CertPath)) {
    Write-Error "Certificate file not found: $CertPath"
    exit 1
}

# Check if executable exists
if (-not (Test-Path $ExePath)) {
    Write-Error "Executable file not found: $ExePath"
    exit 1
}

# Convert SecureString to plain text (required for signtool)
$BSTR = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($CertPassword)
$PlainPassword = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto($BSTR)

Write-Host "Signing $ExePath with certificate $CertPath..." -ForegroundColor Green

# Sign the executable
$signArgs = @(
    'sign',
    '/f', $CertPath,
    '/p', $PlainPassword,
    '/t', $TimestampServer,
    '/fd', 'sha256',
    '/v',
    $ExePath
)

try {
    & $signtool $signArgs
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Successfully signed $ExePath" -ForegroundColor Green
        
        # Verify the signature
        Write-Host "Verifying signature..." -ForegroundColor Yellow
        & $signtool verify /pa $ExePath
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "Signature verified successfully!" -ForegroundColor Green
        } else {
            Write-Warning "Signature verification failed!"
        }
    } else {
        Write-Error "Failed to sign executable"
        exit 1
    }
} finally {
    # Clear the password from memory
    [System.Runtime.InteropServices.Marshal]::ZeroFreeBSTR($BSTR)
    $PlainPassword = $null
}

Write-Host "Code signing completed!" -ForegroundColor Green