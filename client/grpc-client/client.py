#!/usr/bin/env python3
"""
Python gRPC client for tf-telemetry server.

This example demonstrates how to send telemetry data (metrics and logs) 
to the tf-telemetry server using the gRPC protocol.
"""

import time
import uuid
import base64
import grpc
from datetime import datetime

# Import the generated protobuf classes
import telemetry_pb2
import telemetry_pb2_grpc
import common_pb2
import metrics_pb2
import logs_pb2


def generate_uuid_v7_bytes():
    """
    Generate a UUID v7 (time-ordered) as bytes for installation/journey IDs.
    
    UUID v7 format (RFC 9562):
    - 48 bits: Unix timestamp in milliseconds
    - 12 bits: sub-millisecond precision / random
    - 2 bits: version (0b111 for v7)
    - 62 bits: random
    """
    import struct
    import random
    
    # Get current timestamp in milliseconds (48 bits)
    timestamp_ms = int(time.time() * 1000)
    
    # Generate random bytes for the rest
    rand_bytes = random.randbytes(10)
    
    # Construct UUID v7:
    # - First 6 bytes: timestamp_ms (48 bits)
    # - Next 2 bytes: random + version bits
    # - Last 8 bytes: random
    
    # Pack timestamp (48 bits = 6 bytes)
    time_bytes = timestamp_ms.to_bytes(6, byteorder='big')
    
    # Set version to 7 (4 bits) and variant to RFC4122 (2 bits)
    # Version 7 = 0111, Variant = 10
    rand_a = rand_bytes[0:2]
    rand_b = rand_bytes[2:10]
    
    # Apply version (0111 = 7) to bits 48-51
    version_byte_high = (rand_a[0] & 0x0F) | 0x70  # 0111xxxx
    # Apply variant (10) to bits 64-65
    variant_byte_high = (rand_b[0] & 0x3F) | 0x80  # 10xxxxxx
    
    # Combine all parts
    uuid_bytes = (
        time_bytes + 
        bytes([version_byte_high, rand_a[1]]) +
        bytes([variant_byte_high]) +
        rand_b[1:]
    )
    
    return uuid_bytes


def create_sample_metrics():
    """Create a sample metrics batch with CPU, memory, and battery data."""
    metrics_batch = metrics_pb2.MetricBatch()
    
    # Create a metric point
    point = metrics_batch.points.add()
    point.client_timestamp_ms = int(time.time() * 1000)
    point.network_type = common_pb2.NET_WIFI
    point.battery_level_percent = 85.5
    
    # CPU details
    point.cpu.total_usage_percent = 45.2
    point.cpu.core_usage_percent.extend([42.1, 48.3, 43.5, 46.8])
    
    # Memory details
    point.memory.app_resident_bytes = 256 * 1024 * 1024  # 256 MB
    point.memory.app_virtual_bytes = 512 * 1024 * 1024   # 512 MB
    point.memory.system_free_bytes = 2 * 1024 * 1024 * 1024  # 2 GB
    point.memory.system_active_bytes = 4 * 1024 * 1024 * 1024  # 4 GB
    
    return metrics_batch


def create_sample_logs():
    """Create a sample log batch with different severity levels."""
    log_batch = logs_pb2.LogBatch()
    
    # INFO log
    info_log = log_batch.entries.add()
    info_log.client_timestamp_ms = int(time.time() * 1000)
    info_log.network_type = common_pb2.NET_WIFI
    info_log.level = logs_pb2.INFO
    info_log.tag = "application"
    info_log.message = "Application started successfully"
    info_log.context["user_agent"] = "Python gRPC Client"
    info_log.context["version"] = "1.0.0"
    
    # WARN log
    warn_log = log_batch.entries.add()
    warn_log.client_timestamp_ms = int(time.time() * 1000)
    warn_log.network_type = common_pb2.NET_WIFI
    warn_log.level = logs_pb2.WARN
    warn_log.tag = "performance"
    warn_log.message = "High memory usage detected"
    warn_log.context["memory_usage"] = "85%"
    
    # ERROR log with stack trace
    error_log = log_batch.entries.add()
    error_log.client_timestamp_ms = int(time.time() * 1000)
    error_log.network_type = common_pb2.NET_WIFI
    error_log.level = logs_pb2.ERROR
    error_log.tag = "network"
    error_log.message = "Failed to connect to external service"
    error_log.context["endpoint"] = "https://api.example.com"
    error_log.context["status_code"] = "503"
    error_log.stack_trace = "Traceback (most recent call last):\n  File \"client.py\", line 42, in connect\n    raise ConnectionError()"
    
    return log_batch


def create_telemetry_packet():
    """Create a complete telemetry packet with metadata, metrics, and logs."""
    packet = telemetry_pb2.TelemetryPacket()
    
    # Schema version
    packet.schema_version = 1
    
    # Client metadata
    metadata = packet.metadata
    metadata.platform = common_pb2.WEB
    metadata.installation_id = generate_uuid_v7_bytes()
    metadata.journey_id = generate_uuid_v7_bytes()
    metadata.sdk_version_packed = 10000  # e.g., version 1.0.0
    metadata.host_app_version = "2.3.1"
    metadata.host_app_name = "Python gRPC Demo Client"
    
    # Device hardware info
    metadata.device_hardware.physical_cores = 4
    metadata.device_hardware.logical_cpus = 8
    metadata.device_hardware.l1_cache_kb = 256
    metadata.device_hardware.l2_cache_kb = 1024
    metadata.device_hardware.l3_cache_kb = 8192
    metadata.device_hardware.total_physical_memory = 16 * 1024 * 1024 * 1024  # 16 GB
    
    # Add metrics and logs
    packet.metrics.CopyFrom(create_sample_metrics())
    packet.logs.CopyFrom(create_sample_logs())
    
    return packet


def send_telemetry(server_address, username=None, password=None):
    """
    Send telemetry data to the gRPC server.
    
    Args:
        server_address: Server address in format "host:port"
        username: Optional basic auth username
        password: Optional basic auth password
    """
    print(f"Connecting to gRPC server at {server_address}...")
    
    # Create a gRPC channel
    channel = grpc.insecure_channel(server_address)
    
    # Create a stub (client)
    stub = telemetry_pb2_grpc.CollectorStub(channel)
    
    # Create telemetry packet
    packet = create_telemetry_packet()
    
    print(f"Sending telemetry packet:")
    print(f"  - Schema version: {packet.schema_version}")
    print(f"  - Platform: {common_pb2.Platform.Name(packet.metadata.platform)}")
    print(f"  - App: {packet.metadata.host_app_name} v{packet.metadata.host_app_version}")
    print(f"  - Metrics points: {len(packet.metrics.points)}")
    print(f"  - Log entries: {len(packet.logs.entries)}")
    
    # Prepare metadata for basic auth if credentials provided
    metadata = []
    if username and password:
        credentials = f"{username}:{password}"
        encoded_credentials = base64.b64encode(credentials.encode()).decode()
        metadata.append(("authorization", f"Basic {encoded_credentials}"))
        print(f"  - Using basic auth with username: {username}")
    
    try:
        # Send the telemetry
        response = stub.SendTelemetry(packet, metadata=metadata)
        
        print(f"\n✓ Success!")
        print(f"  - Success: {response.success}")
        print(f"  - Message: {response.message}")
        
    except grpc.RpcError as e:
        print(f"\n✗ Error!")
        print(f"  - Status: {e.code()}")
        print(f"  - Details: {e.details()}")
        raise
    
    finally:
        channel.close()


def main():
    """Main entry point for the gRPC client."""
    import argparse
    
    parser = argparse.ArgumentParser(
        description="Python gRPC client for tf-telemetry server"
    )
    parser.add_argument(
        "--server",
        default="localhost:50051",
        help="gRPC server address (default: localhost:50051)"
    )
    parser.add_argument(
        "--username",
        help="Basic auth username (optional)"
    )
    parser.add_argument(
        "--password",
        help="Basic auth password (optional)"
    )
    
    args = parser.parse_args()
    
    try:
        send_telemetry(args.server, args.username, args.password)
    except Exception as e:
        print(f"\nFailed to send telemetry: {e}")
        exit(1)


if __name__ == "__main__":
    main()
