---
microservice: safe-socket
type: repository
status: active
language: go
tags:
  - domain/networking
---

# Safe Socket

**Safe Socket** is a high-performance, robust socket library for Go. It provides a reliable abstraction over **TCP**, **UDP**, and **Shared Memory (SHM)** transports with a flexible, profile-based configuration system.

## Installation

```bash
go get github.com/Bastien-Antigravity/safe-socket
```

## Features

-   **Modular Transports**:
-   **Modular Transports**:
    -   **Framed TCP**: Reliable, persistent connections with message framing. Optimizes `Read()` via buffering to support safe buffer pooling (prevents header loss on short reads). 
    -   **Heartbeat Support**: Automatically handles 0-length frames as heartbeats. `Read()` and `ReadMessage()` return `n=0` / `empty buffer` when a heartbeat arrives, allowing application loops to refresh deadlines.
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
    // Set a Deadline for the read operation
    socket.SetReadDeadline(time.Now().Add(2 * time.Second))
    
    msg, err := socket.Receive()
    if err != nil {
        // Handle timeout
        log.Printf("Receive failed: %v", err)
    } else {
        log.Printf("Received: %s", string(msg))
    }

    // Alternative: Use Read() for fixed buffers (io.Reader compliant)
    // buf := make([]byte, 1024)
    // n, _ := socket.Read(buf)
}
```

### Advanced Creation

For more control (e.g., setting a default deadline), use `safesocket.CreateWithConfig`:

```go
config := models.SocketConfig{
    PublicIP: "1.2.3.4",
    Deadline: 5 * time.Minute, // Idle Timeout: Connection stays alive as long as active
}

// Note: Use Deadline: 0 for a completely open (infinite) connection.

// CreateWithConfig(profile, address, config, type, autoConnect)
socket, err := safesocket.CreateWithConfig("tcp-hello", "127.0.0.1:9000", config, "server", true)
if err != nil {
    log.Fatal(err)
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

### Compound Profiles (Identity Injection)

You can specify a custom identity name for any protocol-aware profile by using the syntax `[profile]:[name]`.
For example, `tcp-hello:my-service` will use the `tcp-hello` transport but identify itself as `my-service` during the handshake. If no name is provided, default generic names are used.

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

You can access the metadata exchanged during the Hello Handshake (e.g., Peer Name, Hostname, IP) by using the unified `safesocket.GetIdentity` helper. This works for both session-based (TCP/SHM) and stateless (UDP) connections without needing to import internal packages.

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


## Python Bindings

`safe-socket` is also available as a Python library, providing the same high-level API.

### Installation

You can install it directly from the GitHub repository:

```bash
pip install git+https://github.com/Bastien-Antigravity/safe-socket.git#egg=safe-socket&subdirectory=python
```

*Note: Ensure you have Go installed on your system as it is required to compile the underlying shared library during installation.*


Or download a pre-built wheel from [GitHub Releases](https://github.com/Bastien-Antigravity/safe-socket/releases) and install it:

```bash
# Example for a downloaded wheel
pip install safe_socket-<VERSION>-py3-none-any.whl
```

### Usage Example

```python
from safesocket import safesocket

# Create and open a client
with safesocket.create(profile="tcp-hello", address="127.0.0.1:9000", public_ip="1.2.3.4") as client:
    client.open()
    client.send(b"Hello from Python!")
    response = client.receive()
    print(f"Received: {response.decode()}")

# Server side
server = safesocket.create(profile="tcp", address="0.0.0.0:9000", socket_type="server")
server.listen()
conn = server.accept()
with conn:
    data = conn.receive()
    conn.send(b"Echo: " + data)
server.close()
```

