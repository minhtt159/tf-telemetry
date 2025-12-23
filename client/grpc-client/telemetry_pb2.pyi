import common_pb2 as _common_pb2
import metrics_pb2 as _metrics_pb2
import logs_pb2 as _logs_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Ack(_message.Message):
    __slots__ = ("success", "message")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    message: str
    def __init__(self, success: bool = ..., message: _Optional[str] = ...) -> None: ...

class TelemetryPacket(_message.Message):
    __slots__ = ("schema_version", "metadata", "metrics", "logs")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    LOGS_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    metadata: ClientMetadata
    metrics: _metrics_pb2.MetricBatch
    logs: _logs_pb2.LogBatch
    def __init__(self, schema_version: _Optional[int] = ..., metadata: _Optional[_Union[ClientMetadata, _Mapping]] = ..., metrics: _Optional[_Union[_metrics_pb2.MetricBatch, _Mapping]] = ..., logs: _Optional[_Union[_logs_pb2.LogBatch, _Mapping]] = ...) -> None: ...

class ClientMetadata(_message.Message):
    __slots__ = ("platform", "installation_id", "journey_id", "sdk_version_packed", "host_app_version", "host_app_name", "device_hardware")
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    INSTALLATION_ID_FIELD_NUMBER: _ClassVar[int]
    JOURNEY_ID_FIELD_NUMBER: _ClassVar[int]
    SDK_VERSION_PACKED_FIELD_NUMBER: _ClassVar[int]
    HOST_APP_VERSION_FIELD_NUMBER: _ClassVar[int]
    HOST_APP_NAME_FIELD_NUMBER: _ClassVar[int]
    DEVICE_HARDWARE_FIELD_NUMBER: _ClassVar[int]
    platform: _common_pb2.Platform
    installation_id: bytes
    journey_id: bytes
    sdk_version_packed: int
    host_app_version: str
    host_app_name: str
    device_hardware: _metrics_pb2.DeviceHardwareInfo
    def __init__(self, platform: _Optional[_Union[_common_pb2.Platform, str]] = ..., installation_id: _Optional[bytes] = ..., journey_id: _Optional[bytes] = ..., sdk_version_packed: _Optional[int] = ..., host_app_version: _Optional[str] = ..., host_app_name: _Optional[str] = ..., device_hardware: _Optional[_Union[_metrics_pb2.DeviceHardwareInfo, _Mapping]] = ...) -> None: ...
