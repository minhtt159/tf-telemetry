# tf-telemetry

A telemetry collection server for mobile and web applications, with support for metrics and logs ingestion via gRPC and HTTP.

## Quick Start

```bash
# Start with hot reload (development mode - default)
docker compose up

# Start in production mode
docker compose --profile prod up
```

This will start:

- **Telemetry Server** on ports 8080 (HTTP) and 50051 (gRPC)
- **Web Client Demo** on port 3000

Open your browser to: **http://localhost:3000**

Default configuration:

- **Basic Auth**: username `demo`, password `demo123`
- **Null Indexer**: Enabled by default (no Elasticsearch required for demo)

## Build and Test

```bash
make build     # builds ./bin/tf-telemetry
make test      # runs go test ./...
make lint      # runs golangci-lint
make dev       # starts development server with Air
make swagger   # generates Swagger documentation
make clean     # removes build artifacts
```

## Configuration

Edit `config.yaml` to configure:

- Server bind address and ports
- Basic authentication credentials
- Rate limiting settings
- Elasticsearch connection (empty addresses = null indexer)
- Logging level

To use with Elasticsearch, update `config.yaml`:

```yaml
elasticsearch:
  addresses:
    - "http://elasticsearch:9200"
  username: "elastic"
  password: "changeme"
```

## API Endpoints

### HTTP

- `POST /v1/telemetry` - Submit telemetry data (JSON)
- `GET /healthz` - Health check
- `GET /swagger/*` - Swagger UI documentation

### gRPC

- Service: `observability.Collector`
- Method: `SendTelemetry(TelemetryPacket) returns (Ack)`
- Port: 50051

## Architecture

```
cmd/app             - Main server application
internal/service    - Collector service implementation
internal/httpserver - HTTP server wiring
internal/grpcserver - gRPC server wiring
internal/indexer    - Elasticsearch bulk indexer (with null implementation)
internal/ingest     - Telemetry packet processing
internal/config     - Configuration management
internal/logger     - Structured logging
api/proto           - Protocol buffer definitions
client/http-client  - Browser-based HTML/JS demo client
client/grpc-client  - Python gRPC client example
```

## Development

### Local Development with Air

```bash
# Install Air
go install github.com/air-verse/air@latest

# Run with hot reload
air
```

### Docker Development

The default `docker compose up` uses development mode with Air hot reload.
Source code is mounted into the container and changes are automatically detected.

Configuration:

- `.air.toml` - Local Air configuration
- `.air-docker.toml` - Docker Air configuration (uses polling)
- `build/Dockerfile.dev` - Development Dockerfile

## Client Examples

See [client/README.md](client/README.md) for HTTP and gRPC client examples.

## Features

- **Dual Protocol**: HTTP and gRPC endpoints
- **Authentication**: Optional basic auth
- **Rate Limiting**: Configurable per-client
- **Batch Indexing**: Efficient Elasticsearch bulk indexing
- **Demo Mode**: Null indexer for testing without Elasticsearch
- **Swagger UI**: API documentation at `/swagger/`
- **Hot Reload**: Air-based development workflow
