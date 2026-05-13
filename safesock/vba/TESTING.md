# Testing: VBA SDK

## Prerequisites

1. Build the library:
   ```bash
   make build-lib
   # or for Windows from macOS/Linux:
   make build-dll
   ```

## Running Tests

1. Open Microsoft Excel.
2. Press `Alt + F11` to open the VBA Editor.
3. `File > Import File...` and select `SafeSocket.bas`.
4. Run the `DemoSafeSocket` macro.
5. Check the `Immediate Window` (`Ctrl + G`) for output logs.

## Coverage

- **DLL Loading**: Confirms `libsafesocket` can be located by the Office process.
- **Marshalling**: Verifies that Byte arrays and Strings are correctly passed to Go.
- **Handle Lifecycle**: Ensures that sockets can be opened and closed without crashing the host application.
