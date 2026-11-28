---
tags:
- '#ai/ignore'
- '#zone/3-fleet'
- '#service/safe-socket'
- '#type/overview'
- '#state/active'
microservice: safe-socket
type: overview
status: active
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
| `"auto"` | TCP or TLS | None | `IP:Port` | Local TCP vs. Remote TLS via `MachineDetector`. |
| `"auto-hello"` | TCP or TLS | Hello | `IP:Port` | Auto-resolved TCP/TLS + Identity Handshake. |

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

## Unified Hybrid Timeout Matrix (`ResolveHybridTimeouts`)

`Create`, `CreateClient`, `CreateServer`, and `CreateHybrid` share the canonical `ResolveHybridTimeouts` matrix unless overridden via explicit `SocketConfig`:

| Condition / Role | Handshake Timeout | Data Deadline (Idle) | Heartbeat Interval |
| :--- | :--- | :--- | :--- |
| **Local TCP/TLS (Client)** | 200ms | 60s | 24s (`60s / 2.5`) |
| **Local TCP/TLS (Server)** | 200ms | 10m | 0 (Passive) |
| **Docker TCP/TLS (Client)** | 500ms | 30s | 12s (`30s / 2.5`) |
| **Docker TCP/TLS (Server)** | 500ms | 3m | 0 (Passive) |
| **UDP (Client / Server)** | 200ms–500ms | 5s | Disabled |
| **SHM (Client / Server)** | 100ms | 0 (`DisableTimeout: true`) | Disabled |

## Project Structure

- `cmd/libsafesocket`: CGO bridge implementation.
- `cmd/test`: Integration and scenario tests.
- `safesock/`: Foreign language bindings.
- `src/`: Core Go implementation.
    - `facade/`: Public API implementation.
    - `factory/`: Socket instantiation logic.
    - `protocols/`: Handshake and framing logic.
    - `transports/`: I/O implementations (TCP, UDP, SHM).
