# Future Roadmap: safe-socket

## High Priority: Ecosystem Integration

- [ ] **Dynamic PublicIP Management**:
    - Implement a mechanism to refresh the `PublicIP` on connection errors.
    - Integration with `microservice-toolbox` for network discovery.
    - Support for `SAFE_SOCKET_PUBLIC_IP` environment variable as a source of truth.
- [ ] **Environmental Overrides**:
    - Allow global override of aggressive defaults via environment variables:
        - `SAFE_SOCKET_HANDSHAKE_MS`
        - `SAFE_SOCKET_DEADLINE_MS`
        - `SAFE_SOCKET_HEARTBEAT_MS`

## ✅ Completed (v1.9.0)
- [x] **Infinite Wait Architecture**: Native support for `idleTimeout = 0` (Forever) across TCP, UDP, and SHM.
- [x] **Zombie Connection Parity**: Verified detection/persistence behavior in silent-peer scenarios.
- [x] **CAPI/Python Parity**: Full support for `set_idle_timeout` in the shared library and Python wheel.

## Technical Debt & New Features

- [ ] **Rust Implementation**: Create a native Rust version of the library to eliminate the Go-to-C bridge overhead in high-performance Rust microservices.
- [ ] **Enhanced SHM**: Explore `MONITOR`/`MWAIT` polling in more detail for zero-CPU spin-wait on supported architectures.
- [ ] **NATS Transport**: Evaluate adding a NATS-based transport for pub/sub and distributed messaging patterns.

---

## Research Archive: Low-Latency Execution via Shared Memory

The following notes are preserved for reference from early research into cross-language execution:

- **Goal**: Minimize jitter and context switching.
- **Challenges**: Process isolation (Segmentation Faults), ASLR, NX bit, and runtime (Go/Python) stack management.
- **Conclusion**: OS enforcement of Virtual Memory makes direct code execution from Process A to Process B unsafe/impossible without kernel-level bypasses.