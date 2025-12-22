from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from typing import ClassVar as _ClassVar

DESCRIPTOR: _descriptor.FileDescriptor

class Platform(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PLATFORM_UNKNOWN: _ClassVar[Platform]
    WEB: _ClassVar[Platform]
    IOS: _ClassVar[Platform]
    ANDROID: _ClassVar[Platform]
    OTHER: _ClassVar[Platform]

class NetworkType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    NET_UNKNOWN: _ClassVar[NetworkType]
    NET_WIFI: _ClassVar[NetworkType]
    NET_CELLULAR_5G: _ClassVar[NetworkType]
    NET_CELLULAR_4G: _ClassVar[NetworkType]
    NET_CELLULAR_3G: _ClassVar[NetworkType]
    NET_CELLULAR_2G: _ClassVar[NetworkType]
    NET_WIRED: _ClassVar[NetworkType]
    NET_OFFLINE: _ClassVar[NetworkType]
PLATFORM_UNKNOWN: Platform
WEB: Platform
IOS: Platform
ANDROID: Platform
OTHER: Platform
NET_UNKNOWN: NetworkType
NET_WIFI: NetworkType
NET_CELLULAR_5G: NetworkType
NET_CELLULAR_4G: NetworkType
NET_CELLULAR_3G: NetworkType
NET_CELLULAR_2G: NetworkType
NET_WIRED: NetworkType
NET_OFFLINE: NetworkType
