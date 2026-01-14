#!/bin/bash

# Build script for multiple platforms
BINARY_NAME="openllmjudge"
VERSION=${1:-"v1.0.0"} # Default version or from first argument
DIST_DIR="dist"

mkdir -p $DIST_DIR

platforms=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
)

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    
    output_name="${BINARY_NAME}_${GOOS}_${GOARCH}"
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi

    echo "Building $output_name..."
    GOOS=$GOOS GOARCH=$GOARCH go build -o "$DIST_DIR/$output_name" ./main.go
done

echo "Build complete! Artifacts are in $DIST_DIR/"
