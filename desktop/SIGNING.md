# Code Signing Guide for WhoDB Desktop

This guide explains how to sign WhoDB desktop applications for each platform to prevent security warnings and ensure users can run the application without issues.

## Overview

Code signing is crucial for desktop applications to:
- Prevent security warnings on Windows (SmartScreen)
- Prevent Gatekeeper warnings on macOS
- Ensure authenticity and integrity on Linux
- Build user trust

## Prerequisites

### Windows
- Windows SDK with `signtool.exe`
- Valid code signing certificate (.pfx file)
- Certificate password
- Trusted Certificate Authority (CA) certificate

### macOS
- Xcode Command Line Tools
- Apple Developer ID certificate
- Apple Developer account (for notarization)
- App-specific password for notarization

### Linux
- GPG (GNU Privacy Guard)
- GPG key pair for signing
- dpkg-sig (for .deb packages)

## Platform-Specific Instructions

### Windows Signing

1. **Obtain a Code Signing Certificate**
   - Purchase from a trusted CA (DigiCert, Sectigo, etc.)
   - Or use a self-signed certificate for internal distribution

2. **Sign the Executable**
   ```powershell
   # Basic signing
   .\sign-windows.ps1 -ExePath "path\to\whodb.exe" -CertPath "path\to\cert.pfx" -CertPassword (Read-Host -AsSecureString)
   
   # With custom timestamp server
   .\sign-windows.ps1 -ExePath "path\to\whodb.exe" -CertPath "path\to\cert.pfx" -CertPassword $securePassword -TimestampServer "http://timestamp.sectigo.com"
   ```

3. **Verify the Signature**
   - Right-click the executable → Properties → Digital Signatures
   - Or use: `signtool verify /pa whodb.exe`

### macOS Signing

1. **Set Up Developer Certificate**
   - Enroll in Apple Developer Program
   - Create a "Developer ID Application" certificate
   - Install in Keychain Access

2. **Sign the App**
   ```bash
   # Sign only
   ./sign-macos.sh -a WhoDB.app -i "Developer ID Application: Your Name (TEAMID)"
   
   # Sign and notarize
   ./sign-macos.sh -a WhoDB.app -i "Developer ID Application: Your Name (TEAMID)" \
     -t TEAMID -u your@email.com -p app-specific-password
   ```

3. **Create App-Specific Password**
   - Go to https://appleid.apple.com
   - Sign in → Security → App-Specific Passwords
   - Generate a password for "WhoDB Notarization"

4. **Verify the Signature**
   ```bash
   # Check signature
   codesign -dv --verbose=4 WhoDB.app
   
   # Check Gatekeeper acceptance
   spctl -a -v WhoDB.app
   ```

### Linux Signing

1. **Create GPG Key** (if needed)
   ```bash
   gpg --full-generate-key
   # Choose: RSA and RSA, 4096 bits, no expiration
   ```

2. **Sign Files**
   ```bash
   # Sign AppImage with detached signature
   ./sign-linux.sh -f WhoDB.AppImage -k your@email.com -d
   
   # Sign .deb package
   ./sign-linux.sh -f whodb.deb -k your@email.com
   
   # Sign binary executable
   ./sign-linux.sh -f whodb -k your@email.com -d
   ```

3. **Distribute Public Key**
   ```bash
   # Export public key
   gpg --armor --export your@email.com > whodb-public.asc
   
   # Users import with:
   gpg --import whodb-public.asc
   ```

## Automated CI/CD Signing

### GitHub Actions Example

```yaml
name: Sign Release

on:
  release:
    types: [created]

jobs:
  sign-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Sign Windows Binary
        env:
          CERT_BASE64: ${{ secrets.WINDOWS_CERT_BASE64 }}
          CERT_PASSWORD: ${{ secrets.WINDOWS_CERT_PASSWORD }}
        run: |
          # Decode certificate
          [System.Convert]::FromBase64String($env:CERT_BASE64) | Set-Content cert.pfx -Encoding Byte
          
          # Sign
          .\desktop\sign-windows.ps1 -ExePath "dist\whodb.exe" -CertPath "cert.pfx" -CertPassword (ConvertTo-SecureString $env:CERT_PASSWORD -AsPlainText -Force)
          
          # Clean up
          Remove-Item cert.pfx

  sign-macos:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Sign macOS App
        env:
          MACOS_CERTIFICATE: ${{ secrets.MACOS_CERTIFICATE }}
          MACOS_CERTIFICATE_PWD: ${{ secrets.MACOS_CERTIFICATE_PWD }}
          APPLE_ID: ${{ secrets.APPLE_ID }}
          APPLE_PASSWORD: ${{ secrets.APPLE_PASSWORD }}
          TEAM_ID: ${{ secrets.TEAM_ID }}
        run: |
          # Import certificate
          echo $MACOS_CERTIFICATE | base64 --decode > certificate.p12
          security create-keychain -p actions temp.keychain
          security import certificate.p12 -k temp.keychain -P $MACOS_CERTIFICATE_PWD -T /usr/bin/codesign
          security set-key-partition-list -S apple-tool:,apple: -s -k actions temp.keychain
          
          # Sign and notarize
          ./desktop/sign-macos.sh -a dist/WhoDB.app -i "Developer ID Application: Company (TEAMID)" \
            -t $TEAM_ID -u $APPLE_ID -p $APPLE_PASSWORD
          
          # Clean up
          security delete-keychain temp.keychain
          rm certificate.p12
```

## Security Best Practices

1. **Protect Signing Keys**
   - Never commit certificates or passwords to version control
   - Use secure secret management (GitHub Secrets, HashiCorp Vault, etc.)
   - Rotate certificates before expiration

2. **Timestamp All Signatures**
   - Ensures signatures remain valid after certificate expiration
   - Use reliable timestamp servers

3. **Verify Before Distribution**
   - Always verify signatures after signing
   - Test on clean systems without development certificates

4. **Certificate Management**
   - Keep certificates in hardware security modules (HSMs) when possible
   - Use separate certificates for development and production
   - Document certificate expiration dates

## Troubleshooting

### Windows Issues
- **"Certificate not trusted"**: Ensure using a certificate from a trusted CA
- **"Timestamp failed"**: Try a different timestamp server
- **SmartScreen warnings**: Build reputation over time with consistent signing

### macOS Issues
- **"Identity not found"**: Check certificate is in Keychain and valid
- **Notarization fails**: Ensure all binaries are signed with hardened runtime
- **"Damaged app"**: Usually indicates notarization is required

### Linux Issues
- **"No secret key"**: Ensure GPG key is available in keyring
- **Verification fails**: Check file wasn't modified after signing
- **Key trust issues**: Users need to import and trust your public key

## Additional Resources

- [Windows Authenticode Signing](https://docs.microsoft.com/en-us/windows/win32/seccrypto/cryptography-tools)
- [Apple Code Signing Guide](https://developer.apple.com/documentation/security/code_signing_services)
- [GPG Documentation](https://www.gnupg.org/documentation/)
- [Tauri Code Signing](https://tauri.app/v1/guides/distribution/sign)