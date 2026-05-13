# Architecture: Python SDK

The Python SDK acts as a robust FFI layer over the Go-based `libsafesocket` shared library.

## Design Patterns

- **Handle-based Lifecycle**: The `SafeSocket` and `SafeSocketConnection` classes manage an `int32` handle registered in the Go bridge. They ensure `SafeSocket_Close` is called via the `__exit__` context manager or the `close()` method.
- **ctypes Mapping**: Standard `ctypes` mappings are used for all C exports, with aliased functional imports (e.g., `ctypesCDLL`, `ctypesPOINTER`) to distinguish standard actions from local variables.
- **Memory Safety**: Data is moved between Python and Go using `ctypes.memmove` and `GoBytes`, ensuring safe crossing of the FFI boundary without memory corruption.

## Data Flow

1. Python calls `SafeSocket_Create` or `SafeSocket_CreateExtended`.
2. The CGO Bridge (`src/cgo_bridge`) initializes the Go socket and registers it in a `sync.Map`.
3. An integer handle is returned to Python.
4. Subsequent calls pass this handle back to Go, which retrieves the socket from the registry to perform I/O.
5. Errors are captured in a thread-safe C global (`last_socket_error`) and retrieved by Python upon failure.
