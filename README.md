# Safe Socket

**Safe Socket** is a high-performance, ultra-low-latency socket library for Go. It provides a reliable abstraction over TCP, UDP, and Shared Memory transports with a flexible profile-based configuration.

## Installation

```bash
go get github.com/Bastien-Antigravity/safe-socket
```

## Features

-   **Modular Transports**: Framed TCP, UDP, Shared Memory (Ring Buffer).
-   **Profiles**: Pre-configured connection strategies (e.g., `tcp-hello`, `tcp`).
-   **Reliability**: Optimized buffers (4MB for UDP) and strict deadline enforcement.
-   **Protocols**: Pluggable protocol execution (Handshake/KeepAlive).

## Usage

### Simple Connection

Use `safesocket.Create` for a zero-boilerplate experience.

```go
package main

import (
	"log"

	"github.com/Bastien-Antigravity/safe-socket"
	"github.com/Bastien-Antigravity/safe-socket/src/schemas"
)

func main() {
    // Connect using the 'tcp-hello' profile
    // 1. Profile Name
    // 2. Destination Address
    // 3. Your Public IP (for protocol handshake)
	client, err := safesocket.Create("tcp-hello", "127.0.0.1:8081", "203.0.113.10")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Send a Message (Protobuf)
	msg := &schemas.HelloMsg{
		Name: "Alice",
		Host: "localhost",
	}
	if err := client.Send(msg); err != nil {
		log.Printf("Send error: %v", err)
	}

    // Receive a Message
    var response schemas.HelloMsg
    if err := client.Receive(&response); err != nil {
        log.Printf("Receive error: %v", err)
    }
}
```

### Supported Profiles

| Profile Name | Transport | Protocol | Description |
| :--- | :--- | :--- | :--- |
| `"tcp-hello"` | TCP (Framed) | SayHello | Establishes connection and performs a Hello handshake. |
| `"tcp"` | TCP (Framed) | None | Raw persistent TCP connection. |

## Configuration

The library uses a default connection timeout of **5 seconds**. This can be customized by using the `SocketProfile` interface directly if you need advanced configuration.

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## License

[MIT](https://choosealicense.com/licenses/mit/)
