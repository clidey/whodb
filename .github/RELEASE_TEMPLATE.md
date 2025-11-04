{{PR_DESCRIPTION}}

## Installation

### Mac App Store
Coming soon

### Microsoft Store
[Download from Microsoft Store](https://apps.microsoft.com/detail/9pftx5bv4ds6)

### Snap Store
```bash
sudo snap install whodb
```
[View on Snapcraft](https://snapcraft.io/whodb)

### AppImage (Linux)

Download the AppImage for your architecture from the assets below, make it executable, and run:

```bash
# For AMD64/x86_64
chmod +x WhoDB-{{VERSION}}-amd64.AppImage
./WhoDB-{{VERSION}}-amd64.AppImage

# For ARM64/aarch64
chmod +x WhoDB-{{VERSION}}-arm64.AppImage
./WhoDB-{{VERSION}}-arm64.AppImage
```

All AppImages are signed with Sigstore. To verify:

```bash
cosign verify-blob --signature WhoDB-{{VERSION}}-amd64.AppImage.sig --certificate WhoDB-{{VERSION}}-amd64.AppImage.pem WhoDB-{{VERSION}}-amd64.AppImage
```

### Docker
```bash
docker pull clidey/whodb:{{VERSION}}
docker pull clidey/whodb:latest
```

### Direct Downloads
See assets below for platform-specific packages (DMG, MSIX, etc.).

## Documentation

- [Documentation](https://whodb.com/docs)
- [Report Issues](https://github.com/clidey/whodb/issues)

## Upgrade Notes

To upgrade from a previous version:
- **Docker**: Pull the latest image and restart your container
- **Snap**: Run `sudo snap refresh whodb`
- **AppImage**: Download the new AppImage and replace the old one
- **Desktop Apps**: Download and install the new version

---
