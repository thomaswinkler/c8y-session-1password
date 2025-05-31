#!/bin/bash

# Build script for semantic-release
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

# Create dist directory
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
    
    output_name="${BINARY_NAME}-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    echo "Building ${output_name}..."
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="$LDFLAGS" -o "dist/${output_name}" .
done

echo "Creating archives..."

# Create archives for each platform
cd dist

# Linux amd64
echo "Creating Linux amd64 archive..."
tar -czf "${BINARY_NAME}-linux-amd64.tar.gz" "${BINARY_NAME}-linux-amd64"

# Linux arm64
echo "Creating Linux arm64 archive..."
tar -czf "${BINARY_NAME}-linux-arm64.tar.gz" "${BINARY_NAME}-linux-arm64"

# macOS amd64
echo "Creating macOS amd64 archive..."
tar -czf "${BINARY_NAME}-darwin-amd64.tar.gz" "${BINARY_NAME}-darwin-amd64"

# macOS arm64
echo "Creating macOS arm64 archive..."
tar -czf "${BINARY_NAME}-darwin-arm64.tar.gz" "${BINARY_NAME}-darwin-arm64"

# Windows amd64
echo "Creating Windows amd64 archive..."
zip -q "${BINARY_NAME}-windows-amd64.zip" "${BINARY_NAME}-windows-amd64.exe"

echo "Generating checksums..."

# Generate checksums
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
