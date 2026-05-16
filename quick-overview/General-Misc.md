---
tags:
- '#ai/ignore'
---
# General Miscellaneous Information

This document contains miscellaneous information about the `safe-socket` project, including installation, supported profiles, polyglot SDKs, and default behaviors.

## Installation

```bash
go get github.com/Bastien-Antigravity/safe-socket
```

## Supported Profiles

| Profile | Transport | Protocol | Address Format | Description |
| :--- | :--- | :--- | :--- | :--- |
| `"tcp"` | TCP | None | `IP:Port` | Raw TCP stream. |
| `"tcp-hello"` | TCP | Hello | `IP:Port` | TCP + Identity Handshake. |
| `"udp"` | UDP | None | `IP:Port` | Raw UDP packets. |
| `"udp-hello"` | UDP | Hello | `IP:Port` | Stateless Envelope (Identity + Payload). |
| `"shm"` | SHM | None | File Path | Raw Memory Mapped File. |
| `"shm-hello"` | SHM | Hello | File Path | SHM + Identity Handshake. |
| `"tls"` | TLS | None | `IP:Port` | Raw TLS stream (Encrypted TCP). |
| `"tls-hello"` | TLS | Hello | `IP:Port` | TLS + Identity Handshake. |

## TLS Configuration

When using `tls` or `tls-hello` profiles, you must provide certificates via `models.SocketConfig`:

- **`CertFile`**: Path to the public certificate (.crt, .pem).
- **`KeyFile`**: Path to the private key (.key).
- **`CAFile`**: Path to a CA bundle for verifying peers (required for mTLS).
- **`ServerName`**: Expected hostname (used for SNI validation on clients).
- **`InsecureSkipVerify`**: If true, skips certificate chain and hostname validation (not recommended for production).

## Polyglot SDK (CGO Bridge)

The core Go engine is exposed via a CGO bridge, providing high-level bindings for multiple languages:

- **[Python](./safesock/python)**: Type-safe wrapper.
- **[Rust](./safesock/rust)**: Memory-safe RAII wrapper.
- **[C++](./safesock/cpp)**: Modern header-only wrapper.
- **[VBA](./safesock/vba)**: High-performance access for MS Office.

## High-Responsiveness Defaults

To ensure system health, the library defaults to aggressive timeouts:

| Condition | Handshake Timeout | Data Deadline (Idle) | Heartbeat Interval |
| :--- | :--- | :--- | :--- |
| **Network** | 500ms | 500ms | 200ms |
| **Local** | 200ms | 200ms | 80ms |
| **SHM** | 100ms | 100ms | 40ms |

## Project Structure

- `cmd/libsafesocket`: CGO bridge implementation.
- `cmd/test`: Integration and scenario tests.
- `safesock/`: Foreign language bindings.
- `src/`: Core Go implementation.
    - `facade/`: Public API implementation.
    - `factory/`: Socket instantiation logic.
    - `protocols/`: Handshake and framing logic.
    - `transports/`: I/O implementations (TCP, UDP, SHM).
