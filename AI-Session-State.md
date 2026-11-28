---
microservice: safe-socket
type: session-state
status: active
tags:
  - '#service/safe-socket'
  - '#type/session-state'
  - '#state/active'
  - '#zone/3-fleet'
---

# AI Session State: safe-socket

Mission-ID: MISSION-AUTONOMOUS-001
Trace-ID: TRACE-AUTONOMOUS-001
Status: Active / Ready

## Latest Audit & Hardening Session (2026-11-27)
- **Architectural Cleanup**: Removed committed binaries & duplicate C headers; unified `Create*` and `CreateHybrid` onto `ResolveHybridTimeouts`; honored infinite waits (`DisableTimeout: true` / `Deadline < 0` / `SetIdleTimeout(0)`); segregated `ClientSocket` / `ServerSocket` facades; removed redundant backoff loop from `SocketClient.Open` (delegated to `microservice-toolbox/conn_manager`).
- **Critical Transport & FFI Fixes**:
  - `SocketServer.Accept()`: Two-phase timeout isolation (`HandshakeTimeout` during `WaitInitiation`, immediate close on stall/corrupt packet, upgrade to `IdleTimeout` post-handshake).
  - `FramedTCPSocket`: Unwraps `*tls.Conn` in `underlyingTCPConn()`; single-write atomic framing (`[512]byte` stack buffer for `<=508B`, `net.Buffers` `writev` for large cleartext TCP, coalesced buffer for TLS, close on partial write failure); atomic `idleTimeoutNs` + preserved custom read/write deadlines.
  - `ShmTransport` / `ShmListener`: Reconnectable session lifecycle, isolated `NewShmProfileWithIdentity`, `writeMu`/`readMu` synchronization, adaptive spin backoff (`1µs -> 50µs`), lock-free status checks on accept/close.
  - `UdpListener` / `UdpSocket`: 65KB `demuxLoop` session routing for non-truncated datagrams and multi-packet `ReliableConnection` sessions.
  - `cgo_bridge` & Python SDK: Added Darwin-safe C `get_socket_error()` / `free_socket_error()` with `runtime.LockOSThread()`, atomic `NextHandle`, modular Python SDK with Bastien granular camelCase aliasing, and consolidated `scenario_test.py`.
- **Docs & Specs Synced**: Updated `FEAT-000`, `FEAT-003`, `FEAT-004`, `FEAT-007`, added `FEAT-009`, updated `ADR-001`, created `ADR-003`, updated `07-Hidden-Patterns-and-Gotchas.md` `#11`, and added `Benchmark*` (`-benchmem`) + `TestHandshakeSlowLorisTimeout` + `TestRoleSegregatedClientAndServer`.
