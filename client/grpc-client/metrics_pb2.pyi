import common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DeviceHardwareInfo(_message.Message):
    __slots__ = ("physical_cores", "logical_cpus", "l1_cache_kb", "l2_cache_kb", "l3_cache_kb", "total_physical_memory")
    PHYSICAL_CORES_FIELD_NUMBER: _ClassVar[int]
    LOGICAL_CPUS_FIELD_NUMBER: _ClassVar[int]
    L1_CACHE_KB_FIELD_NUMBER: _ClassVar[int]
    L2_CACHE_KB_FIELD_NUMBER: _ClassVar[int]
    L3_CACHE_KB_FIELD_NUMBER: _ClassVar[int]
    TOTAL_PHYSICAL_MEMORY_FIELD_NUMBER: _ClassVar[int]
    physical_cores: int
    logical_cpus: int
    l1_cache_kb: int
    l2_cache_kb: int
    l3_cache_kb: int
    total_physical_memory: int
    def __init__(self, physical_cores: _Optional[int] = ..., logical_cpus: _Optional[int] = ..., l1_cache_kb: _Optional[int] = ..., l2_cache_kb: _Optional[int] = ..., l3_cache_kb: _Optional[int] = ..., total_physical_memory: _Optional[int] = ...) -> None: ...

class MetricBatch(_message.Message):
    __slots__ = ("points",)
    POINTS_FIELD_NUMBER: _ClassVar[int]
    points: _containers.RepeatedCompositeFieldContainer[MetricPoint]
    def __init__(self, points: _Optional[_Iterable[_Union[MetricPoint, _Mapping]]] = ...) -> None: ...

class MetricPoint(_message.Message):
    __slots__ = ("client_timestamp_ms", "network_type", "battery_level_percent", "cpu", "memory")
    CLIENT_TIMESTAMP_MS_FIELD_NUMBER: _ClassVar[int]
    NETWORK_TYPE_FIELD_NUMBER: _ClassVar[int]
    BATTERY_LEVEL_PERCENT_FIELD_NUMBER: _ClassVar[int]
    CPU_FIELD_NUMBER: _ClassVar[int]
    MEMORY_FIELD_NUMBER: _ClassVar[int]
    client_timestamp_ms: int
    network_type: _common_pb2.NetworkType
    battery_level_percent: float
    cpu: CpuDetail
    memory: MemoryDetail
    def __init__(self, client_timestamp_ms: _Optional[int] = ..., network_type: _Optional[_Union[_common_pb2.NetworkType, str]] = ..., battery_level_percent: _Optional[float] = ..., cpu: _Optional[_Union[CpuDetail, _Mapping]] = ..., memory: _Optional[_Union[MemoryDetail, _Mapping]] = ...) -> None: ...

class CpuDetail(_message.Message):
    __slots__ = ("total_usage_percent", "core_usage_percent")
    TOTAL_USAGE_PERCENT_FIELD_NUMBER: _ClassVar[int]
    CORE_USAGE_PERCENT_FIELD_NUMBER: _ClassVar[int]
    total_usage_percent: float
    core_usage_percent: _containers.RepeatedScalarFieldContainer[float]
    def __init__(self, total_usage_percent: _Optional[float] = ..., core_usage_percent: _Optional[_Iterable[float]] = ...) -> None: ...

class MemoryDetail(_message.Message):
    __slots__ = ("app_resident_bytes", "app_virtual_bytes", "system_free_bytes", "system_active_bytes", "system_inactive_bytes", "system_wired_bytes")
    APP_RESIDENT_BYTES_FIELD_NUMBER: _ClassVar[int]
    APP_VIRTUAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    SYSTEM_FREE_BYTES_FIELD_NUMBER: _ClassVar[int]
    SYSTEM_ACTIVE_BYTES_FIELD_NUMBER: _ClassVar[int]
    SYSTEM_INACTIVE_BYTES_FIELD_NUMBER: _ClassVar[int]
    SYSTEM_WIRED_BYTES_FIELD_NUMBER: _ClassVar[int]
    app_resident_bytes: int
    app_virtual_bytes: int
    system_free_bytes: int
    system_active_bytes: int
    system_inactive_bytes: int
    system_wired_bytes: int
    def __init__(self, app_resident_bytes: _Optional[int] = ..., app_virtual_bytes: _Optional[int] = ..., system_free_bytes: _Optional[int] = ..., system_active_bytes: _Optional[int] = ..., system_inactive_bytes: _Optional[int] = ..., system_wired_bytes: _Optional[int] = ...) -> None: ...
