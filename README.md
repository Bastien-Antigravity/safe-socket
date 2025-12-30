# Safe Socket

**Safe Socket** is a high-performance, robust socket library for Go. It provides a reliable abstraction over **TCP**, **UDP**, and **Shared Memory (SHM)** transports with a flexible, profile-based configuration system.

## Version
Current Version: `v1.1.2`

## Installation

```bash
go get github.com/Bastien-Antigravity/safe-socket
```

## Features

-   **Modular Transports**:
    -   **Framed TCP**: Reliable, persistent connections with message framing.
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
    socket, err := safesocket.Create("tcp-hello", "127.0.0.1:9000", "192.168.1.50", safesocket.SocketTypeClient, true)
    if err != nil {
        log.Fatal(err)
    }
    defer socket.Close()

    // Send Data
    socket.Send([]byte("Hello Server!"))

    // Receive Data
    buf := make([]byte, 1024)
    n, _ := socket.Read(buf)
    log.Printf("Received: %s", string(buf[:n]))
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

-   **Hello Handshake (TCP/SHM)**: Upon connection, the client sends a `HelloMsg` (Name, Host, IP). The server verifies it before allowing data exchange.
-   **Stateless Envelope (UDP)**: Since UDP is connectionless, there is no "session". When using `udp-hello`, the library automatically wraps **every** packet in a lightweight `PacketEnvelope` (Sender Name + Payload). The server transparently unwraps this, so implementation code just sees the payload and knows the sender is verified.

## Advanced Usage

### Server Example

```go
func runServer() {
    // Create a UDP Server handling enveloped packets
    server, _ := safesocket.Create("udp-hello", "0.0.0.0:9000", "", safesocket.SocketTypeServer, true)
    defer server.Close()

    for {
        // Accept blocks until a packet arrives.
        // For UDP, this returns a "Transient Socket" representing that specific packet/sender.
        conn, err := server.Accept()
        if err != nil { continue }

        go func(c safesocket.Socket) {
            defer c.Close()
            
            // Read the payload (Decapsulation happens automatically)
            buf := make([]byte, 1024)
            n, _ := c.Read(buf)
            
            // Reply (Encapsulation happens automatically)
            c.Write([]byte("Message Received: " + string(buf[:n])))
        }(conn)
    }
}
```

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## License
[MIT](https://choosealicense.com/licenses/mit/)
