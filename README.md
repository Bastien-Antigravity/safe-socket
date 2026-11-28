---
microservice: safe-socket
type: repository
status: active
language: go
tags:
- '#service/safe-socket'
- '#domain/networking'
- '#zone/3-fleet'
- '#type/repository'
- '#state/active'
---

# Safe Socket

**Safe Socket** is a high-performance, robust socket library for Go. It provides a reliable abstraction over **TCP**, **TLS**, **UDP**, and **Shared Memory (SHM)** transports with a flexible, profile-based configuration system.

## Installation

```bash
go get github.com/Bastien-Antigravity/safe-socket
```

## Features & Capabilities
For a detailed breakdown of this library's features, polyglot SDKs, and BDD behavior specifications, refer to the **[🚀 Features & Behavior Guide](quick-overview/Features-Behavior.md)**.

## Usage

### Zero-Boilerplate Creation

Use `safesocket.CreateClient`, `safesocket.CreateServer`, or `safesocket.Create` to instantiate and connect in one line.

```go
import (
	"log"
	"time"

	"github.com/Bastien-Antigravity/safe-socket"
)

func main() {
    // Role-segregated client socket (recommended)
    client, err := safesocket.CreateClient("tcp-hello:my-service", "127.0.0.1:9000", "", true)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

	// Send Data
    client.Send([]byte("Hello Server!"))

    // Receive Data (Dynamic Buffer)
    // Receive() automatically allocates the correct size.
    // Set a Deadline for the read operation
    client.SetReadDeadline(time.Now().Add(2 * time.Second))
    
    msg, err := client.Receive()
    if err != nil {
        // Handle timeout
        log.Printf("Receive failed: %v", err)
    } else {
        log.Printf("Received: %s", string(msg))
    }

    // Alternative: Use Read() for fixed buffers (io.Reader compliant)
    // buf := make([]byte, 1024)
    // n, _ := client.Read(buf)

    // Update Idle Timeout at runtime (0 disables timeouts and waits 'forever')
    client.SetIdleTimeout(0)
}
```

### Advanced Creation

For more control (e.g., setting a custom deadline or disabling timeouts upfront), use `safesocket.CreateClientWithConfig`, `safesocket.CreateServerWithConfig`, or `safesocket.CreateWithConfig`:

```go
config := safesocket.SocketConfig{
    PublicIP: "1.2.3.4",
    Deadline: 5 * time.Minute, // Idle Timeout: Connection stays alive as long as active
}

// Note: Set DisableTimeout: true (or Deadline: -1) for an infinite (forever) connection
// across TCP, TLS, UDP, and Shared Memory transports.
// Leaving Deadline: 0 applies transport- and deployment-mode-aware hybrid defaults.

server, err := safesocket.CreateServerWithConfig("tcp-hello", "127.0.0.1:9000", config, true)
if err != nil {
    log.Fatal(err)
}
```

### Supported Profiles

| Profile | Transport | Protocol | Address Format | Behavior |
| :--- | :--- | :--- | :--- | :--- |
| `"auto-hello"`, `"auto"` | **Auto (TCP / TLS)** | Hello | `IP:Port` | **Intelligent Auto-Encryption**: Evaluates target address using `MachineDetector`. Connects via unencrypted Framed TCP if local machine; automatically enables TLS if destination is on a remote machine. |
| `"tcp"` | Framed TCP | None | `IP:Port` | Length-prefixed TCP stream (4-byte Big-Endian header + payload). |
| `"tcp-hello"` | Framed TCP | Hello | `IP:Port` | Framed TCP + Cap'n Proto Identity Handshake. |
| `"tls"` | Framed TLS | None | `IP:Port` | Length-prefixed TLS stream (4-byte Big-Endian header + payload). |
| `"tls-hello"` | Framed TLS | Hello | `IP:Port` | Framed TLS + Cap'n Proto Identity Handshake. |
| `"udp"` | UDP | None | `IP:Port` | Raw UDP datagrams (up to 64KB, demultiplexed per remote peer on server). |
| `"udp-hello"` | UDP | Hello | `IP:Port` | **Stateless Envelope**: Wraps every datagram with Identity + Payload. |
| `"shm"` | SHM | None | File Path | Framed Memory-Mapped Ring Buffer IPC. |
| `"shm-hello"` | SHM | Hello | File Path | SHM + Cap'n Proto Identity Handshake (`shm-hello:<identity>`). |

### 🔒 Intelligent Auto-Encryption (`auto-hello`)

When using profile `"auto-hello"` (or shorthand `"auto"`), `safe-socket` removes the need to manually choose between `tcp-hello` and `tls-hello`:
- **Same Machine / Local Network Interface**: Detects loopback (`127.0.0.0/8`, `::1`, `localhost`), hostnames, and any active host network interface IP (via `net.InterfaceAddrs()` cached in `MachineDetector`). Uses high-performance unencrypted framed TCP with zero TLS CPU overhead.
- **Remote Machine**: Automatically engages TLS encryption with certificate/CA verification for cross-machine communication (LAN, WAN, or Cloud).

### 📡 Dynamic Service Discovery Address Advertising

When connecting to an ecosystem registry (such as `config-server`), a service can advertise its real inbound listening port rather than the OS-assigned ephemeral client port:

```go
config := safesocket.SocketConfig{
    ServiceAddress: "127.0.0.2:1026", // Real inbound listening address
}
socket, err := safesocket.CreateWithConfig("auto-hello:notif-server", "127.0.0.2:3306", config, "client", true)
```

The `ServiceAddress` is transmitted in `HelloMsg.FromAddress` during handshake, allowing servers and sibling services to dynamically route to this service.

### Compound Profiles (Identity Injection)

You can specify a custom identity name for any protocol-aware profile by using the syntax `[profile]:[name]`.
For example, `tcp-hello:my-service` will use the `tcp-hello` transport but identify itself as `my-service` during the handshake. If no name is provided, the library injects generic identifies (e.g., `TcpClient-Generic`).

### Unified Hybrid Timeouts & Handshake Defaults

If `SocketConfig.Deadline` is left at `0` (and `DisableTimeout` is `false`), `Create` / `CreateWithConfig` applies `ResolveHybridTimeouts` automatically alongside fail-fast connect/handshake timeouts:

| Condition | Handshake Timeout | Client Idle Timeout | Client Heartbeat (`Deadline / 2.5`) | Server Idle Timeout |
| :--- | :--- | :--- | :--- | :--- |
| **Local TCP/TLS** | 200ms | 60s | 24s | 10m (server outbound HB off by default) |
| **Docker TCP/TLS (`DOCKER_ENV=true`)** | 500ms (remote) / 200ms (local) | 30s | 12s | 3m (server outbound HB off by default) |
| **UDP** | 500ms / 200ms | 5s | Disabled | 5s |
| **Shared Memory (SHM)** | 100ms | Infinite (`DisableTimeout: true`) | Disabled | Infinite (`DisableTimeout: true`) |

### Heartbeat Optimization & Thresholds

To maximize performance, heartbeats are **automatically disabled** if the `IdleTimeout` (Deadline) is set below these thresholds:
-   **Network**: < 300ms
-   **Local**: < 150ms
-   **SHM**: < 50ms

When heartbeats are disabled due to these thresholds, a warning is printed to `stdout` to notify that the connection will close if genuine data is not transmitted within the window.

> [!TIP]
> Use `safesocket.CreateWithConfig` to override these defaults if your environment requires more latency headroom.

### Protocol Details

-   **Hello Handshake (TCP/TLS/SHM)**: Upon connection, the client sends a `HelloMsg` (Name, Host, IP, **Dynamic Addresses**). The library automatically resolves local and remote addresses to provide full network observability. The server verifies this before allowing data exchange.
-   **Stateless Envelope (UDP)**: Since UDP is connectionless, there is no "session". When using `udp-hello`, the library automatically wraps **every** packet in a lightweight `PacketEnvelope` (Sender Name + Payload). The server transparently unwraps this, so implementation code just sees the payload and knows the sender is verified.

## Advanced Usage

### Server Example

```go
func runServer() {
    // Create a UDP Server handling enveloped packets
    server, _ := safesocket.Create("udp-hello", "0.0.0.0:9000", "", "server", true)
    defer server.Close()

    for {
        // Accept blocks until a packet arrives.
        // For UDP, this returns a "Transient Socket" representing that specific packet/sender.
        // Returns interfaces.TransportConnection
        conn, err := server.Accept()
        if err != nil { continue }

        go func(c interfaces.TransportConnection) {
            defer c.Close()
            
            // Read the payload (Decapsulation happens automatically)
            // Use ReadMessage() for dynamic allocation
            msg, _ := c.ReadMessage()
            
            // Reply (Encapsulation happens automatically via Write)
            c.Write([]byte("Message Received: " + string(msg)))
        }(conn)
    }
}
```

### Accessing Peer Identity

You can access the metadata exchanged during the Hello Handshake (e.g., Peer Name, Hostname, IP) by using the unified `safesocket.GetIdentity` helper. This works for both session-based (TCP/TLS/SHM) and stateless (UDP) connections without needing to import internal packages.

```go
conn, _ := server.Accept()

// GetIdentity automatically handles all wrapper types
identity := safesocket.GetIdentity(conn)
if identity != nil {
    fmt.Printf("Connected Peer: %s (IP: %s)\n", identity.FromName(), identity.FromPublicIP())
}
```

> [!NOTE]
> For UDP (`udp-hello`), the identity is only available **after** the first packet has been successfully read, as it is extracted from the packet envelope.

### 📡 Lock-Free Real-Time Telemetry Wrapper (WireTap)

`safe-socket` includes a built-in, zero-mutex telemetry wrapper (`src/facade/telemetry_wrapper.go`) used by `microservice-toolbox` WireTap to track live inter-service TCP/TLS/UDP/SHM connections and segregated logical channels in real time:

```go
// Enable or disable the atomic telemetry wrapper at runtime (zero mutexes)
safesocket.SetTelemetryEnabled(true)

// Register a callback hook for CONNECT, ACCEPT, SEND, RECV, and CLOSE events
safesocket.SetTelemetryHook(func(ev safesocket.TelemetryEvent) {
    // ev.EventType, ev.Profile, ev.LocalService, ev.PeerService, ev.RemoteAddr, ev.Channel, ev.PayloadBytes
})
```

When disabled (`safesocket.SetTelemetryEnabled(false)`), the wrapper short-circuits with a single `atomic.Bool` load and zero heap allocations.


## Polyglot SDK Bindings

The core Go engine is exposed via a CGO shared library (`libsafesocket`), providing high-level, object-oriented wrappers for other languages. The bindings are maintained in the [safesock/](safesock) folder.

### 🐍 Python SDK ([safesock/python](safesock/python))

```python
from safesocket import safesocket

# Create a client and send/receive data
with safesocket.create(profile_name="tcp-hello", address="127.0.0.1:9000") as client:
    client.open()
    client.send(b"Hello from Python!")
    response = client.receive()
    print(f"Received: {response.decode()}")
```

### 🦀 Rust SDK ([safesock/rust](safesock/rust))

```rust
use safesocket::SafeSocket;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let mut client = SafeSocket::new("tcp-hello", "127.0.0.1:9000", None, "client", true, "libsafesocket.so")?;
    client.send(b"Hello from Rust")?;
    let response = client.receive(1024)?;
    println!("Received: {:?}", response);
    Ok(())
}
```

### 💧 C++ SDK ([safesock/cpp](safesock/cpp))

```cpp
#include "SafeSocket.hpp"

int main() {
    try {
        auto client = safesock::create("tcp-hello", "127.0.0.1:9000", "", "client", true);
        client->send({ 'H', 'e', 'l', 'l', 'o' });
        auto response = client->receive(1024);
    } catch (const safesock::SafeSocketError& e) {
        std::cerr << "Error: " << e.what() << std::endl;
    }
    return 0;
}
```

### 📊 VBA SDK ([safesock/vba](safesock/vba))

```vba
' VBA example to connect and send data
Dim handle As Long
handle = SafeSocket_Create("tcp-hello", "127.0.0.1:9000", "", "client", 1)
If handle <> -1 Then
    Dim data() As Byte
    data = StrConv("Hello from VBA", vbFromUnicode)
    SafeSocket_Send handle, data(0), UBound(data) + 1
    SafeSocket_Close handle
End If
```

### Rebuilding the CGO Bridge

If the Go source code or API changes, you must rebuild the shared library:

```bash
make build-lib
```

This generates `libsafesocket.so`, `libsafesocket.dylib`, or `libsafesocket.dll` under `safesock/libsafesocket/`.
