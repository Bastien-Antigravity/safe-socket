# SafeSocket Polyglot SDK

This directory contains the official language wrappers for the `safe-socket` CGO bridge. These wrappers provide a native, object-oriented interface for non-Go languages while maintaining 100% architectural parity with the Go core.

## Directory Structure

- **[cpp/](./cpp)**: Header-only C++14+ wrapper in `SafeSocket.hpp`.
- **[python/](./python)**: Type-safe Python wrapper following ecosystem standards.
- **[rust/](./rust)**: Safe Rust wrapper using `libloading` for dynamic link access.
- **[vba/](./vba)**: VBA module for high-performance network access in Office.
- **[libsafesocket/](./libsafesocket)**: Compiled shared library binaries and C header.

## Core API Design

All wrappers provide a unified high-level interface through the `SafeSocket` (manager) and `SafeSocketConnection` (active stream) classes.

| Method | Description |
| :--- | :--- |
| `Create(profile, address, ...)` | Factory for standard socket creation. |
| `CreateExtended(profile, address, config, ...)` | Factory for advanced configuration (timeouts). |
| `Open()` | Actively connects the socket. |
| `Send(data)` | Transmits data through the socket. |
| `Receive(max_length)` | Retrieves data from the socket. |
| `Listen()` | Activates the server listener. |
| `Accept()` | Returns a new `SafeSocketConnection` for inbound clients. |
| `SetDeadline(seconds)` | Sets an absolute I/O deadline. |
| `SetIdleTimeout(seconds)` | Sets an activity-based deadline refresh. |
| `Close()` | Safely shuts down the connection and releases bridge handles. |

## Build Instructions

Before using any of these SDKs, you must build the shared library:

```bash
cd ..
make build-lib
```

This generates the necessary binaries in `safesock/libsafesocket/libsafesocket.so` (or `.dylib`, `.dll`).

## Testing

Each language has its own example or test suite.
Refer to the `TESTING.md` in each subfolder for detailed instructions.
