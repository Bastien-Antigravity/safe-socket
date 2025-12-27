# Safe Socket

**Safe Socket** is a high-performance, ultra-low-latency socket library for Go. It provides a reliable abstraction over TCP, UDP, and Shared Memory transports with a flexible profile-based configuration.

## Installation

```bash
go get github.com/Bastien-Antigravity/safe-socket
```

## Features

-   **Modular Transports**: Framed TCP, UDP, Shared Memory (Ring Buffer).
-   **Profiles**: Pre-configured connection strategies (e.g., `tcp-hello`, `tcp`).
-   **Lifecycle Management**: Explicit `Open()` and `Close()` methods via the `Socket` interface.
-   **Reliability**: Optimized buffers and strict deadline enforcement.
-   **Protocols**: Pluggable protocol execution (Handshake/KeepAlive).

## Usage

### Simple Connection

Use `safesocket.Create` for a zero-boilerplate experience. This returns an **already opened** `Socket` interface.

```go
package main

import (
	"log"

	"github.com/Bastien-Antigravity/safe-socket"
)

func main() {
    // Connect using the 'tcp-hello' profile
    // Returns a Socket interface
	client, err := safesocket.Create("tcp-hello", "127.0.0.1:8081", "203.0.113.10")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Send raw data
	data := []byte("Hello Safe Socket")
	if err := client.Send(data); err != nil {
		log.Printf("Send error: %v", err)
	}

    // Receive raw data
    buf := make([]byte, 1024)
    n, err := client.Receive(buf)
    if err != nil {
        log.Printf("Receive error: %v", err)
    }
    log.Printf("Received: %s", string(buf[:n]))
}
```

### Advanced Lifecycle (Reconnection)

Since `Socket` is a facade, you can Close and Re-open the connection without recreating the object.

```go
// Close the current connection
client.Close()

// Re-open using the same profile configuration
if err := client.Open(); err != nil {
    log.Fatalf("Reconnect failed: %v", err)
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

Project Link: [https://github.com/Bastien-Antigravity/safe-socket](https://github.com/Bastien-Antigravity/safe-socket)

## License

[MIT](https://choosealicense.com/licenses/mit/)
