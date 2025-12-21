# Docker Development Setup

Quick reference for Docker development with Air hot reload.

## Files

- `build/Dockerfile.dev` - Development Dockerfile with Air installed
- `compose.dev.yaml` - Docker Compose for development with volume mounts
- `.air-docker.toml` - Air configuration optimized for Docker (uses polling)

## Quick Start

```bash
# Start the development environment
docker-compose -f compose.dev.yaml up

# Rebuild if Dockerfile changed
docker-compose -f compose.dev.yaml up --build

# Stop the environment
docker-compose -f compose.dev.yaml down
```

## How It Works

1. **Dockerfile.dev** builds an image with:
   - Go 1.25
   - Air hot reload tool pre-installed
   - Project dependencies downloaded

2. **compose.dev.yaml** mounts your source code:
   - Local files → `/app` in container
   - Changes detected via polling
   - Air rebuilds and restarts automatically

3. **.air-docker.toml** uses:
   - `poll = true` for Docker compatibility
   - File system polling instead of fsnotify
   - Optimized for containerized environments

## Troubleshooting

### Air not installed in container

If Air wasn't installed during build (network issues), install it manually:

```bash
# Enter the container
docker-compose -f compose.dev.yaml exec telemetry-server sh

# Install Air
go install github.com/air-verse/air@latest

# Air should now be in /go/bin/air
```

Or rebuild the image:
```bash
docker-compose -f compose.dev.yaml build --no-cache telemetry-server
```

### Changes not detected

Ensure:
- Volume mounts are correct in `compose.dev.yaml`
- Polling is enabled in `.air-docker.toml` (`poll = true`)
- Files aren't in excluded directories

### Build is slow

First build downloads dependencies (can take 2-3 minutes). Subsequent builds are fast.

To speed up:
```bash
# Use BuildKit
export DOCKER_BUILDKIT=1
docker-compose -f compose.dev.yaml build
```

## What Gets Reloaded

Air watches and rebuilds when you change:
- `.go` files
- `.yaml` configuration files
- `.html` template files (if added)

Air does NOT watch:
- `*_test.go` files
- `vendor/` directory
- `tmp/` directory
- `client/` directory
- `api/proto/` directory

## Ports

- **8080** - HTTP API
- **50051** - gRPC API
- **3000** - Web client (if started)

## Environment Variables

Set in `compose.dev.yaml`:
- `TZ=UTC` - Timezone
- `CGO_ENABLED=0` - Disable CGO for faster builds

## Tips

1. **First run**: Takes longer to download dependencies
2. **Logs**: Use `docker-compose -f compose.dev.yaml logs -f` to follow
3. **Restart**: Changes to `config.yaml` are picked up automatically
4. **Clean**: `docker-compose -f compose.dev.yaml down -v` removes volumes
