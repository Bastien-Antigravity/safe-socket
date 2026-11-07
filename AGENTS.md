# AGENTS.md: safe-socket

## Service Mission & Architecture Role
`safe-socket` is the high-reliability TCP framing and transport library for the Bastien-Antigravity fleet. It solves network partitions, half-open sockets, and message fragmentation via automatic exponential backoff reconnection, 4-byte length-prefixed packet framing, and 30-second ping/pong heartbeat mechanisms.

- **Ecosystem Role**: High-speed, robust TCP transport layer.
- **Consumers**: `log-server`, `universal-logger` network sinks, inter-agent data streams.
- **Protocol Contract**:
  - Framing: Big-Endian 4-byte integer length header followed by byte payload.
  - Heartbeats: `PING` / `PONG` frames automatically exchanged every 30s.

## Key Build & Test Commands
```bash
# Run unit & integration tests
go test -v ./...

# Run race condition checks
go test -race ./...
```

## AI Development & Integration Guidelines
1. **Never Alter Wire Protocol Without Fleet-Wide Coordination**: The 4-byte prefix and heartbeat framing are implemented in Go, Rust (`log-server`), and Python. Breaking the wire format will break fleet logging.
2. **Goroutine Leak Prevention**: Any network read/write loop must bind to a `context.Context` and terminate cleanly when the context is cancelled.
3. **Header Ritual**: All Go files MUST begin with the Triple-Block header (`ESSENTIAL PROCESS`, `DATA FLOW`, `KEY PARAMETERS`).
4. **Section Dividers**: Use `// -----------------------------------------------------------------------------` between exported methods.
5. **TCP Probe Resilience**: In `SocketServer.Accept()`, 0-byte TCP health checks and port monitors (`io.EOF` / `io.ErrUnexpectedEOF` during handshake) must be cleanly closed and loop continued. Active protocol violations (non-hello traffic) must strictly return an error immediately.
6. **EnsureSafeLogger Standard**: Injected loggers must always be wrapped with `interfaces.EnsureSafeLogger(l)` in constructors (`NewSocketServer`, `NewSocketClient`) and `SetLogger` to avoid `nil` pointer panics and avoid defensive `if Logger != nil` checks.
7. **Lock-Free Atomic Telemetry Wrapper (`src/facade/telemetry_wrapper.go`)**:
   - Real-time connection and frame observability (WireTap) is implemented in a dedicated wrapper file (`src/facade/telemetry_wrapper.go`) and exposed via root package functions `safesocket.SetTelemetryEnabled(bool)`, `safesocket.IsTelemetryEnabled() bool`, and `safesocket.SetTelemetryHook(TelemetryHook)`.
   - **Strictly Lock-Free**: The enable/disable switch (`telemetryEnabled` via `sync/atomic.Bool`) and hook registration (`telemetryHookHolder` via `sync/atomic.Value`) MUST remain 100% lock-free (zero `sync.Mutex` or channel waits on the I/O path).
   - **Zero Overhead When Disabled**: When `IsTelemetryEnabled()` is `false`, `wrapConnectionWithTelemetry` and `wrapSocketWithTelemetry` return the underlying connection/socket directly without wrapping or allocating.
   - **Segregated Channel Extraction**: The wrapper transparently inspects non-binary payload prefixes (e.g., `SYNC_CONFIG`, `LOG_BATCH`, `HEARTBEAT`) and peer `HelloMsg` identities (`FromName()`) without mutating the 4-byte length-prefixed wire protocol.


