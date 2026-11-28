#!/usr/bin/env python
# coding:utf-8

"""
ESSENTIAL PROCESS:
Defines SafeSocketConnection representing an accepted server-side peer connection,
providing framed send/receive I/O, deadline management, and RAII context-manager cleanup.

DATA FLOW:
1. Input: Integer handle returned by SafeSocket_Accept.
2. Logic: Invokes libsafesocket C ABI methods on the handle and translates errors.
3. Output: Framed byte payloads and lifecycle state transitions.

KEY PARAMETERS:
- handle: Opaque 32-bit integer handle registered in Go cgo_bridge.
"""

from ctypes import c_double as ctypesCDouble, c_ubyte as ctypesCUbyte
from sys import stderr as sysStderr
from typing import Any as typeAny

from .bridge import get_last_error, lib
from .errors import SafeSocketError

# -----------------------------------------------------------------------------------------------


class SafeSocketConnection:
    """Represents an active connection returned by accept()."""

    Name = "SafeSocketConnection"

    def __init__(self, handle: int) -> None:
        self.handle = handle
        self._closed = False

    # -----------------------------------------------------------------------------------------------

    def send(self, data: bytes) -> int:
        data_len = len(data)
        data_ptr = (ctypesCUbyte * data_len).from_buffer_copy(data)
        n = lib.SafeSocket_Send(self.handle, data_ptr, data_len)
        if n == -1:
            raise SafeSocketError(f"SafeSocket (Python): Send failed - {get_last_error()}")
        return n

    # -----------------------------------------------------------------------------------------------

    def receive(self, max_length: int = 65535) -> bytes:
        buffer = (ctypesCUbyte * max_length)()
        n = lib.SafeSocket_Receive(self.handle, buffer, max_length)
        if n == -1:
            raise SafeSocketError(f"SafeSocket (Python): Receive failed - {get_last_error()}")
        return bytes(buffer[:n])

    # -----------------------------------------------------------------------------------------------

    def close(self) -> None:
        if not self._closed:
            if lib.SafeSocket_Close(self.handle) == -1:
                self._closed = True
                raise SafeSocketError(f"SafeSocket (Python): Close failed - {get_last_error()}")
            self._closed = True

    # -----------------------------------------------------------------------------------------------

    def set_deadline(self, seconds: float) -> None:
        if lib.SafeSocket_SetDeadline(self.handle, ctypesCDouble(seconds)) == -1:
            raise SafeSocketError(f"SafeSocket (Python): SetDeadline failed - {get_last_error()}")

    # -----------------------------------------------------------------------------------------------

    def set_idle_timeout(self, seconds: float) -> None:
        if lib.SafeSocket_SetIdleTimeout(self.handle, ctypesCDouble(seconds)) == -1:
            raise SafeSocketError(f"SafeSocket (Python): SetIdleTimeout failed - {get_last_error()}")

    # -----------------------------------------------------------------------------------------------

    def __enter__(self) -> "SafeSocketConnection":
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
