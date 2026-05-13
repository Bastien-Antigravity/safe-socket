# SafeSocket VBA SDK

High-performance network access for Excel, Access, and other Office applications via the `safe-socket` CGO bridge.

## Installation

1. Build the library using `make build-lib`.
2. Copy `libsafesocket.dll` (Windows) or `libsafesocket.dylib` (macOS) to a directory in your system path or the same folder as your Office document.
3. Import `SafeSocket.bas` into your VBA project.

## Usage

```vba
' In a Standard Module
Public Sub Demo()
    Dim h As Long
    ' Connect to a TCP Hello server
    h = SafeSocket_Create("tcp-hello", "127.0.0.1:8080", "", "client", 1)
    
    If h <> -1 Then
        Dim msg As String: msg = "Hello from Excel"
        Dim b() As Byte: b = StrConv(msg, vbFromUnicode)
        
        ' Send first byte reference
        SafeSocket_Send h, b(0), UBound(b) + 1
        
        SafeSocket_Close h
    End If
End Sub
```

## Platform Support

- **Windows**: Supports 64-bit Office (`PtrSafe` and `LongPtr`).
- **macOS**: Supports 64-bit Office using `.dylib`.
