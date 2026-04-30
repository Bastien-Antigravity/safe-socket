# ⚡ AI Initialization: safe-socket

> [!IMPORTANT] MANDATORY INITIALIZATION
> Copy and paste this prompt when starting a new session in this repository:
> 
> *"Read the ecosystem map in **[[00-Master-MOC]]** and restore session state from **[[safe-socket/AI-Session-State]]**. Follow the standardized loop in **[[00-Daily-AI-Playbook]]**."*

## 🛡️ Architectural Guardrails (v1.9.0+)
- **Infinite Wait (0)**: Setting `idleTimeout = 0` MUST clear system deadlines ONCE. Do NOT call `SetDeadline` in the `Read/Write` hot paths when `idleTimeout == 0` (Performance Critical).
- **Heartbeat Synergy**: `HeartbeatConnection` automatically stops/restarts its ticker when `SetIdleTimeout` is called.
