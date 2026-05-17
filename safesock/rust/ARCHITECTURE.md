# Architecture: Rust SDK

The Rust SDK provides a memory-safe wrapper around the `libsafesocket` C interface.

## Design Patterns

- **RAII Lifecycle**: The `SafeSocket` struct implements the `Drop` trait. When the socket goes out of scope, it automatically calls `SafeSocket_Close` in the Go bridge.
- **Dynamic Loading**: Uses the `libloading` crate to load the shared library at runtime. This avoids complex linker configurations and allows for flexible library path resolution.
- **Safe I/O**: Byte slices (`&[u8]`) are converted to raw pointers only at the FFI boundary, maintaining Rust's safety guarantees throughout the application logic.

## Memory Management

- **Leaked Library Handle**: The `Library` object is leaked (`Box::leak`) upon the first initialization. This is a deliberate choice because the Go runtime cannot be safely unloaded via `dlclose` once initialized; attempting to do so can cause deadlocks or segmentation faults.
