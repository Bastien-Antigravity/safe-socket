import ctypes
import os
import sys
from typing import Optional, Union

# Find the shared library
base_path = os.path.dirname(os.path.abspath(__file__))
lib_path = None

if sys.platform == "darwin":
    lib_path = os.path.join(base_path, "libsafe_socket.dylib")
elif sys.platform == "win32":
    lib_path = os.path.join(base_path, "libsafe_socket.dll")
else:
    lib_path = os.path.join(base_path, "libsafe_socket.so")

if not os.path.exists(lib_path):
    raise FileNotFoundError(f"Shared library not found at: {lib_path}. Please run 'make build' first.")

lib = ctypes.CDLL(lib_path)

# Argument and return types
lib.CreateSocket.argtypes = [ctypes.c_char_p, ctypes.c_char_p, ctypes.c_char_p, ctypes.c_char_p, ctypes.c_int]
lib.CreateSocket.restype = ctypes.c_int32

lib.SocketOpen.argtypes = [ctypes.c_int32]
lib.SocketOpen.restype = ctypes.c_int32

lib.SocketClose.argtypes = [ctypes.c_int32]
lib.SocketClose.restype = ctypes.c_int32

lib.SocketSend.argtypes = [ctypes.c_int32, ctypes.POINTER(ctypes.c_ubyte), ctypes.c_int]
lib.SocketSend.restype = ctypes.c_int32

lib.SocketReceive.argtypes = [ctypes.c_int32, ctypes.POINTER(ctypes.c_ubyte), ctypes.c_int]
lib.SocketReceive.restype = ctypes.c_int32

lib.SocketListen.argtypes = [ctypes.c_int32]
lib.SocketListen.restype = ctypes.c_int32

lib.SocketAccept.argtypes = [ctypes.c_int32]
lib.SocketAccept.restype = ctypes.c_int32

lib.SocketSetDeadline.argtypes = [ctypes.c_int32, ctypes.c_double]
lib.SocketSetDeadline.restype = ctypes.c_int32

lib.GetSocketError.argtypes = []
lib.GetSocketError.restype = ctypes.c_char_p

class SafeSocketError(Exception):
    pass

def _get_last_error() -> str:
    err_ptr = lib.GetSocketError()
    if err_ptr:
        return err_ptr.decode('utf-8')
    return "Unknown error"

class SafeSocket:
    def __init__(self, profile: str, address: str, public_ip: str = "", socket_type: str = "client", auto_connect: bool = False):
        self.handle = lib.CreateSocket(
            profile.encode('utf-8'),
            address.encode('utf-8'),
            public_ip.encode('utf-8'),
            socket_type.encode('utf-8'),
            1 if auto_connect else 0
        )
        if self.handle == -1:
            raise SafeSocketError(_get_last_error())
        self._closed = False

    def open(self):
        if lib.SocketOpen(self.handle) == -1:
            raise SafeSocketError(_get_last_error())

    def close(self):
        if not self._closed:
            if lib.SocketClose(self.handle) == -1:
                # Still try to mark as closed even if error
                self._closed = True
                raise SafeSocketError(_get_last_error())
            self._closed = True

    def send(self, data: bytes):
        data_len = len(data)
        data_ptr = (ctypes.c_ubyte * data_len).from_buffer_copy(data)
        if lib.SocketSend(self.handle, data_ptr, data_len) == -1:
            raise SafeSocketError(_get_last_error())

    def receive(self, max_length: int = 65535) -> bytes:
        buffer = (ctypes.c_ubyte * max_length)()
        n = lib.SocketReceive(self.handle, buffer, max_length)
        if n == -1:
            raise SafeSocketError(_get_last_error())
        return bytes(buffer[:n])

    def listen(self):
        if lib.SocketListen(self.handle) == -1:
            raise SafeSocketError(_get_last_error())

    def accept(self) -> 'SafeSocketConnection':
        conn_handle = lib.SocketAccept(self.handle)
        if conn_handle == -1:
            raise SafeSocketError(_get_last_error())
        return SafeSocketConnection(conn_handle)

    def set_deadline(self, seconds: float):
        if lib.SocketSetDeadline(self.handle, ctypes.c_double(seconds)) == -1:
            raise SafeSocketError(_get_last_error())

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()

class SafeSocketConnection:
    def __init__(self, handle: int):
        self.handle = handle
        self._closed = False

    def send(self, data: bytes):
        data_len = len(data)
        data_ptr = (ctypes.c_ubyte * data_len).from_buffer_copy(data)
        n = lib.SocketSend(self.handle, data_ptr, data_len)
        if n == -1:
            raise SafeSocketError(_get_last_error())
        return n

    def receive(self, max_length: int = 65535) -> bytes:
        buffer = (ctypes.c_ubyte * max_length)()
        n = lib.SocketReceive(self.handle, buffer, max_length)
        if n == -1:
            raise SafeSocketError(_get_last_error())
        return bytes(buffer[:n])

    def close(self):
        if not self._closed:
            if lib.SocketClose(self.handle) == -1:
                self._closed = True
                raise SafeSocketError(_get_last_error())
            self._closed = True

    def set_deadline(self, seconds: float):
        if lib.SocketSetDeadline(self.handle, ctypes.c_double(seconds)) == -1:
            raise SafeSocketError(_get_last_error())

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()

def create(profile: str, address: str, public_ip: str = "", socket_type: str = "client", auto_connect: bool = False) -> SafeSocket:
    return SafeSocket(profile, address, public_ip, socket_type, auto_connect)
