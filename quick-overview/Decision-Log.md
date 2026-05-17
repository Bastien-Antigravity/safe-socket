---
microservice: safe-socket
type: decision-log
status: active

tags:
- \'#service/safe-socket\'
- '#ai/ignore'
---

# Decision Log

Historical record of significant architectural and technical decisions for `safe-socket`.

- **v1.10.0 (2026-05-17)**: Hardened repository stability and thread-safety.
    - **Thread-Safety**: Implemented mutex protection in `HeartbeatConnection` to prevent race conditions during concurrent I/O.
    - **Robust Recovery**: Added jittered exponential backoff to `SocketClient.Open` for autonomous reconnection.
    - **Configuration Compliance**: Removed direct environment variable overrides (`SAFE_SOCKET_*`) to align with the ecosystem's centralized configuration standards (Rule 3/7).
- **v1.9.0 (2026-05-15)**: Standardized "Infinite Wait" (0) logic across all transports (TCP, UDP, SHM). This allows users to disable idle timeouts for persistent connections that should remain open indefinitely unless explicitly closed.
- **CI Stabilization (2026-05-15)**: Migrated to `golangci-lint` v1.64.2 and standardized CI workflows to ensure consistent code quality and faster feedback loops across the polyglot SDK.
- **SDK Polyglot Refactoring (2026-05-14)**: Relocated CGO bridge to `src/cgo_bridge` and updated Python, Rust, C++, and VBA bindings to maintain functional parity with the core Go implementation and simplify cross-language development.
- **Zombie Detection (2026-05-13)**: Implemented silent-peer detection to identify and clean up "zombie" connections that are no longer responsive but haven't been formally closed. Verified via dedicated stress and scenario unit tests.

---
[Back to Map of Content](./Map-of-Content.md)
