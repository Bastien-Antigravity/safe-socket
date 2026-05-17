use safesocket::SafeSocket;

fn main() {
    let lib_name = if cfg!(target_os = "macos") {
        "libsafesocket.dylib"
    } else if cfg!(target_os = "windows") {
        "libsafesocket.dll"
    } else {
        "libsafesocket.so"
    };

    // Assuming we run from safesock/rust
    let lib_path = format!("../libsafesocket/{}", lib_name);
    
    match SafeSocket::new("tcp-client", "localhost:8080", None, "client", false, &lib_path) {
        Ok(_sock) => {
            println!("SafeSocket created successfully");
        }
        Err(e) => {
            // It might fail with "connection refused" or similar if no server,
            // but "unknown profile" would also prove it works.
            eprintln!("Result: {}", e);
        }
    }
}
