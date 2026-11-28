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
- profile_name: The safe-socket profile to use (e.g., 'tcp-hello', 'udp', 'shm').
- address: Target destination (IP:Port or FilePath).
- config: Advanced SocketConfig for timeouts and identity.
"""

from ctypes import c_double as ctypesCDouble, c_ubyte as ctypesCUbyte
from sys import stderr as sysStderr
from typing import Any as typeAny, Optional as typeOptional

from .bridge import get_last_error, lib
from .connection import SafeSocketConnection
from .errors import SafeSocketError
from .models import SocketConfig

# -----------------------------------------------------------------------------------------------


class SafeSocket:
    """High-level Socket manager for Clients and Servers."""

    Name = "SafeSocket"

    def __init__(
        self,
        profile_name: str,
        address: str,
        config: typeOptional[SocketConfig] = None,
        socket_type: str = "client",
        auto_connect: bool = False,
    ) -> None:
        if config is None:
            config = SocketConfig()

        self.handle = lib.SafeSocket_CreateExtended(
            profile_name.encode("utf-8"),
            address.encode("utf-8"),
            config.public_ip.encode("utf-8"),
            socket_type.encode("utf-8"),
            config.handshake_timeout_ms,
            config.deadline_ms,
            config.heartbeat_interval_ms,
            1 if auto_connect else 0,
        )
        if self.handle == -1:
            raise SafeSocketError(f"SafeSocket (Python): Initialization failed - {get_last_error()}")
        self._closed = False

    # -----------------------------------------------------------------------------------------------

    def open(self) -> None:
        if lib.SafeSocket_Open(self.handle) == -1:
            raise SafeSocketError(f"SafeSocket (Python): Open failed - {get_last_error()}")

    # -----------------------------------------------------------------------------------------------

    def close(self) -> None:
        if not self._closed:
            if lib.SafeSocket_Close(self.handle) == -1:
                self._closed = True
                raise SafeSocketError(f"SafeSocket (Python): Close failed - {get_last_error()}")
            self._closed = True

    # -----------------------------------------------------------------------------------------------

    def send(self, data: bytes) -> None:
        data_len = len(data)
        data_ptr = (ctypesCUbyte * data_len).from_buffer_copy(data)
        if lib.SafeSocket_Send(self.handle, data_ptr, data_len) == -1:
            raise SafeSocketError(f"SafeSocket (Python): Send failed - {get_last_error()}")

    # -----------------------------------------------------------------------------------------------

    def receive(self, max_length: int = 65535) -> bytes:
        buffer = (ctypesCUbyte * max_length)()
        n = lib.SafeSocket_Receive(self.handle, buffer, max_length)
        if n == -1:
            raise SafeSocketError(f"SafeSocket (Python): Receive failed - {get_last_error()}")
        return bytes(buffer[:n])

    # -----------------------------------------------------------------------------------------------

    def listen(self) -> None:
        if lib.SafeSocket_Listen(self.handle) == -1:
            raise SafeSocketError(f"SafeSocket (Python): Listen failed - {get_last_error()}")

    # -----------------------------------------------------------------------------------------------

    def accept(self) -> SafeSocketConnection:
        conn_handle = lib.SafeSocket_Accept(self.handle)
        if conn_handle == -1:
            raise SafeSocketError(f"SafeSocket (Python): Accept failed - {get_last_error()}")
        return SafeSocketConnection(conn_handle)

    # -----------------------------------------------------------------------------------------------

    def set_deadline(self, seconds: float) -> None:
        if lib.SafeSocket_SetDeadline(self.handle, ctypesCDouble(seconds)) == -1:
            raise SafeSocketError(f"SafeSocket (Python): SetDeadline failed - {get_last_error()}")

    # -----------------------------------------------------------------------------------------------

    def set_idle_timeout(self, seconds: float) -> None:
        if lib.SafeSocket_SetIdleTimeout(self.handle, ctypesCDouble(seconds)) == -1:
            raise SafeSocketError(f"SafeSocket (Python): SetIdleTimeout failed - {get_last_error()}")

    # -----------------------------------------------------------------------------------------------

    def __enter__(self) -> "SafeSocket":
        return self

    # -----------------------------------------------------------------------------------------------

    def __exit__(self, exc_type: typeAny, exc_val: typeAny, exc_tb: typeAny) -> None:
        self.close()

    # -----------------------------------------------------------------------------------------------

    def __del__(self) -> None:
        if not getattr(self, "_closed", True):
            try:
                self.close()
            except SafeSocketError as err:
                sysStderr.write(f"{self.Name} : destructor close error: {err}\n")


# -----------------------------------------------------------------------------------------------


def create(
    profile_name: str,
    address: str,
    public_ip: str = "",
    socket_type: str = "client",
    auto_connect: bool = False,
) -> SafeSocket:
    """Simplified entry point matching Go safesocket.Create() signature."""
    config = SocketConfig(public_ip=public_ip)
    return SafeSocket(profile_name, address, config, socket_type, auto_connect)


# -----------------------------------------------------------------------------------------------


def create_with_config(
    profile_name: str,
    address: str,
    config: SocketConfig,
    socket_type: str = "client",
    auto_connect: bool = False,
) -> SafeSocket:
    """Advanced entry point matching Go safesocket.CreateWithConfig() signature."""
    return SafeSocket(profile_name, address, config, socket_type, auto_connect)
