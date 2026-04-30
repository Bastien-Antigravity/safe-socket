---
microservice: safe-socket
type: documentation
status: active
---

# Testing Guide

`safe-socket` is a high-reliability library. To ensure it handles race conditions and network edge cases correctly, follow these testing procedures.

## 🚀 Running Standard Tests

Run all unit tests across the workspace:

```bash
go test -v ./...
```

## 🏎️ Race Detection (Mandatory for CI)

Because the library utilizes background heartbeat goroutines and dynamic deadlines, **Race Detection** must be enabled for all validation:

```bash
# Requires CGO_ENABLED=1 and a localized GCC/MinGW on Windows
go test -race -v ./cmd/test/...
```

## 🛡️ Critical Safety Scenarios

We have implemented specialized tests for the "Safe" aspects of the library:

### 1. Inactivity Death (Deadlines)
`TestServerConfigDeadline` verifies that if a peer goes silent, the connection correctly closes after the `IdleTimeout`. 
- **Verification**: Even though the server is sending heartbeats, the split-deadline architecture ensures its own "Write" activity doesn't refresh its "Read" deadline.

### 2. Adaptive Heartbeat Thresholds
We verify that heartbeats are correctly disabled when the deadline is too low to prevent unnecessary CPU overhead.
- **Thresholds**: 300ms (Network), 150ms (Local), 50ms (SHM).

### 3. Handshake & Identity
Tests in `factory_test.go` and `hello_test.go` ensure that:
- Identity is preserved and extractable via `GetIdentity(conn)`.

### 4. Infinite Wait (Forever)
`TestForeverTimeoutParity` (in `transports`) confirms that setting `idleTimeout = 0` successfully clears all system deadlines. 
- **Verification**: The test intentionally waits for a period 4x longer than the initial default (400ms vs 100ms) to ensure the connection remains open.

### 5. Zombie Connection Detection
`TestZombieDetection` (in `facade`) simulates a peer that is connected but completely silent.
- **Verification**: Confirms that while standard connections remove zombies via timeouts, "Forever" connections prioritize persistence over detection.

## 🧪 Simulation Tools

The `cmd/test/` directory contains scenario-based tests that simulate network latency and tight timeouts:

- **Scenario Tests**: Custom parameters for Handshake, Data Deadline, and Heartbeat.
- **SHM Tests**: Verifies Shared Memory ring buffer synchronization between producer and consumer.

## 📊 Coverage

To generate a coverage report:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```
