# 🧬 Project DNA: safe-socket

## 🎯 High-Level Intent (BDD)
- **Goal**: Provide a rock-solid, framed TCP transport layer with built-in heartbeat and automatic reconnection logic.
- **Key Pattern**: **Length-Prefixed Framing** (using Cap'n Proto for serialization) and **Non-Blocking I/O**.
- **Behavioral Source of Truth**: [[business-bdd-brain/02-Behavior-Specs/safe-socket]]
- **Spec Gate**: [HARDENED] No implementation without an `approved` spec in the folder above.

## 🛠️ Role Specifics
- **Architect**: 
    - Ensure zero-copy memory management where possible (mmap support).
    - Maintain strict separation between transport logic and application payload.
- **QA**: 
    - Focus on network edge cases: network splits, silent peers (Zombies), and high-latency environments.
    - Mandatory verification of "Infinite Wait" (timeout=0) behavior.
- **Developer**:
    - Ensure Python and C wrappers remain in 1:1 parity with the Go core.

## 🚦 Lifecycle & Versioning
- **Primary Branch**: `develop`
- **Protected Branches**: `main`, `master`
- **Versioning Strategy**: Semantic Versioning (vX.Y.Z).
- **Version Source of Truth**: `VERSION.txt`.
