---
tags:
- '#ai/ignore'
- '#zone/3-fleet'
---
# Features & Behavior: Safe Socket

## 🚀 Core Features

-   **Modular Transports**:
    -   **Framed TCP**: Reliable, persistent connections with message framing. Optimizes `Read()` via buffering to support safe buffer pooling (prevents header loss on short reads). 
    -   **Heartbeat Support**: Automatically handles 0-length frames as heartbeats. Includes an **Adaptive safety-ratio (2.5x)** that ensures heartbeats always fire before an idle timeout occurs.
    -   **UDP**: High-speed, connectionless communication with optional reliability layers.
    -   **Shared Memory (SHM)**: Ultra-low latency IPC for local processes using memory-mapped files (Ring Buffer).
-   **Intelligent Protocols**:
    -   **Hello Protocol**: Identity exchange handshake.
    -   **Stateless Envelope (UDP)**: Zero-handshake authentication where every packet carries the sender's identity and payload.
-   **Unified Facade**: Interact with any transport using `Open()`, `Close()`, `Send()`, `Receive()`, and `Accept()`.
- **Aggressive Responsiveness**: Optimized for high-frequency microservice environments with extremely tight default timeouts (500ms network / 100ms SHM) and an active activity-refresh deadline model.

### 🚀 Polyglot SDK

The core Go engine is exposed via a CGO-bridge, providing high-level, object-oriented bindings for multiple languages. All bindings are located in the `safesock/` directory and maintain 100% functional parity with the Go implementation.

*   **[Python SDK](../safesock/python)**: Type-safe wrapper following ecosystem standards.
*   **[Rust SDK](../safesock/rust)**: Memory-safe RAII wrapper.
*   **[C++ SDK](../safesock/cpp)**: Modern header-only wrapper.
*   **[VBA SDK](../safesock/vba)**: High-performance access for Microsoft Office.

Refer to the **[Polyglot SDK Documentation](../safesock/README.md)** for architecture and build details.

## 🛡️ Feature Specs & Governance (BDD)
The behavior of this microservice is governed by strict specifications in the **[[business-bdd-brain|Business-Specs Brain]]**:
- **Connection Lifecycle**: [[FEAT-000-Connection-Lifecycle|FEAT-000: State Transitions]]
- **Establishment**: [[FEAT-001-Connection-Establishment|FEAT-001: Connection Management]]
- **Framing Protocol**: [[FEAT-002-Framing-Protocol|FEAT-002: Length-Prefixed Framing]]
- **Deadline Management**: [[FEAT-003-Deadline-Management|FEAT-003: Adaptive Timeouts]]
- **Identity Handshake**: [[FEAT-004-Identity-Handshake|FEAT-004: Mutual Authentication]]
- **Shared Memory**: [[FEAT-005-Shared-Memory-Transport|FEAT-005: Ultra-low Latency IPC]]
- **Graceful Shutdown**: [[FEAT-006-Graceful-Shutdown|FEAT-006: Safe Termination]]
- **Reconnection Strategy**: [[FEAT-007-Reconnection-Strategy|FEAT-007: Automatic Recovery]]
- **Resource Boundaries**: [[FEAT-008-Resource-Boundaries|FEAT-008: Memory Isolation]]
