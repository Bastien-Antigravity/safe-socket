# AGENTS.md: safe-socket

## Service Mission & Architecture Role
`safe-socket` is the high-reliability framed transport library for the Bastien-Antigravity fleet. It solves half-open sockets, message fragmentation, and local-vs-remote transport selection via 4-byte length-prefixed packet framing, activity-refreshed hybrid deadlines, and unilateral 0-length heartbeat frames at the `2.5x` safety ratio (`IdleTimeout / 2.5`). Exponential-backoff reconnection is orchestrated by `microservice-toolbox/go/pkg/conn_manager` above `SocketClient.Open()`.

- **Ecosystem Role**: High-speed, robust transport layer (Framed TCP, TLS, UDP, Shared Memory).
- **Consumers**: `log-server`, `notif-server`, `config-server`, `universal-logger`, `distributed-config`, `microservice-toolbox`.
- **Interfaces**: Role-segregated `ClientSocket` (`CreateClient` / `CreateClientWithConfig`) and `ServerSocket` (`CreateServer` / `CreateServerWithConfig`), plus unified `Socket` (`Create` / `CreateWithConfig`) for backward compatibility.
- **Protocol Contract**:
  - Framing: Big-Endian 4-byte integer length header followed by byte payload (on `"tcp"`, `"tcp-hello"`, `"tls"`, `"tls-hello"`, `"auto"`, `"shm"`, and `"shm-hello"`).
  - Heartbeats: Unilateral 0-length frames (`0x00000000`) dispatched by clients every `IdleTimeout / 2.5` (e.g., `24s` for `60s` local idle timeout, `12s` for `30s` Docker idle timeout); servers disable outbound heartbeats unless `HeartbeatInterval > 0` is explicitly configured.

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


