# Telemetry Client Examples

This directory contains example client implementations for sending telemetry data to the tf-telemetry server.

## Available Clients

### 1. HTTP Client (`http-client/`)

A browser-based web client using HTML, JavaScript, and HTTP/JSON protocol.

**Features:**
- Interactive web interface
- Send metrics and logs via HTTP REST API
- Basic authentication support
- Offline queue with automatic retry
- Docker-based deployment with nginx

**Best for:**
- Web applications
- Browser-based demos
- Quick testing and visualization

[→ See HTTP Client README](http-client/README.md)

### 2. gRPC Client (`grpc-client/`)

A Python command-line client using gRPC and binary protobuf protocol.

**Features:**
- Native gRPC with binary protobuf encoding
- Complete metrics and logs example
- Basic authentication support via gRPC metadata
- Command-line interface

**Best for:**
- Server-side applications
- Mobile SDK examples (iOS, Android)
- High-performance telemetry
- Production integrations

[→ See gRPC Client README](grpc-client/README.md)

## Quick Start

### Running the HTTP Client

```bash
# From repository root
docker compose up

# Open browser to http://localhost:3000
```

### Running the gRPC Client

```bash
# From repository root, first start the server
docker compose up -d

# Install dependencies and run the Python client
cd client/grpc-client
pip install -r requirements.txt
python client.py --username demo --password demo123
```

## Protocol Comparison

| Feature | HTTP Client | gRPC Client |
|---------|-------------|-------------|
| **Protocol** | HTTP/1.1 + JSON | HTTP/2 + Protobuf |
| **Encoding** | Text (JSON) | Binary (Protobuf) |
| **Packet Size** | Larger | Smaller (~50-70% reduction) |
| **Performance** | Good | Excellent |
| **Browser Support** | Native | Requires grpc-web proxy |
| **Mobile Support** | Standard HTTP libs | Native gRPC SDKs |
| **Streaming** | Not supported | Bi-directional streaming |
| **Type Safety** | Runtime validation | Compile-time validation |

## When to Use Which

### Use HTTP Client when:
- Building web applications
- Need browser compatibility
- Want simple REST API integration
- Developing quick prototypes or demos
- Working with JSON-based tools

### Use gRPC Client when:
- Building mobile applications (iOS/Android)
- Need high performance and low bandwidth
- Have high-frequency telemetry (many requests/sec)
- Want strong typing and code generation
- Building server-to-server integrations

## Server Configuration

The telemetry server supports both protocols simultaneously:

- **HTTP endpoint**: Port 8080 at `/v1/telemetry`
- **gRPC service**: Port 50051, service `observability.Collector`

Both endpoints support the same authentication and rate limiting configuration.

See the [main README](../README.md) for server configuration details.
