#!/bin/bash
#
# build.sh - Build script for Igor
#
# This script builds the Igor binary with version information embedded.
#

set -e

# Change to project root
cd "$(dirname "$0")/.."

# Read version from VERSION file
VERSION=$(cat VERSION)
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

# Default build for current platform
build_current() {
    echo "Building Igor v${VERSION} for current platform..."
    go build -ldflags "${LDFLAGS}" -o igor ./cmd/igor
    echo "Build complete: ./igor"
}

# Build for all major platforms
build_all() {
    echo "Building Igor v${VERSION} for all platforms..."
    
    mkdir -p dist
    
    # Linux AMD64
    echo "Building for linux/amd64..."
    GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o dist/igor-linux-amd64 ./cmd/igor
    
    # Linux ARM64
    echo "Building for linux/arm64..."
    GOOS=linux GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o dist/igor-linux-arm64 ./cmd/igor
    
    # Linux 386
    echo "Building for linux/386..."
    GOOS=linux GOARCH=386 go build -ldflags "${LDFLAGS}" -o dist/igor-linux-386 ./cmd/igor
    
    echo "All builds complete. Binaries are in ./dist/"
    ls -la dist/
}

# Show usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --all     Build for all supported platforms"
    echo "  --help    Show this help message"
    echo ""
    echo "Without options, builds for current platform only."
}

# Parse arguments
case "${1:-}" in
    --all)
        build_all
        ;;
    --help|-h)
        usage
        ;;
    "")
        build_current
        ;;
    *)
        echo "Unknown option: $1"
        usage
        exit 1
        ;;
esac
