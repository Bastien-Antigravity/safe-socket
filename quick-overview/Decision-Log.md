---
microservice: safe-socket
type: decision-log
status: active
tags:
- '#service/safe-socket'
- '#ai/ignore'
- '#zone/3-fleet'
- '#type/decision-log'
- '#state/active'
---
# Decision Log

Historical record of significant architectural and technical decisions for `safe-socket`.

- **v1.9.1 (2026-11-27)**: Two-Phase Handshake Isolation, Unified Hybrid Timeouts, Role Segregation & Hot-Path Hardening (`ADR-003`).
    - **Two-Phase Handshake Isolation**: `SocketServer.Accept()` now enforces tight `HandshakeTimeout` (`200ms`/`500ms`) during `proto.WaitInitiation(conn)` before upgrading accepted connections to the runtime `IdleTimeout` (`10m`/`3m`), blocking Slow-Loris stalls.
    - **Single-Attempt Fail-Fast `SocketClient.Open()`**: Removed the conflicting exponential-backoff retry loop and `MaxRetries`/`RetryInterval` fields from `SocketClient`/`SocketConfig`; fleet reconnection backoff is delegated solely to `microservice-toolbox/conn_manager`.
    - **Role-Segregated Interfaces**: Added `ClientSocket` / `ServerSocket` (`CreateClient`, `CreateClientWithConfig`, `CreateServer`, `CreateServerWithConfig`).
    - **Unified Hybrid Defaults & Client-Driven Heartbeats**: Unified `Create` and `CreateHybrid` around `ResolveHybridTimeouts`, honored infinite waits (`DisableTimeout: true` / `Deadline < 0` / `SetIdleTimeout(0)`), preserved custom `SetReadDeadline`/`SetWriteDeadline`, and disabled server-outbound heartbeats when `HeartbeatInterval <= 0`.
    - **Hot-Path & Transport Fixes**: Stack-allocated `[512]byte` (`0 B/op` for `<= 508B` writes) and `net.Buffers` `writev` framing + close-on-partial-write in `FramedTCPSocket`, `underlyingTCPConn()` TLS socket optimization unwrapping, multi-session SHM reconnects with `ShmProfile` identity separation + adaptive spin backoff + write/read mutexes, 65KB UDP `demuxLoop` session routing, Darwin CGO `get_socket_error()` / `free_socket_error()` getter with `runtime.LockOSThread()`, and Bastien-standard Python SDK modularization.
- **v0.0.1 (2026-05-17)**: Hardened repository stability and thread-safety.
    - **Thread-Safety**: Implemented mutex protection in `HeartbeatConnection` to prevent race conditions during concurrent I/O.
    - **Robust Recovery**: Added jittered exponential backoff to `SocketClient.Open` for autonomous reconnection.
    - **Configuration Compliance**: Removed direct environment variable overrides (`SAFE_SOCKET_*`) to align with the ecosystem's centralized configuration standards (Rule 3/7).
- **v1.9.0 (2026-05-15)**: Standardized "Infinite Wait" (0) logic across all transports (TCP, UDP, SHM). This allows users to disable idle timeouts for persistent connections that should remain open indefinitely unless explicitly closed.
- **CI Stabilization (2026-05-15)**: Migrated to `golangci-lint` v1.64.2 and standardized CI workflows to ensure consistent code quality and faster feedback loops across the polyglot SDK.
- **SDK Polyglot Refactoring (2026-05-14)**: Relocated CGO bridge to `src/cgo_bridge` and updated Python, Rust, C++, and VBA bindings to maintain functional parity with the core Go implementation and simplify cross-language development.
- **Zombie Detection (2026-05-13)**: Implemented silent-peer detection to identify and clean up "zombie" connections that are no longer responsive but haven't been formally closed. Verified via dedicated stress and scenario unit tests.

---
[Back to Map of Content](./Map-of-Content.md)
