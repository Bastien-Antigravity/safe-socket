# Safe Socket

**Safe Socket** is a high-performance, robust socket library for Go. It provides a reliable abstraction over **TCP**, **UDP**, and **Shared Memory (SHM)** transports with a flexible, profile-based configuration system.

## Version
Current Version: `v1.3.0`

## Installation

```bash
go get github.com/Bastien-Antigravity/safe-socket
```

## Features

-   **Modular Transports**:
-   **Modular Transports**:
    -   **Framed TCP**: Reliable, persistent connections with message framing. Optimizes `Read()` via buffering to support safe buffer pooling (prevents header loss on short reads).
    -   **UDP**: High-speed, connectionless communication with optional reliability layers.
    -   **Shared Memory (SHM)**: Ultra-low latency IPC for local processes using memory-mapped files (Ring Buffer).
-   **Intelligent Protocols**:
    -   **Hello Protocol**: Identity exchange handshake.
    -   **Stateless Envelope (UDP)**: Zero-handshake authentication where every packet carries the sender's identity and payload.
-   **Unified Facade**: Interact with any transport using `Open()`, `Close()`, `Send()`, `Receive()`, and `Accept()`.

## Usage

### Zero-Boilerplate Creation

Use `safesocket.Create` to instantiate and connect in one line.

```go
package main

import (
    "log"
    "github.com/Bastien-Antigravity/safe-socket"
)

func main() {
    // Example: Connect to a server using TCP with Hello Handshake
    // publicIP is required for the handshake identity.
    // socketType: "client" or "server"
    socket, err := safesocket.Create("tcp-hello", "127.0.0.1:9000", "192.168.1.50", "client", true)
    if err != nil {
        log.Fatal(err)
    }
    defer socket.Close()

	// Send Data
    socket.Send([]byte("Hello Server!"))

    // Receive Data (Dynamic Buffer)
    // Receive() automatically allocates the correct size.
    msg, _ := socket.Receive()
    log.Printf("Received: %s", string(msg))

    // Alternative: Use Read() for fixed buffers (io.Reader compliant)
    // buf := make([]byte, 1024)
    // n, _ := socket.Read(buf)
}
```

### Supported Profiles

| Profile | Transport | Protocol | Address Format | Behavior |
| :--- | :--- | :--- | :--- | :--- |
| `"tcp"` | TCP | None | `IP:Port` | Raw TCP stream. |
| `"tcp-hello"` | TCP | Hello | `IP:Port` | TCP + Identity Handshake. |
| `"udp"` | UDP | None | `IP:Port` | Raw UDP packets. |
| `"udp-hello"` | UDP | Hello | `IP:Port` | **Stateless Envelope**: Wraps every packet with Identity + Payload. |
| `"shm"` | SHM | None | File Path | Raw Memory Mapped File. |
| `"shm-hello"` | SHM | Hello | File Path | SHM + Identity Handshake. |

### Protocol Details

-   **Hello Handshake (TCP/SHM)**: Upon connection, the client sends a `HelloMsg` (Name, Host, IP, **Dynamic Addresses**). The library automatically resolves local and remote addresses to provide full network observability. The server verifies this before allowing data exchange.
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

You can access the metadata exchanged during the Hello Handshake (e.g., Peer Name, Public IP) by type-asserting the connection.

**For TCP / SHM:**

```go
conn, _ := server.Accept()
// Check if it's a HandshakeConnection
if hc, ok := conn.(*facade.HandshakeConnection); ok {
    fmt.Printf("Connected Peer: %s (IP: %s)\n", hc.Identity.FromName(), hc.Identity.FromPublicIP())
}
```

**For UDP:**

```go
conn, _ := server.Accept()
// Read essential to receive packet and populate identity
conn.Read(buf) 

if ec, ok := conn.(*facade.EnvelopedConnection); ok {
    fmt.Printf("Last Packet From: %s\n", ec.LastIdentity.FromName())
}
```

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## License
[MIT](https://choosealicense.com/licenses/mit/)
