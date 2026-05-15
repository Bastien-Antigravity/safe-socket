---
microservice: safe-socket
type: tasks
status: active
---
# TODO: safe-socket

## 🚨 High Priority (Governance Gaps)
- [x] **OOM Protection**: Implement `MaxPayloadSize` check in `ReadMessage` to prevent memory exhaustion from oversized frames (FEAT-004).
- [x] **Synchronous Shutdown**: Implement a wait-state in `Close()` to ensure all background goroutines and buffers are flushed before returning (FEAT-003).
- [x] **Autonomous Reconnection**: Implement a retry loop with jittered exponential backoff in `SocketClient.Open` (FEAT-005).
- [x] **Heartbeat Audit**: Audit internal protocol timings to ensure strict compliance with the **2.5x Heartbeat Safety Ratio** (Ref: 08-Networking-Protocols).

## 🏗️ Architecture & Refactoring
- [ ] Refactor heartbeat logic to be profile-independent.
- [ ] Implement `TransportConnection` interface for all socket types.

## 🧪 Testing & CI/CD
- [x] Add stress tests for high-concurrency connection/disconnection.

## ✅ Completed
- [x] Initial BDD Spec migration.
