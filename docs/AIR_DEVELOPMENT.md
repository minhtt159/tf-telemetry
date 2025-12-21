# Air Development Guide

This guide provides examples for using Air (hot reload) for development of tf-telemetry.

## What is Air?

Air is a live-reload tool for Go applications. It watches your source files and automatically rebuilds and restarts your application when changes are detected, making development faster and more efficient.

## Local Development

### Prerequisites

```bash
# Install Air
go install github.com/cosmtrek/air@latest

# Verify installation
air -v
```

### Basic Usage

```bash
# Start development server with hot reload
air

# Air will:
# 1. Build the application to tmp/main
# 2. Start the server
# 3. Watch for file changes
# 4. Rebuild and restart automatically on changes
```

### Configuration

The `.air.toml` file configures Air for local development:

- **Watched files**: `.go`, `.yaml`, `.html` files
- **Excluded**: `_test.go`, `vendor/`, `tmp/`, `bin/`, `client/`, `api/proto/`
- **Build command**: `go build -o ./tmp/main ./cmd/app`
- **Hot reload**: Enabled with 1 second delay after file changes
- **Graceful restart**: Sends interrupt signal before killing the process

### Example Workflow

1. Start Air:
   ```bash
   air
   ```

2. Edit any Go file (e.g., `internal/server/server.go`)

3. Save the file

4. Air automatically:
   - Detects the change
   - Rebuilds the binary
   - Restarts the server
   - Shows build output or errors

5. Test your changes immediately at `http://localhost:8080`

### Tips

- Air cleans up the `tmp/` directory on exit
- Build errors are logged to `tmp/build-errors.log`
- Use `Ctrl+C` to stop Air and the server
- The screen clears on each rebuild for better readability

## Docker Development

### Why Docker + Air?

- Consistent development environment across team members
- No need to install Go locally
- Isolated dependencies
- Closer to production environment

### Prerequisites

```bash
# Only Docker and Docker Compose needed
docker --version
docker-compose --version
```

### Basic Usage

```bash
# Start development environment with hot reload
docker-compose -f compose.dev.yaml up

# Or rebuild images first
docker-compose -f compose.dev.yaml up --build

# Stop the environment
docker-compose -f compose.dev.yaml down
```

### Configuration

The Docker setup includes:

1. **`build/Dockerfile.dev`**: Development Dockerfile with Air installed
2. **`.air-docker.toml`**: Air config optimized for Docker (uses polling)
3. **`compose.dev.yaml`**: Docker Compose with volume mounts for hot reload

### How It Works

The `compose.dev.yaml` mounts your local source code into the container:

```yaml
volumes:
  - ./:/app:cached        # Mount source code
  - /app/tmp             # Exclude tmp from host
```

When you edit files locally:
1. Changes are synced to the container
2. Air detects changes via polling (required for Docker volumes)
3. Air rebuilds inside the container
4. Server restarts automatically

### Example Workflow

1. Start the development stack:
   ```bash
   docker-compose -f compose.dev.yaml up
   ```

2. Wait for initial build (first time may take 1-2 minutes)

3. Edit Go files on your local machine using your favorite editor

4. Save changes

5. Watch the Docker logs - Air will rebuild and restart:
   ```
   telemetry-server_1  | building...
   telemetry-server_1  | running...
   ```

6. Test changes at `http://localhost:8080` and `http://localhost:3000`

### Tips

- The first build downloads dependencies and may take time
- Subsequent rebuilds are fast (5-10 seconds)
- Use `docker-compose -f compose.dev.yaml logs -f telemetry-server` to follow logs
- File polling is used (`.air-docker.toml` has `poll = true`) for Docker compatibility
- The web client is also available for testing at port 3000

### Troubleshooting

**Changes not detected?**
- Ensure polling is enabled in `.air-docker.toml` (`poll = true`)
- Check if files are excluded in `exclude_dir` or `exclude_regex`
- Verify volume mounts in `compose.dev.yaml`

**Build errors?**
- Check `tmp/build-errors.log` in the container
- View logs: `docker-compose -f compose.dev.yaml logs telemetry-server`

**Slow rebuild?**
- First build downloads dependencies (slow)
- Subsequent builds should be fast
- Consider increasing `poll_interval` if CPU usage is high

## Comparison: Local vs Docker

| Feature | Local Air | Docker Air |
|---------|-----------|------------|
| Setup | Requires Go installed | Only needs Docker |
| Speed | Faster rebuilds | Slightly slower first build |
| Environment | Uses local Go version | Consistent Go version |
| Isolation | Shares host environment | Isolated container |
| Config | `.air.toml` | `.air-docker.toml` |
| File watching | fsnotify | Polling |
| Best for | Quick iteration | Team consistency |

## Advanced Configuration

### Custom Air Settings

Edit `.air.toml` (local) or `.air-docker.toml` (Docker) to customize:

```toml
[build]
  # Delay after file change before rebuild (ms)
  delay = 1000
  
  # Send interrupt signal for graceful shutdown
  send_interrupt = true
  
  # Delay after interrupt before kill (ms)
  kill_delay = 500
  
  # Additional files to watch
  include_ext = ["go", "yaml", "html"]
  
  # Exclude patterns
  exclude_regex = ["_test.go"]
```

### Running with Different Configs

```bash
# Use custom config
air -c config-demo.yaml

# Pass arguments to the binary
# Edit .air.toml and set: args_bin = ["--config", "custom.yaml"]
```

### Integration with IDEs

**VS Code**: Install "Run on Save" extension and configure:
```json
{
  "emeraldwalk.runonsave": {
    "commands": [
      {
        "match": "\\.go$",
        "cmd": "air"
      }
    ]
  }
}
```

**GoLand/IntelliJ**: Air works automatically when running in terminal.

## Production vs Development

Remember:

- **Development**: Use Air for hot reload (`.air.toml`)
- **Production**: Use compiled binary or `build/Dockerfile`

Air is **only** for development. Production builds should use:
```bash
# Production build
make build

# Or Docker production image
docker build -f build/Dockerfile -t tf-telemetry:latest .
```
