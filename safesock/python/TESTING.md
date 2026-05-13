# Testing: Python SDK

## Prerequisites

1. Build the shared library:
   ```bash
   cd ../..
   make build-lib
   ```
2. Ensure `pytest` is installed.

## Running Tests

From the `safesock/python` directory:

```bash
pytest tests/scenario_test.py
```

## Coverage

- **Lifecycle**: Verifies creation, opening, and context manager cleanup.
- **Custom Config**: Tests the Extended API with tight handshake and I/O deadlines.
- **Resilience**: Confirms that Go-level timeouts are correctly propagated as `SafeSocketError` in Python.
- **Protocol Parity**: Validates the `tcp-hello` handshake across the FFI.
