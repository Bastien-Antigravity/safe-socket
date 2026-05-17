# Testing: C++ SDK

## Prerequisites

1. Build the shared library:
   ```bash
   cd ../..
   make build-lib
   ```
2. A C++14 compatible compiler (g++, clang++).

## Running Examples

From the `safesock/cpp/examples` directory:

```bash
make
DYLD_LIBRARY_PATH=../../libsafesocket ./basic_usage
```

## Coverage

- **Symbol Resolution**: Verifies that the compiler can find and link all `SafeSocket_` exports.
- **Exception Handling**: Ensures that bridge-level errors are correctly caught and reported via `try-catch` blocks.
- **Handle Lifecycle**: Validates that destructors correctly close Go-side sockets.
