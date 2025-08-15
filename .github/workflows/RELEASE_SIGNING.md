# Release Signing Setup

This document explains how release signing is implemented for bera-geth, including PGP signing for binaries and Cosign signing for Docker images.

## Overview

The bera-geth release process includes cryptographic signing for all release artifacts:
- **Binary releases**: Signed with PGP/GPG
- **Docker images**: Signed with Cosign (keyless signing via OIDC)

## Configuration

### Prerequisites for Release Managers

#### PGP Signing Setup

1. **Create the `LINUX_SIGNING_KEY` secret in GitHub**:
   ```bash
   # Export your PGP private key
   gpg --export-secret-keys --armor YOUR_KEY_ID > private.key
   
   # Base64 encode it
   cat private.key | base64 -w 0
   
   # Copy the output and add it as the LINUX_SIGNING_KEY secret in GitHub repository settings
   ```

2. **Ensure the public key matches**: The public key at `.github/workflows/release.asc` should correspond to the private key used for signing.

#### Cosign Setup

Cosign uses keyless signing (no secret required). The workflow uses GitHub's OIDC provider to sign images, which means:
- No private keys to manage
- Signatures are tied to the GitHub Actions workflow identity
- Full transparency via Rekor transparency log

## Release Process

### Triggering a Release

Releases are triggered by:
- Pushing a tag matching `v1.*` (e.g., `v1.0.0`, `v1.0.0-rc1`)
- The workflow will automatically create a draft release with all signed artifacts

### What Gets Signed

1. **Binary Archives**:
   - `bera-geth-linux-amd64-*.tar.gz` → `bera-geth-linux-amd64-*.tar.gz.asc`
   - `bera-geth-linux-arm64-*.tar.gz` → `bera-geth-linux-arm64-*.tar.gz.asc`
   - `bera-geth-alltools-linux-amd64-*.tar.gz` → `bera-geth-alltools-linux-amd64-*.tar.gz.asc`
   - `bera-geth-alltools-linux-arm64-*.tar.gz` → `bera-geth-alltools-linux-arm64-*.tar.gz.asc`

2. **Docker Images**:
   - Multi-arch images at `ghcr.io/berachain/bera-geth:VERSION`
   - Signed with Cosign using keyless signing

## Verification Instructions

### Verifying Binary Signatures

Users can verify the PGP signatures of release binaries:

```bash
# Download and import the public key
curl -sSL https://raw.githubusercontent.com/berachain/bera-geth/main/.github/workflows/release.asc | gpg --import

# Download a release archive and its signature
wget https://github.com/berachain/bera-geth/releases/download/v1.0.0/bera-geth-linux-amd64-v1.0.0.tar.gz
wget https://github.com/berachain/bera-geth/releases/download/v1.0.0/bera-geth-linux-amd64-v1.0.0.tar.gz.asc

# Verify the signature
gpg --verify bera-geth-linux-amd64-v1.0.0.tar.gz.asc bera-geth-linux-amd64-v1.0.0.tar.gz
```

Expected output:
```
gpg: Signature made [date] using RSA key ID [key-id]
gpg: Good signature from "bera-geth-linux-signing-key"
```

### Verifying Docker Images

Docker images are signed with Cosign and can be verified:

```bash
# Install cosign if not already installed
brew install cosign  # macOS
# or see https://docs.sigstore.dev/cosign/installation/

# Verify a specific version
cosign verify ghcr.io/berachain/bera-geth:v1.0.0

# Verify the latest image
cosign verify ghcr.io/berachain/bera-geth:latest
```

The verification will show:
- The GitHub Actions workflow that created the image
- The commit SHA
- The OIDC issuer (GitHub)

## Security Considerations

1. **PGP Key Security**:
   - The PGP private key should be kept secure and only accessible to authorized release managers
   - Regularly rotate keys and update the public key in the repository
   - Use a strong passphrase for the private key

2. **Cosign Keyless Signing**:
   - Signatures are tied to the GitHub Actions workflow identity
   - Verification includes checking the workflow that signed the image
   - All signatures are recorded in the Rekor transparency log

3. **Best Practices**:
   - Always verify signatures before using release artifacts in production
   - Check that the signing workflow matches the official repository
   - Monitor the repository for any changes to signing keys or workflows

## Troubleshooting

### PGP Signing Issues

If PGP signing fails:
1. Check that the `LINUX_SIGNING_KEY` secret is properly set
2. Verify the key hasn't expired: `gpg --list-secret-keys`
3. Ensure the base64 encoding was done correctly

### Cosign Signing Issues

If Cosign signing fails:
1. Ensure the workflow has `id-token: write` permission
2. Check that the Docker image was successfully pushed before signing
3. Verify the image tag/digest is correct

### Release Draft Issues

If the release draft fails:
1. Ensure all artifacts were successfully uploaded
2. Check that the tag follows the correct format (`v1.*`)
3. Verify the workflow has `contents: write` permission
