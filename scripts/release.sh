#!/bin/bash

# Build script for multiple platforms
BINARY_NAME="openjudges"
VERSION=${1:-"v0.0.1"} # Default version or from first argument

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

DIST_DIR="dist"
mkdir -p $DIST_DIR

platforms=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "linux/arm/7"
    "linux/arm/6"
    "windows/amd64"
    "windows/386"
    "freebsd/amd64"
    "freebsd/arm64"
    "netbsd/amd64"
    "openbsd/amd64"
)

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    GOARM=${platform_split[2]:-""}
    
    target="${GOOS}-${GOARCH}"
    if [ -n "$GOARM" ]; then
        target="${GOOS}-armv${GOARM}"
    fi
    
    binary_name="$BINARY_NAME"
    if [ $GOOS = "windows" ]; then
        binary_name="${BINARY_NAME}.exe"
    fi
    
    archive_name="${BINARY_NAME}_${target}"
    
    # Determine archive extension
    if [ $GOOS = "windows" ] || [ $GOOS = "darwin" ]; then
        archive_ext=".zip"
    else
        archive_ext=".tar.gz"
    fi
    
    archive_file="${archive_name}${archive_ext}"
    
    echo "Building ${archive_file}..."
    
    # Build binary
    if [ -n "$GOARM" ]; then
        GOOS=$GOOS GOARCH=$GOARCH GOARM=$GOARM go build -o "$DIST_DIR/$binary_name" ./main.go
    else
        GOOS=$GOOS GOARCH=$GOARCH go build -o "$DIST_DIR/$binary_name" ./main.go
    fi
    
    # Create archive
    if [ "$archive_ext" = ".zip" ]; then
        cd "$DIST_DIR"
        zip -q "$archive_file" "$binary_name"
        rm "$binary_name"
        cd "$PROJECT_ROOT"
    else
        cd "$DIST_DIR"
        tar -czf "$archive_file" "$binary_name"
        rm "$binary_name"
        cd "$PROJECT_ROOT"
    fi
done

echo "Build complete! Artifacts are in $DIST_DIR/"
