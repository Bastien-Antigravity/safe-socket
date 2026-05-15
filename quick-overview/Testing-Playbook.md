# Testing Playbook

`safe-socket` is a high-reliability library. To ensure it handles race conditions, network edge cases, and high-frequency communication correctly, follow these testing procedures.

## Running Tests

### Standard Unit Tests
Run all unit tests across the workspace:
```bash
go test -v ./...
```

### Race Detection (Mandatory)
Because the library utilizes background heartbeat goroutines and dynamic deadlines, **Race Detection** must be enabled for all validation to catch concurrency issues:
```bash
go test -race -v ./...
```

### Coverage Reports
To generate and view a coverage report:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Critical Safety Scenarios

We have implemented specialized tests for the "Safe" aspects of the library:

### 1. Inactivity Death (Deadlines)
Verifies that if a peer goes silent, the connection correctly closes after the `IdleTimeout`. 
- **Key Test**: `TestServerConfigDeadline` (in `src/facade/shutdown_test.go`).
- **Logic**: Even if the server is sending heartbeats, its own write activity must not refresh its read deadline.

### 2. Adaptive Heartbeat Thresholds
Verifies that heartbeats are correctly disabled when the deadline is too low to prevent unnecessary CPU overhead.
- **Thresholds**: 300ms (Network), 150ms (Local), 50ms (SHM).

### 3. Identity Verification
Ensures that identity is preserved and extractable via `GetIdentity(conn)`.
- **Key Tests**: `factory_test.go` and `hello_test.go`.

### 4. TLS & Encryption
Verifies that TLS connections are established correctly and identity handshakes are performed over encrypted channels.
- **Key Test**: `TestTLS_Hello` (in `cmd/test/factory_test.go`).
- **Logic**: Confirms that certificates are correctly loaded and mutual authentication works.

### 5. Infinite Wait (Forever)
Confirms that setting `idleTimeout = 0` successfully clears all system deadlines.
- **Key Test**: `TestForeverTimeoutParity` (in `src/transports/forever_test.go`).

### 6. Zombie Connection Detection
Simulates a peer that is connected but completely silent.
- **Key Test**: `TestZombieDetection` (in `src/transports/zombie_test.go`).

## Simulation Tools (`cmd/test/`)

The `cmd/test/` directory contains scenario-based tests that simulate real-world conditions:
- **`scenario_test.go`**: Custom parameters for Handshake, Data Deadline, and Heartbeat.
- **`stress_test.go`**: High-frequency message exchange to verify stability under load.
- **`shm_test.go`**: Verifies Shared Memory ring buffer synchronization between producer and consumer.
