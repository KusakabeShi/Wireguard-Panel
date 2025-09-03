# WG-Panel Build System

This document explains how to build WG-Panel from source and create releases.

## Prerequisites

**⚠️ Linux Only**: This application only works on Linux due to dependencies on:
- `netlink` (Linux kernel interface for network management)
- `libpcap` (raw packet capture for pseudo-bridge functionality)

**Build requirements:**
- Go 1.21 or later
- Node.js 18 or later  
- npm (comes with Node.js)
- Git (for version information)
- libpcap-dev (for packet capture)

## Quick Build

To build for your current platform:

```bash
./build.sh
```

This will:
1. Build the React frontend in `frontend/`
2. Embed the frontend files into the Go binary using `embed.FS`
3. Build the Go binary with version information
4. Create `wg-panel` executable

## Cross-Platform Build

To build for all supported platforms:

```bash
./build-cross.sh
```

This creates release archives in the `releases/` directory for:
- Linux AMD64/ARM64

**Note**: Windows and macOS are not supported due to netlink/libpcap dependencies.

## Version Management

### Setting Version

You can set the version when building:

```bash
VERSION="v1.2.3" ./build.sh
```

### Version Information

The binary includes detailed version information:

```bash
./wg-panel -v
```

Output example:
```
WG-Panel v1.0.0 (commit: abc1234, built: 2025-09-02_12:34:56 by user, go: go1.21.0)
```

## GitHub Actions

### Automatic Releases

The project includes GitHub Actions workflows that automatically:

1. **On Tag Push**: Builds and creates a release when you push a version tag:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **Manual Trigger**: You can manually trigger a release from the GitHub Actions tab

### Release Assets

Each release includes pre-built binaries for all supported platforms:
- `wg-panel-v1.0.0-linux-amd64.tar.gz`
- `wg-panel-v1.0.0-linux-arm64.tar.gz`
- `wg-panel-v1.0.0-windows-amd64.zip`
- `wg-panel-v1.0.0-darwin-amd64.tar.gz`
- `wg-panel-v1.0.0-darwin-arm64.tar.gz`

## Build Configuration

### LDFLAGS

The build system sets these version variables at compile time:
- `wg-panel/internal/version.Version`
- `wg-panel/internal/version.GitCommit`
- `wg-panel/internal/version.BuildDate`
- `wg-panel/internal/version.BuildUser`

### Frontend Integration

The frontend is built using Create React App and embedded into the Go binary using:
```go
//go:embed all:frontend/build
var frontendFS embed.FS
```

This means the final binary includes all frontend assets and can serve them without external files.

## Development Workflow

1. Make your changes to Go or React code
2. Test locally: `./build.sh && ./wg-panel`
3. For releases: 
   - Update version in git tag
   - Push tag to trigger automated release
   - Or manually trigger release via GitHub Actions

## Troubleshooting

### Frontend Build Fails
- Ensure Node.js 18+ is installed
- Run `cd frontend && npm install` to install dependencies
- Check for any React compilation errors

### Go Build Fails
- Ensure Go 1.21+ is installed
- Check that all Go dependencies are available: `go mod tidy`
- Verify embed path exists: `frontend/build/`

### Version Shows "dev"
- Make sure you set the VERSION environment variable
- For git commits, ensure you're in a git repository
- Check that git is installed and working