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
# Features & Behavior: Safe Socket

## 🚀 Core Features

-   **Modular Transports**:
    -   **Framed TCP**: Reliable, persistent connections with single-syscall message framing (stack `[512]byte` `0 allocs/op` for small frames `<= 508B`, `net.Buffers` `writev` for large cleartext TCP frames, and coalesced single records for TLS). Optimizes `Read()` via `Peek`/`Discard` to support safe buffer pooling.
    -   **Heartbeat Support**: Client-driven unilateral 0-length heartbeat frames at the **2.5x safety ratio (`IdleTimeout / 2.5`)**. Servers default to `HeartbeatInterval = 0` so unidirectional producer clients never suffer heartbeat traffic storms or kernel buffer stalls.
    -   **UDP**: High-speed, connectionless datagram transport with 65KB demultiplexed session channels and multi-packet `ReliableConnection` ACKs/retries.
    -   **Shared Memory (SHM)**: Ultra-low latency IPC for local processes using memory-mapped files (Ring Buffer) with isolated service identities, reconnectable listeners, and adaptive spin-wait backoff (`1µs -> 50µs`).
-   **Intelligent Protocols**:
    -   **Auto-Encryption**: Automatically detects local vs remote machines using `MachineDetector`, switching seamlessly between unencrypted framed TCP (zero CPU overhead) and TLS (with full `TCP_NODELAY`, `SO_KEEPALIVE`, and 4MB buffer tuning via `underlyingTCPConn()`).
    -   **Hello Protocol**: Two-phase identity handshake enforcing tight `HandshakeTimeout` (`200ms`/`500ms`) during `WaitInitiation` before upgrading accepted sessions to runtime `IdleTimeout`, plus dynamic service address advertising (`ServiceAddress`).
    -   **Stateless Envelope (UDP)**: Zero-handshake authentication where every packet carries the sender's identity and payload.
-   **Role-Segregated & Unified Facades**: Use explicit `CreateClient` (`ClientSocket`) and `CreateServer` (`ServerSocket`) entrypoints, or the legacy `Create` / `CreateHybrid` facades.
- **Hybrid Timeout Matrix**: Uses `ResolveHybridTimeouts` (`60s` client / `10m` server local TCP; `30s` client / `3m` server Docker TCP; `5s` UDP; `0` infinite SHM), honors explicit `DisableTimeout: true` / `SetIdleTimeout(0)`, and preserves custom per-operation `SetReadDeadline` / `SetWriteDeadline` values.

### 🚀 Polyglot SDK

The core Go engine is exposed via a CGO-bridge, providing high-level, object-oriented bindings for multiple languages. All bindings are located in the `safesock/` directory and maintain 100% functional parity with the Go implementation.

*   **[Python SDK](../safesock/python)**: Type-safe wrapper following ecosystem standards.
*   **[Rust SDK](../safesock/rust)**: Memory-safe RAII wrapper.
*   **[C++ SDK](../safesock/cpp)**: Modern header-only wrapper.
*   **[VBA SDK](../safesock/vba)**: High-performance access for Microsoft Office.

Refer to the **[Polyglot SDK Documentation](../safesock/README.md)** for architecture and build details.

## 🛡️ Feature Specs & Governance (BDD)
The behavior of this microservice is governed by strict specifications in the **[[business-bdd-brain|Business-Specs Brain]]**:
- **Connection Lifecycle**: [[FEAT-000-Connection-Lifecycle|FEAT-000: State Transitions]]
- **Establishment**: [[FEAT-001-Connection-Establishment|FEAT-001: Connection Management]]
- **Framing Protocol**: [[FEAT-002-Framing-Protocol|FEAT-002: Length-Prefixed Framing]]
- **Deadline Management**: [[FEAT-003-Deadline-Management|FEAT-003: Adaptive Timeouts]]
- **Identity Handshake**: [[FEAT-004-Identity-Handshake|FEAT-004: Mutual Authentication]]
- **Shared Memory**: [[FEAT-005-Shared-Memory-Transport|FEAT-005: Ultra-low Latency IPC]]
- **Graceful Shutdown**: [[FEAT-006-Graceful-Shutdown|FEAT-006: Safe Termination]]
- **Reconnection Strategy**: [[FEAT-007-Reconnection-Strategy|FEAT-007: Fail-Fast & Fleet Recovery]]
- **Resource Boundaries**: [[FEAT-008-Resource-Boundaries|FEAT-008: Memory Isolation]]
- **Auto-Hello & Telemetry**: [[FEAT-009-Auto-Hello-And-WireTap-Telemetry|FEAT-009: Auto-Hello & WireTap Telemetry]]
