import common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class LogLevel(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DEBUG: _ClassVar[LogLevel]
    INFO: _ClassVar[LogLevel]
    WARN: _ClassVar[LogLevel]
    ERROR: _ClassVar[LogLevel]
    FATAL: _ClassVar[LogLevel]
DEBUG: LogLevel
INFO: LogLevel
WARN: LogLevel
ERROR: LogLevel
FATAL: LogLevel

class LogBatch(_message.Message):
    __slots__ = ("entries",)
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[LogEntry]
    def __init__(self, entries: _Optional[_Iterable[_Union[LogEntry, _Mapping]]] = ...) -> None: ...

class LogEntry(_message.Message):
    __slots__ = ("client_timestamp_ms", "network_type", "level", "tag", "message", "context", "stack_trace")
    class ContextEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CLIENT_TIMESTAMP_MS_FIELD_NUMBER: _ClassVar[int]
    NETWORK_TYPE_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    STACK_TRACE_FIELD_NUMBER: _ClassVar[int]
    client_timestamp_ms: int
    network_type: _common_pb2.NetworkType
    level: LogLevel
    tag: str
    message: str
    context: _containers.ScalarMap[str, str]
    stack_trace: str
    def __init__(self, client_timestamp_ms: _Optional[int] = ..., network_type: _Optional[_Union[_common_pb2.NetworkType, str]] = ..., level: _Optional[_Union[LogLevel, str]] = ..., tag: _Optional[str] = ..., message: _Optional[str] = ..., context: _Optional[_Mapping[str, str]] = ..., stack_trace: _Optional[str] = ...) -> None: ...
