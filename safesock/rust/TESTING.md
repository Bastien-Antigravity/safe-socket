# Testing: Rust SDK

## Prerequisites

1. Build the Go library:
   ```bash
   cd ../..
   make build-lib
   ```
2. Install Rust toolchain.

## Running Examples

From the `safesock/rust` directory:

```bash
cargo run --example basic_usage
```

## Coverage

- **Initialization**: Ensures the shared library can be found and symbols resolved.
- **Error Propagation**: Verifies that Go-level errors are successfully converted into Rust `std::error::Error`.
- **Handle Tracking**: Tests that handles are correctly managed and released.
