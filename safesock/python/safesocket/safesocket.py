#!/usr/bin/env python
# coding:utf-8

"""
ESSENTIAL PROCESS:
Python wrapper for libsafesocket. Provides native access to the safe-socket ecosystem, 
allowing high-performance, resilient network connections across multiple protocols.

DATA FLOW:
Loads shared library -> Creates handles via CGO bridge -> Manages socket state and 
data transmission through ctypes buffers.

KEY PARAMETERS:
- profile_name: The safe-socket profile to use (e.g., 'tcp', 'udp', 'shm').
- address: Target destination (IP:Port or FilePath).
- config: Advanced SocketConfig for timeouts and identity.
"""

from ctypes import CDLL as ctypesCDLL, POINTER as ctypesPOINTER, c_int32, c_char_p, c_int, c_double, c_ubyte, memmove as ctypesMemmove
from os import path as osPath, getenv as osGetenv
from sys import platform as sysPlatform
from typing import Optional, List

# ### LIBRARY LOADING ###

# Find the shared library
_base_path = osPath.dirname(osPath.abspath(__file__))
_search_paths = [
    _base_path,
    osPath.join(_base_path, ".."),
    osPath.join(_base_path, "../../libsafesocket"),
    osPath.join(_base_path, "../../../safesock/libsafesocket"),
]

_lib_name = "libsafesocket"
_lib_ext = ".so"
if sysPlatform == "darwin":
    _lib_ext = ".dylib"
elif sysPlatform == "win32":
    _lib_ext = ".dll"

_lib_path = None
for path in _search_paths:
    full_path = osPath.join(path, f"{_lib_name}{_lib_ext}")
    if osPath.exists(full_path):
        _lib_path = full_path
        break

if not _lib_path:
    _lib_path = osGetenv("LIBSAFESOCKET_PATH")
    if not _lib_path or not osPath.exists(_lib_path):
         raise FileNotFoundError(f"SafeSocket (Python): Shared library {_lib_name}{_lib_ext} not found. Please run 'make build-lib' first.")

lib = ctypesCDLL(_lib_path)

# Signatures
lib.SafeSocket_Create.argtypes = [c_char_p, c_char_p, c_char_p, c_char_p, c_int]
lib.SafeSocket_Create.restype = c_int32

lib.SafeSocket_CreateExtended.argtypes = [c_char_p, c_char_p, c_char_p, c_char_p, c_int, c_int, c_int, c_int]
lib.SafeSocket_CreateExtended.restype = c_int32

lib.SafeSocket_Open.argtypes = [c_int32]
lib.SafeSocket_Open.restype = c_int32

lib.SafeSocket_Close.argtypes = [c_int32]
lib.SafeSocket_Close.restype = c_int32

lib.SafeSocket_Send.argtypes = [c_int32, ctypesPOINTER(c_ubyte), c_int]
lib.SafeSocket_Send.restype = c_int32

lib.SafeSocket_Receive.argtypes = [c_int32, ctypesPOINTER(c_ubyte), c_int]
lib.SafeSocket_Receive.restype = c_int32

lib.SafeSocket_Listen.argtypes = [c_int32]
lib.SafeSocket_Listen.restype = c_int32

lib.SafeSocket_Accept.argtypes = [c_int32]
lib.SafeSocket_Accept.restype = c_int32

lib.SafeSocket_SetDeadline.argtypes = [c_int32, c_double]
lib.SafeSocket_SetDeadline.restype = c_int32

lib.SafeSocket_SetIdleTimeout.argtypes = [c_int32, c_double]
lib.SafeSocket_SetIdleTimeout.restype = c_int32

lib.SafeSocket_GetSocketError.argtypes = []
lib.SafeSocket_GetSocketError.restype = c_char_p

# ### ERROR HANDLING ###

class SafeSocketError(Exception):
    """Base exception for SafeSocket operations."""
    pass

# -----------------------------------------------------------------------------------------------

def _get_last_error() -> str:
    err_ptr = lib.SafeSocket_GetSocketError()
    if err_ptr:
        return err_ptr.decode('utf-8')
    return "Unknown error"

# ### MODELS ###

class SocketConfig:
    """
    SocketConfig mirrors the Go models.SocketConfig struct.
    It allows full customization of responsiveness and identity.
    """
    def __init__(self, public_ip: str = "", deadline_ms: int = 0, heartbeat_interval_ms: int = 0, handshake_timeout_ms: int = 0) -> None:
        self.public_ip = public_ip
        self.deadline_ms = deadline_ms
        self.heartbeat_interval_ms = heartbeat_interval_ms
        self.handshake_timeout_ms = handshake_timeout_ms

# ### CORE CLASSES ###

class SafeSocketConnection:
    """Represents an active connection returned by Accept()."""
    
    Name = "SafeSocketConnection"

    def __init__(self, handle: int) -> None:
        self.handle = handle
        self._closed = False

    # -----------------------------------------------------------------------------------------------

    def send(self, data: bytes) -> int:
        data_len = len(data)
        data_ptr = (c_ubyte * data_len).from_buffer_copy(data)
        n = lib.SafeSocket_Send(self.handle, data_ptr, data_len)
        if n == -1:
            raise SafeSocketError(f"SafeSocket (Python): Send failed - {_get_last_error()}")
        return n

    # -----------------------------------------------------------------------------------------------

    def receive(self, max_length: int = 65535) -> bytes:
        buffer = (c_ubyte * max_length)()
        n = lib.SafeSocket_Receive(self.handle, buffer, max_length)
        if n == -1:
            raise SafeSocketError(f"SafeSocket (Python): Receive failed - {_get_last_error()}")
        return bytes(buffer[:n])

    # -----------------------------------------------------------------------------------------------

    def close(self) -> None:
        if not self._closed:
            if lib.SafeSocket_Close(self.handle) == -1:
                self._closed = True
                raise SafeSocketError(f"SafeSocket (Python): Close failed - {_get_last_error()}")
            self._closed = True

    # -----------------------------------------------------------------------------------------------

    def set_deadline(self, seconds: float) -> None:
        if lib.SafeSocket_SetDeadline(self.handle, c_double(seconds)) == -1:
            raise SafeSocketError(f"SafeSocket (Python): SetDeadline failed - {_get_last_error()}")

    # -----------------------------------------------------------------------------------------------

    def set_idle_timeout(self, seconds: float) -> None:
        if lib.SafeSocket_SetIdleTimeout(self.handle, c_double(seconds)) == -1:
            raise SafeSocketError(f"SafeSocket (Python): SetIdleTimeout failed - {_get_last_error()}")

    # -----------------------------------------------------------------------------------------------

    def __enter__(self) -> 'SafeSocketConnection':
        return self

    # -----------------------------------------------------------------------------------------------

    def __exit__(self, exc_type, exc_val, exc_tb) -> None:
        self.close()

# -----------------------------------------------------------------------------------------------

class SafeSocket:
    """High-level Socket manager for Clients and Servers."""

    Name = "SafeSocket"

    def __init__(self, profile_name: str, address: str, config: Optional[SocketConfig] = None, socket_type: str = "client", auto_connect: bool = False) -> None:
        if config is None:
            config = SocketConfig()
        
        self.handle = lib.SafeSocket_CreateExtended(
            profile_name.encode('utf-8'),
            address.encode('utf-8'),
            config.public_ip.encode('utf-8'),
            socket_type.encode('utf-8'),
            config.handshake_timeout_ms,
            config.deadline_ms,
            config.heartbeat_interval_ms,
            1 if auto_connect else 0
        )
        if self.handle == -1:
            raise SafeSocketError(f"SafeSocket (Python): Initialization failed - {_get_last_error()}")
        self._closed = False

    # -----------------------------------------------------------------------------------------------

    def open(self) -> None:
        if lib.SafeSocket_Open(self.handle) == -1:
            raise SafeSocketError(f"SafeSocket (Python): Open failed - {_get_last_error()}")

    # -----------------------------------------------------------------------------------------------

    def close(self) -> None:
        if not self._closed:
            if lib.SafeSocket_Close(self.handle) == -1:
                self._closed = True
                raise SafeSocketError(f"SafeSocket (Python): Close failed - {_get_last_error()}")
            self._closed = True

    # -----------------------------------------------------------------------------------------------

    def send(self, data: bytes) -> None:
        data_len = len(data)
        data_ptr = (c_ubyte * data_len).from_buffer_copy(data)
        if lib.SafeSocket_Send(self.handle, data_ptr, data_len) == -1:
            raise SafeSocketError(f"SafeSocket (Python): Send failed - {_get_last_error()}")

    # -----------------------------------------------------------------------------------------------

    def receive(self, max_length: int = 65535) -> bytes:
        buffer = (c_ubyte * max_length)()
        n = lib.SafeSocket_Receive(self.handle, buffer, max_length)
        if n == -1:
            raise SafeSocketError(f"SafeSocket (Python): Receive failed - {_get_last_error()}")
        return bytes(buffer[:n])

    # -----------------------------------------------------------------------------------------------

    def listen(self) -> None:
        if lib.SafeSocket_Listen(self.handle) == -1:
            raise SafeSocketError(f"SafeSocket (Python): Listen failed - {_get_last_error()}")

    # -----------------------------------------------------------------------------------------------

    def accept(self) -> SafeSocketConnection:
        conn_handle = lib.SafeSocket_Accept(self.handle)
        if conn_handle == -1:
            raise SafeSocketError(f"SafeSocket (Python): Accept failed - {_get_last_error()}")
        return SafeSocketConnection(conn_handle)

    # -----------------------------------------------------------------------------------------------

    def set_deadline(self, seconds: float) -> None:
        if lib.SafeSocket_SetDeadline(self.handle, c_double(seconds)) == -1:
            raise SafeSocketError(f"SafeSocket (Python): SetDeadline failed - {_get_last_error()}")

    # -----------------------------------------------------------------------------------------------

    def set_idle_timeout(self, seconds: float) -> None:
        if lib.SafeSocket_SetIdleTimeout(self.handle, c_double(seconds)) == -1:
            raise SafeSocketError(f"SafeSocket (Python): SetIdleTimeout failed - {_get_last_error()}")

    # -----------------------------------------------------------------------------------------------

    def __enter__(self) -> 'SafeSocket':
        return self

    # -----------------------------------------------------------------------------------------------

    def __exit__(self, exc_type, exc_val, exc_tb) -> None:
        self.close()

# ### FACTORIES ###

def create(profile_name: str, address: str, public_ip: str = "", socket_type: str = "client", auto_connect: bool = False) -> SafeSocket:
    """Simplified entry point matching Go safesocket.Create() signature."""
    config = SocketConfig(public_ip=public_ip)
    return SafeSocket(profile_name, address, config, socket_type, auto_connect)

# -----------------------------------------------------------------------------------------------

def create_with_config(profile_name: str, address: str, config: SocketConfig, socket_type: str = "client", auto_connect: bool = False) -> SafeSocket:
    """Advanced entry point matching Go safesocket.CreateWithConfig() signature."""
    return SafeSocket(profile_name, address, config, socket_type, auto_connect)
