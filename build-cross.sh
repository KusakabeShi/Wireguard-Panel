#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Build information
VERSION=${VERSION:-"dev"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_USER=$(whoami)

echo -e "${GREEN}WG-Panel Cross-Platform Build Script${NC}"
echo -e "${YELLOW}Version: ${VERSION}${NC}"
echo -e "${YELLOW}Commit: ${COMMIT}${NC}"
echo -e "${YELLOW}Build Date: ${BUILD_DATE}${NC}"
echo -e "${YELLOW}Build User: ${BUILD_USER}${NC}"
echo ""

# Build targets - Linux only due to netlink/pcap dependencies
TARGETS=(
    "linux/amd64"
    "linux/arm64"
)

# Check if we're in the correct directory
if [ ! -f "main.go" ]; then
    echo -e "${RED}Error: main.go not found. Please run this script from the project root.${NC}"
    exit 1
fi

# Build frontend first (only once)
echo -e "${GREEN}Building frontend...${NC}"
cd frontend
if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}Installing frontend dependencies...${NC}"
    npm install
fi
echo -e "${YELLOW}Compiling frontend...${NC}"
npm run build
echo -e "${GREEN}Frontend build completed successfully.${NC}"
cd ..

# Create releases directory
mkdir -p releases
rm -f releases/*

# Set common build flags
LDFLAGS="-X wg-panel/internal/version.Version=${VERSION} \
-X wg-panel/internal/version.GitCommit=${COMMIT} \
-X wg-panel/internal/version.BuildDate=${BUILD_DATE} \
-X wg-panel/internal/version.BuildUser=${BUILD_USER} \
-w -s"

echo -e "${GREEN}Building binaries for multiple platforms...${NC}"
echo ""

for target in "${TARGETS[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$target"
    
    echo -e "${BLUE}Building for ${GOOS}/${GOARCH}...${NC}"
    
    BINARY_NAME="wg-panel"
    ARCHIVE_NAME="wg-panel-${VERSION}-${GOOS}-${GOARCH}"
    
    if [ "$GOOS" == "windows" ]; then
        BINARY_NAME="wg-panel.exe"
    fi
    
    # Build binary
    env GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o "$BINARY_NAME" .
    
    # Create archive
    if [ "$GOOS" == "windows" ]; then
        zip "releases/${ARCHIVE_NAME}.zip" "$BINARY_NAME"
        rm "$BINARY_NAME"
        echo -e "${YELLOW}Created: releases/${ARCHIVE_NAME}.zip${NC}"
    else
        tar -czf "releases/${ARCHIVE_NAME}.tar.gz" "$BINARY_NAME"
        rm "$BINARY_NAME"
        echo -e "${YELLOW}Created: releases/${ARCHIVE_NAME}.tar.gz${NC}"
    fi
    
    echo ""
done

echo -e "${GREEN}Cross-platform build completed successfully!${NC}"
echo -e "${YELLOW}Release files created in ./releases/:${NC}"
ls -lh releases/
echo ""
echo -e "${BLUE}Total size: $(du -sh releases/ | cut -f1)${NC}"