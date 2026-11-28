#!/usr/bin/env python
# coding:utf-8

"""
ESSENTIAL PROCESS:
Loads the libsafesocket shared library (.dylib, .so, .dll), binds C ABI function
signatures via ctypes, and provides thread-local error extraction helpers.

DATA FLOW:
1. Input: LIBSAFESOCKET_PATH environment variable or standard workspace library paths.
2. Logic: Locates the compiled CGO shared library and configures argtypes/restype.
3. Output: Shared ctypes CDLL instance and get_last_error helper.

KEY PARAMETERS:
- lib: Loaded ctypes CDLL handle for libsafesocket.
"""

from ctypes import (
    CDLL as ctypesCDLL,
    POINTER as ctypesPOINTER,
    c_char_p as ctypesCCharP,
    c_double as ctypesCDouble,
    c_int as ctypesCInt,
    c_int32 as ctypesCInt32,
    c_ubyte as ctypesCUbyte,
)
from os import getenv as osGetenv
from os.path import (
    abspath as osPathAbspath,
    dirname as osPathDirname,
    exists as osPathExists,
    join as osPathJoin,
)
from sys import platform as sysPlatform

# -----------------------------------------------------------------------------------------------

_base_path = osPathDirname(osPathAbspath(__file__))
_search_paths = [
    _base_path,
    osPathJoin(_base_path, ".."),
    osPathJoin(_base_path, "../../libsafesocket"),
    osPathJoin(_base_path, "../../../safesock/libsafesocket"),
]

_lib_name = "libsafesocket"
_lib_ext = ".so"
if sysPlatform == "darwin":
    _lib_ext = ".dylib"
elif sysPlatform == "win32":
    _lib_ext = ".dll"

_lib_path = osGetenv("LIBSAFESOCKET_PATH")
if not (_lib_path and osPathExists(_lib_path)):
    _lib_path = None
    for path in _search_paths:
        full_path = osPathJoin(path, f"{_lib_name}{_lib_ext}")
        if osPathExists(full_path):
            _lib_path = full_path
            break

if not _lib_path:
    raise FileNotFoundError(
        f"SafeSocket (Python): Shared library {_lib_name}{_lib_ext} not found. "
        "Please check paths or set LIBSAFESOCKET_PATH."
    )

lib = ctypesCDLL(_lib_path)

lib.SafeSocket_Create.argtypes = [ctypesCCharP, ctypesCCharP, ctypesCCharP, ctypesCCharP, ctypesCInt]
lib.SafeSocket_Create.restype = ctypesCInt32

lib.SafeSocket_CreateExtended.argtypes = [
    ctypesCCharP,
    ctypesCCharP,
    ctypesCCharP,
    ctypesCCharP,
    ctypesCInt,
    ctypesCInt,
    ctypesCInt,
    ctypesCInt,
]
lib.SafeSocket_CreateExtended.restype = ctypesCInt32

lib.SafeSocket_Open.argtypes = [ctypesCInt32]
lib.SafeSocket_Open.restype = ctypesCInt32

lib.SafeSocket_Close.argtypes = [ctypesCInt32]
lib.SafeSocket_Close.restype = ctypesCInt32

lib.SafeSocket_Send.argtypes = [ctypesCInt32, ctypesPOINTER(ctypesCUbyte), ctypesCInt]
lib.SafeSocket_Send.restype = ctypesCInt32

lib.SafeSocket_Receive.argtypes = [ctypesCInt32, ctypesPOINTER(ctypesCUbyte), ctypesCInt]
lib.SafeSocket_Receive.restype = ctypesCInt32

lib.SafeSocket_Listen.argtypes = [ctypesCInt32]
lib.SafeSocket_Listen.restype = ctypesCInt32

lib.SafeSocket_Accept.argtypes = [ctypesCInt32]
lib.SafeSocket_Accept.restype = ctypesCInt32

lib.SafeSocket_SetDeadline.argtypes = [ctypesCInt32, ctypesCDouble]
lib.SafeSocket_SetDeadline.restype = ctypesCInt32

lib.SafeSocket_SetIdleTimeout.argtypes = [ctypesCInt32, ctypesCDouble]
lib.SafeSocket_SetIdleTimeout.restype = ctypesCInt32

lib.SafeSocket_GetSocketError.argtypes = []
lib.SafeSocket_GetSocketError.restype = ctypesCCharP

# -----------------------------------------------------------------------------------------------


def get_last_error() -> str:
    """Retrieves the thread-local error message from libsafesocket."""
    err_ptr = lib.SafeSocket_GetSocketError()
    if err_ptr:
        return err_ptr.decode("utf-8")
    return "Unknown error"
