# Python gRPC Client for tf-telemetry

This is a Python gRPC client that demonstrates how to send telemetry data (metrics and logs) to the tf-telemetry server using the gRPC protocol.

## Features

- **Native gRPC**: Uses binary protobuf encoding for efficient data transfer
- **Complete Example**: Demonstrates sending both metrics and logs
- **Basic Authentication**: Optional basic auth support via gRPC metadata
- **Sample Data**: Generates realistic sample telemetry data including:
  - CPU and memory metrics
  - Battery level
  - Log entries with different severity levels (INFO, WARN, ERROR)
  - Device hardware information
  - Network type information

## Prerequisites

- Python 3.7 or higher
- pip (Python package manager)

## Installation

1. Install the required Python packages:

```bash
pip install -r requirements.txt
```

The requirements include:
- `grpcio` - gRPC framework for Python
- `grpcio-tools` - Tools for generating Python code from .proto files
- `protobuf` - Protocol buffer runtime library

## Usage

### Basic Usage (No Authentication)

Send telemetry to a server running on localhost:

```bash
python client.py
```

This connects to `localhost:50051` by default.

### With Custom Server Address

```bash
python client.py --server <host>:<port>
```

Example:
```bash
python client.py --server telemetry.example.com:50051
```

### With Basic Authentication

If the server has basic authentication enabled:

```bash
python client.py --username demo --password demo123
```

### Complete Example

```bash
python client.py --server localhost:50051 --username demo --password demo123
```

## Running with Docker Compose

To test against a local server:

1. Start the telemetry server from the repository root:
   ```bash
   docker compose up
   ```

2. In another terminal, run the Python client:
   ```bash
   cd client/grpc-client
   python client.py --username demo --password demo123
   ```

The demo server runs with basic auth enabled (username: `demo`, password: `demo123`).

## Generated Code

The gRPC client uses Python code generated from the protobuf definitions in `api/proto/`:

- `telemetry_pb2.py` - Message classes for TelemetryPacket, Ack, etc.
- `telemetry_pb2_grpc.py` - Collector service stub
- `common_pb2.py` - Common enums (Platform, NetworkType)
- `metrics_pb2.py` - Metrics-related messages
- `logs_pb2.py` - Log-related messages

### Regenerating the Protobuf Code

If the proto files change, regenerate the Python code:

```bash
python -m grpc_tools.protoc \
  -I../../api/proto \
  --python_out=. \
  --grpc_python_out=. \
  ../../api/proto/common.proto \
  ../../api/proto/metrics.proto \
  ../../api/proto/logs.proto \
  ../../api/proto/telemetry.proto
```

## Code Structure

### Creating Telemetry Data

The client demonstrates how to create a complete telemetry packet:

```python
# Create a telemetry packet
packet = telemetry_pb2.TelemetryPacket()
packet.schema_version = 1

# Add metadata
metadata = packet.metadata
metadata.platform = common_pb2.WEB
metadata.installation_id = generate_uuid_v7_bytes()
metadata.host_app_name = "My App"

# Add metrics
metrics_batch = packet.metrics
point = metrics_batch.points.add()
point.client_timestamp_ms = int(time.time() * 1000)
point.battery_level_percent = 85.5

# Add logs
log_batch = packet.logs
log_entry = log_batch.entries.add()
log_entry.level = logs_pb2.INFO
log_entry.message = "Application started"
```

### Sending via gRPC

```python
# Create channel and stub
channel = grpc.insecure_channel("localhost:50051")
stub = telemetry_pb2_grpc.CollectorStub(channel)

# Send telemetry
response = stub.SendTelemetry(packet)
print(f"Success: {response.success}")
```

### With Authentication

```python
# Create auth metadata
credentials = f"{username}:{password}"
encoded = base64.b64encode(credentials.encode()).decode()
metadata = [("authorization", f"Basic {encoded}")]

# Send with metadata
response = stub.SendTelemetry(packet, metadata=metadata)
```

## Telemetry Data Structure

### Metrics

The client sends sample metrics including:
- **Client timestamp** - When the metric was captured
- **Network type** - WiFi, Cellular, etc.
- **Battery level** - Battery percentage
- **CPU usage** - Total and per-core usage
- **Memory usage** - App and system memory statistics

### Logs

The client sends sample log entries with:
- **Client timestamp** - When the log was generated
- **Network type** - Current network connection
- **Log level** - DEBUG, INFO, WARN, ERROR, FATAL
- **Tag** - Category/component name
- **Message** - Log message
- **Context** - Key-value pairs with additional information
- **Stack trace** - For error logs

## Advantages of gRPC

Compared to the HTTP/JSON client:

- **Smaller packet size**: Binary protobuf is more compact than JSON
- **Better performance**: Lower overhead for serialization/deserialization
- **Type safety**: Strongly typed messages defined in proto files
- **Streaming support**: Can be extended to support bi-directional streaming
- **Native mobile support**: gRPC works well with iOS and Android SDKs

## Troubleshooting

### Connection Refused

If you get a connection error:
- Make sure the telemetry server is running
- Verify the server address and port (default: 50051)
- Check firewall settings

### Authentication Failed

If you get authentication errors:
- Verify the username and password
- Check if basic auth is enabled on the server
- Look at the server logs for authentication details

### Import Errors

If you get import errors for the generated files:
- Make sure you've run the protoc command to generate the Python files
- Check that all `.py` files exist in the grpc-client directory
- Verify you're running the client from the correct directory

## See Also

- [HTTP Client](../http-client/README.md) - Browser-based HTTP/JSON client
- [API Documentation](../../api/proto/README.md) - Protocol buffer definitions
- [Main README](../../README.md) - tf-telemetry server documentation
