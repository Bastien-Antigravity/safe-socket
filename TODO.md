---
microservice: safe-socket
type: tasks
status: active
tags:
- '#service/safe-socket'
- '#zone/3-fleet'
- '#type/tasks'
- '#state/active'
---
# TODO: safe-socket

## 🚨 High Priority (Governance Gaps)

- [X] **OOM Protection**: Implement `MaxPayloadSize` check in `ReadMessage` to prevent memory exhaustion from oversized frames (FEAT-004).
- [X] **Synchronous Shutdown**: Implement a wait-state in `Close()` to ensure all background goroutines and buffers are flushed before returning (FEAT-003).
- [X] **Fail-Fast Client Dial Boundary**: Removed autonomous backoff loop from `SocketClient.Open`; resilience and `AppConfig` live-sync delegated to `microservice-toolbox/conn_manager` (FEAT-007).
- [X] **Two-Phase Handshake Isolation & Hot-Path Optimization**: Isolated pre-handshake `HandshakeTimeout` from runtime `IdleTimeout` in `SocketServer.Accept()`, added zero-alloc stack framing + `writev` scatter-gather I/O + reproducible `-benchmem` benchmarks (ADR-003).
- [X] **Heartbeat Audit**: Audit internal protocol timings to ensure strict compliance with the **2.5x Heartbeat Safety Ratio** (Ref: 08-Networking-Protocols).

## 🏗️ Architecture & Refactoring

- [X] Refactor heartbeat logic to be profile-independent (Implemented via Facade Wrapper).
- [X] Implement `TransportConnection` interface for all socket types.

## 🧪 Testing & CI/CD

- [X] Add stress tests for high-concurrency connection/disconnection.

## ✅ Completed

- [X] Initial BDD Spec migration.
