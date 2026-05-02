# TODO: safe-socket

## 🚨 High Priority (Governance Gaps)
- [ ] **OOM Protection**: Implement `MaxPayloadSize` check in `ReadMessage` to prevent memory exhaustion from oversized frames (FEAT-004). (Approval Required)
- [ ] **Synchronous Shutdown**: Implement a wait-state in `Close()` to ensure all background goroutines and buffers are flushed before returning (FEAT-003). (Approval Required)
- [ ] **Autonomous Reconnection**: Implement a retry loop with jittered exponential backoff in `SocketClient.Open` (FEAT-005). (Approval Required)

## 🏗️ Architecture & Refactoring
- [ ] Refactor heartbeat logic to be profile-independent.
- [ ] Implement `TransportConnection` interface for all socket types.

## 🧪 Testing & CI/CD
- [ ] Add stress tests for high-concurrency connection/disconnection.

## ✅ Completed
- [x] Initial BDD Spec migration.