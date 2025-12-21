# tf-telemetry

A telemetry collection server for mobile and web applications, with support for metrics and logs ingestion via gRPC and HTTP.

## Build and test

```bash
make build   # builds ./bin/tf-telemetry
make test    # runs go test ./...
```

## Container image

Build the image and run it locally:

```bash
docker build -f build/Dockerfile -t tf-telemetry:local .
docker run --rm -p 8080:8080 -p 50051:50051 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  tf-telemetry:local
```


## Quick Start with Docker Compose

```bash
docker compose up
```

This will start:
- **Telemetry Server** on ports 8080 (HTTP) and 50051 (gRPC)
  - The HTTP endpoint accepts telemetry at `POST /v1/telemetry` and basic health is available at `/healthz`.
  - Enable basic auth in `config.yaml` and provide credentials with `curl -u user:pass ...` when needed.
- **Web Client Demo** on port 3000

Then open your browser to: **http://localhost:3000**

### Using the Web Client

The web client provides an interactive interface to:
- Send sample metrics (CPU, memory, battery)
- Send sample logs (with different severity levels)
- Configure server URL and basic authentication
- View sample payloads
- Manage offline queue (localStorage-based retry mechanism)

Default configuration:
- **Server URL**: `http://localhost:8080`
- **Basic Auth**: Enabled in demo config with username `demo` and password `demo123`

### Configuration

The docker-compose stack uses `config-demo.yaml` which:
- Enables basic authentication (username: `demo`, password: `demo123`)
- Uses a null indexer (no Elasticsearch required for demo)
- Logs are output to stdout

To use with Elasticsearch, modify `config-demo.yaml` to include:
```yaml
elasticsearch:
  addresses:
    - "http://elasticsearch:9200"
  username: "elastic"
  password: "changeme"
```

## Architecture

### Components

- **cmd/app**: Main server application
- **internal/server**: HTTP and gRPC server implementation
- **internal/indexer**: Elasticsearch bulk indexer (with null implementation for demos)
- **internal/config**: Configuration management
- **internal/logger**: Structured logging
- **api/proto**: Protocol buffer definitions
- **client**: HTML/JS web client demo

### Protocol

The server accepts telemetry data via two protocols:

**HTTP Endpoint**: `POST /v1/telemetry` (Port 8080)
- Content-Type: `application/json`
- Optional Basic Authentication
- JSON payload matching protobuf schema
- Used by the web demo client

**gRPC Service**: `Collector.SendTelemetry` (Port 50051)
- Service defined in `api/proto/telemetry.proto`
- Binary protobuf encoding (smaller packet size)
- Optional basic authentication via metadata
- Recommended for mobile SDKs and high-frequency telemetry
- Package: `observability`
- Method: `SendTelemetry(TelemetryPacket) returns (Ack)`

### Data Models

**TelemetryPacket** contains:
- `metadata`: Client information (platform, IDs, versions, hardware)
- `metrics`: Performance metrics (CPU, memory, battery, network)
- `logs`: Log entries with levels, tags, messages, and context

See `api/proto/*.proto` for complete schema definitions.

## Development

### Local Development with Air (Hot Reload)

[Air](https://github.com/air-verse/air) provides live reload for Go applications during development. When you save a file, Air automatically rebuilds and restarts the application.

#### Installation

```bash
# Install Air
go install github.com/air-verse/air@latest
```

#### Running with Air

```bash
# Run with Air for hot reload
air

# Or specify a custom config
air -c .air.toml
```

Air will:
- Watch for changes in `.go`, `.yaml`, `.html` files
- Automatically rebuild the application
- Restart the server with the new binary
- Display build errors in the console

Configuration is in `.air.toml` with settings optimized for local development.

### Docker Development with Air

For a complete development environment with hot reload in Docker:

```bash
# Start the development stack with Air hot reload
docker-compose -f compose.dev.yaml up

# Or rebuild and start
docker-compose -f compose.dev.yaml up --build
```

This will:
- Start the telemetry server with Air hot reload
- Mount your local source code into the container
- Automatically rebuild and restart on code changes
- Start the web client on port 3000

The Docker setup uses `.air-docker.toml` with polling enabled (required for Docker volume mounts).

**Note**: The first build in Docker may take a minute. Subsequent rebuilds are much faster.

### Building

```bash
# Build the server
go build -o telemetry-server ./cmd/app

# Run with custom config
./telemetry-server  # loads config.yaml by default
```

### Running Tests

```bash
go test ./...
```

### Client Development

The web client is a static HTML/JS application:

```bash
cd client
python3 -m http.server 3000
```

See [client/README.md](client/README.md) for more details.

## Features

- **Dual Protocol**: HTTP and gRPC endpoints
- **Authentication**: Optional basic auth for both protocols
- **Rate Limiting**: Configurable per-client rate limiting
- **Batch Indexing**: Efficient Elasticsearch bulk indexing
- **Demo Mode**: Null indexer for testing without Elasticsearch
- **Health Check**: `/healthz` endpoint for monitoring

## Configuration

See `config.yaml` for available options:

- Server bind address and ports
- Basic authentication credentials
- Rate limiting settings
- Elasticsearch connection and indexing options
- Logging level