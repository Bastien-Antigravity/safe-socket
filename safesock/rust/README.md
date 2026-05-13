# SafeSocket Rust SDK

A safe, idiomatic Rust wrapper for the `safe-socket` ecosystem.

## Usage

```rust
use safesocket::{SafeSocket, SocketConfig};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let lib_path = "../../libsafesocket/libsafesocket.so"; // adjust extension
    
    // Initialize
    let mut sock = SafeSocket::new(
        "tcp-hello", 
        "localhost:8080", 
        None, 
        "client", 
        true, 
        lib_path
    )?;

    // Send/Receive
    sock.send(b"Hello from Rust")?;
    let data = sock.receive(1024)?;
    println!("Received: {:?}", data);

    Ok(())
}
```

## Setup

Add the following to your `Cargo.toml`:

```toml
[dependencies]
safesocket = { path = "../safesock/rust" }
```
