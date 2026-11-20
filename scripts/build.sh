#!/bin/bash

# Build script for semantic-release
# Builds the binaries and packages them as go-c8y-cli extension archives
# (extension.yaml and bin/c8y-session-1password), which "c8y extensions install" downloads
# Usage: build.sh <version>

set -e

VERSION="$1"
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    exit 1
fi

echo "Starting build for version ${VERSION}..."

# Get commit hash and date
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "Commit: ${COMMIT}, Date: ${DATE}"

# Build flags
LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"
BINARY_NAME="c8y-session-1password"

# Files included in each archive next to the binary
FILES=(extension.yaml README.md LICENSE)

# Create a clean dist directory
rm -rf dist
mkdir -p dist
echo "Created dist directory"

echo "Building binaries for version ${VERSION}..."

# Build for different platforms
platforms=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for platform in "${platforms[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"

    binary="${BINARY_NAME}"
    if [ "$GOOS" = "windows" ]; then
        binary="${binary}.exe"
    fi

    # go-c8y-cli selects the asset by the <os>_<arch> suffix
    archive_name="${BINARY_NAME}_${VERSION}_${GOOS}_${GOARCH}"
    staging="dist/${archive_name}"
    mkdir -p "${staging}/bin"

    echo "Building ${archive_name}..."
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="$LDFLAGS" -o "${staging}/bin/${binary}" .
    cp "${FILES[@]}" "${staging}/"

    if [ "$GOOS" = "windows" ]; then
        (cd "${staging}" && zip -q -r "../${archive_name}.zip" "${FILES[@]}" bin)
    else
        tar -czf "dist/${archive_name}.tar.gz" -C "${staging}" "${FILES[@]}" bin
    fi
    rm -rf "${staging}"
done

echo "Generating checksums..."

# Generate checksums (GoReleaser format, verified by "c8y extensions install")
cd dist
if command -v sha256sum >/dev/null 2>&1; then
    sha256sum *.tar.gz *.zip > checksums.txt
elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 *.tar.gz *.zip > checksums.txt
else
    echo "Warning: No checksum utility found"
fi
cd ..

echo "Build complete! Artifacts in dist/"
ls -la dist/
