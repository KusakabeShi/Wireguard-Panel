#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Build information
VERSION=${VERSION:-"dev"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_USER=$(whoami)

echo -e "${GREEN}WG-Panel Build Script${NC}"
echo -e "${YELLOW}Version: ${VERSION}${NC}"
echo -e "${YELLOW}Commit: ${COMMIT}${NC}"
echo -e "${YELLOW}Build Date: ${BUILD_DATE}${NC}"
echo -e "${YELLOW}Build User: ${BUILD_USER}${NC}"
echo ""

# Check if running on Linux (required for netlink/pcap dependencies)
if [ "$(uname -s)" != "Linux" ]; then
    echo -e "${RED}Warning: This application is designed for Linux only.${NC}"
    echo -e "${YELLOW}Dependencies on netlink and libpcap may not work on $(uname -s).${NC}"
    echo -e "${YELLOW}Continuing build anyway...${NC}"
    echo ""
fi

# Check if we're in the correct directory
if [ ! -f "main.go" ]; then
    echo -e "${RED}Error: main.go not found. Please run this script from the project root.${NC}"
    exit 1
fi

# Check if frontend directory exists
if [ ! -d "frontend" ]; then
    echo -e "${RED}Error: frontend directory not found.${NC}"
    exit 1
fi

# Build frontend
echo -e "${GREEN}Building frontend...${NC}"
cd frontend

# Check if package.json exists
if [ ! -f "package.json" ]; then
    echo -e "${RED}Error: package.json not found in frontend directory.${NC}"
    exit 1
fi

# Install dependencies if node_modules doesn't exist
if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}Installing frontend dependencies...${NC}"
    npm install
fi

# Build frontend
echo -e "${YELLOW}Compiling frontend...${NC}"
npm run build

# Check if build was successful
if [ ! -d "build" ]; then
    echo -e "${RED}Error: Frontend build failed - build directory not found.${NC}"
    exit 1
fi

echo -e "${GREEN}Frontend build completed successfully.${NC}"

# Go back to project root
cd ..

# Build Go binary
echo -e "${GREEN}Building Go binary...${NC}"

# Set build flags with version information
LDFLAGS="-X wg-panel/internal/version.Version=${VERSION} \
-X wg-panel/internal/version.GitCommit=${COMMIT} \
-X wg-panel/internal/version.BuildDate=${BUILD_DATE} \
-X wg-panel/internal/version.BuildUser=${BUILD_USER} \
-w -s"

# Build for current platform
echo -e "${YELLOW}Compiling Go binary for $(go env GOOS)/$(go env GOARCH)...${NC}"
go build -ldflags "${LDFLAGS}" -o wg-panel .

# Verify the build
if [ -f "wg-panel" ]; then
    echo -e "${GREEN}Build completed successfully!${NC}"
    echo -e "${YELLOW}Binary: ./wg-panel${NC}"
    echo -e "${YELLOW}Size: $(du -h wg-panel | cut -f1)${NC}"
    echo ""
    echo -e "${GREEN}Version information:${NC}"
    ./wg-panel -v
else
    echo -e "${RED}Error: Go build failed - binary not found.${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}Build process completed successfully!${NC}"