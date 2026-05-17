# Architecture: C++ SDK

The C++ SDK is designed as a modern, header-only wrapper around the C library.

## Design Patterns

- **RAII Lifecycle**: The `SafeSocket` and `SafeSocketConnection` classes manage handles using class destructors. `std::unique_ptr` is recommended for managing class instances to ensure deterministic cleanup.
- **Exception-based Errors**: C-style return codes are converted into `SafeSocketError` (derived from `std::runtime_error`), simplifying error handling in complex application logic.
- **Buffer Management**: Uses `std::vector<uint8_t>` for data exchange, providing a familiar and safe way to handle binary packets.

## Dependencies

- **C++14**: Required for `std::make_unique` and other modern primitives.
- **libsafesocket**: The shared library must be linked at compile-time or available in the library path at runtime.
