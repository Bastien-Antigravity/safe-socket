# Architecture: VBA SDK

The VBA SDK provides direct access to the `libsafesocket` API using Win32-style `Declare` statements.

## Design Patterns

- **Flat Handle API**: Unlike other languages, VBA uses the flat C API directly. Users are responsible for calling `SafeSocket_Close` to prevent handle leaks in the Go registry.
- **Pointer Safety**: Uses `PtrSafe` and `LongPtr` to ensure compatibility with 64-bit versions of Microsoft Office.
- **Byte Array Marshalling**: Data is passed to Go by passing the first element of a Byte array `ByRef`, allowing CGO to access the memory buffer directly.

## Memory Management

- **External Error Handling**: Since VBA lacks exceptions for DLL calls, users should check return codes (-1) and call `SafeSocket_GetSocketError` to retrieve descriptive error messages.
- **String Handling**: VBA `String` is passed as a null-terminated ANSI string to the C bridge.
