# Telemetry Client Demo

This is a simple HTML/JavaScript client application that demonstrates how to send telemetry data (metrics and logs) to the tf-telemetry server.

## Features

- **Send Metrics**: Generate and send sample performance metrics (CPU, memory, battery)
- **Send Logs**: Generate and send sample log entries with different severity levels
- **Basic Authentication**: Support for basic auth when the server has it enabled
- **Offline Support**: Automatically queues telemetry packets in localStorage when the server is unavailable
- **Automatic Retry**: Retries failed packets every 30 seconds automatically
- **Sample Payloads**: View the structure of metrics and logs payloads

## Architecture

- **Tech Stack**: HTML, JavaScript, nginx, Docker
- **Storage**: localStorage for offline packet queuing
- **Protocol**: HTTP/JSON to the telemetry server's `/v1/telemetry` endpoint

## Usage

### Running with Docker Compose

From the repository root:

```bash
docker compose up
```

This will start:

- **Telemetry Server** on ports 8080 (HTTP) and 50051 (gRPC)
- **Web Client** on port 3000

Then open your browser to: http://localhost:3000

### Configuration

The demo comes with these default settings:

- **Server URL**: `http://localhost:8080`
- **Username**: `demo` (if basic auth is enabled)
- **Password**: `demo123` (if basic auth is enabled)

You can modify these values in the web interface before sending telemetry data.

## How It Works

### Sending Data

1. Click "Send Metrics" to generate and send a metrics payload
2. Click "Send Logs" to generate and send a logs payload
3. Click "Send Both" to send metrics and logs together

### Offline Support

If the server is unavailable:

- Packets are automatically queued in browser localStorage
- The queue status shows how many packets are waiting
- Failed packets are automatically retried every 30 seconds
- You can manually retry by clicking "Retry Failed Packets"
- Maximum queue size: 100 packets (oldest are removed when full)

### Sample Data

The JavaScript library generates realistic sample data:

**Metrics include:**

- Client timestamp
- Network type (WiFi, Cellular, Offline)
- Battery level percentage
- CPU usage (total and per-core)
- Memory usage (app and system)
- Device hardware info

**Logs include:**

- Client timestamp
- Log level (DEBUG, INFO, WARN, ERROR, FATAL)
- Tag/category
- Message
- Context (user agent, URL, screen dimensions)
- Stack trace (for ERROR level)

### Client Identifiers

- **Installation ID**: Stored in localStorage, persists across sessions
- **Journey ID**: Stored in sessionStorage, unique per browser tab/session

## Files

- `index.html` - Main HTML interface
- `telemetry.js` - JavaScript library for generating and sending telemetry data
- `nginx.conf` - nginx web server configuration
- `Dockerfile` - Container image for the web client

## Protocol

The client currently uses HTTP/JSON for sending telemetry data:

### HTTP/JSON Endpoint

```
POST /v1/telemetry HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Authorization: Basic <base64-encoded-credentials>

{
  "schema_version": 1,
  "metadata": { ... },
  "metrics": { ... },
  "logs": { ... }
}
```

Response:

```json
{ "status": "accepted" }
```

### gRPC Support

The server also supports native gRPC on port 50051 for more efficient binary protocol:

```
Service: observability.Collector
Method: SendTelemetry(TelemetryPacket) returns (Ack)
Protocol: gRPC (protobuf binary)
Port: 50051
```

**Benefits of gRPC:**

- Smaller packet size (binary protobuf vs JSON)
- Better performance for high-frequency telemetry
- Native support in mobile SDKs (iOS, Android)

**Note:** gRPC-Web support for browsers requires additional infrastructure (grpc-web proxy or envoy). The current web demo uses HTTP/JSON for simplicity.
