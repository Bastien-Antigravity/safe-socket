# ⚡ AI Initialization: safe-socket

> [!IMPORTANT] MANDATORY INITIALIZATION
> Copy and paste this prompt when starting a new session in this repository:
> 
> *"1. Read the ecosystem map in **[[00-Master-MOC]]**."*
> *"2. Load project constraints from **[[AI-Project-DNA]]**."*
> *"3. Restore session state from **[[AI-Session-State]]**."*
> *"4. **Audit**: Run `git branch --show-current` and `cat VERSION.txt` to verify state matches the Session State."*
> *"5. **Spec Gate**: Before implementing any feature, you MUST read the behavioral spec in `business-bdd-brain`."*

## 🛡️ Architectural Guardrails (v1.9.0+)
- **Infinite Wait (0)**: Setting `idleTimeout = 0` MUST clear system deadlines ONCE. Do NOT call `SetDeadline` in the `Read/Write` hot paths when `idleTimeout == 0` (Performance Critical).
- **Heartbeat Synergy**: `HeartbeatConnection` automatically stops/restarts its ticker when `SetIdleTimeout` is called.
