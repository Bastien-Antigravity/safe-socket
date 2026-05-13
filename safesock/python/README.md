# SafeSocket Python SDK

A type-safe, high-performance Python wrapper for the `safe-socket` ecosystem, following the standard **Python Types and Structure** mandate.

## Installation

Ensure `libsafesocket` is built and accessible. You can set the library path via environment variable if it's not in a standard location:

```bash
export LIBSAFESOCKET_PATH=/path/to/safesock/libsafesocket/libsafesocket.so
```

## Usage

```python
from safesocket import safesocket

# 1. Create with standard profile
# Supports 'tcp', 'udp', 'shm', etc.
with safesocket.create("tcp-hello", "localhost:8080", auto_connect=True) as sock:
    sock.send(b"Hello ecosystem")
    response = sock.receive(1024)
    print(f"Received: {response}")

# 2. Advanced configuration (Extended API)
config = safesocket.SocketConfig(
    handshake_timeout_ms=500,
    deadline_ms=1000,
    heartbeat_interval_ms=5000
)

with safesocket.create_with_config("tcp-hello", "localhost:8080", config) as sock:
    sock.open()
    sock.set_idle_timeout(10.5) # Refresh deadline on every I/O
    # ... business logic ...
```

## Testing

Run the scenario tests using `pytest`:

```bash
pytest tests/scenario_test.py
```
