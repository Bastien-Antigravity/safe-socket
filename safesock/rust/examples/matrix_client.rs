use safesocket::{SafeSocket, SocketConfig};
use std::env;
use std::process;
use std::time::Instant;

fn main() {
    let args: Vec<String> = env::args().collect();
    let addr = if args.len() > 1 { &args[1] } else { "127.0.0.1:9999" };

    let lib_name = if cfg!(target_os = "macos") {
        "libsafesocket.dylib"
    } else if cfg!(target_os = "windows") {
        "libsafesocket.dll"
    } else {
        "libsafesocket.so"
    };

    let possible_paths = vec![
        format!("./safesock/libsafesocket/{}", lib_name),
        format!("../libsafesocket/{}", lib_name),
        format!("../../libsafesocket/{}", lib_name),
        format!("./libsafesocket/{}", lib_name),
    ];

    let mut lib_path = String::new();
    for path in possible_paths {
        if std::path::Path::new(&path).exists() {
            lib_path = path;
            break;
        }
    }

    if lib_path.is_empty() {
        eprintln!("Rust Client: ERROR - Could not find libsafesocket");
        process::exit(1);
    }

    println!("Rust Client: Connecting to {}", addr);

    let public_ip = "1.2.3.4-rust";
    let config = SocketConfig {
        public_ip: public_ip.to_string(),
        ..Default::default()
    };

    match SafeSocket::new("tcp-hello:rust-matrix", addr, Some(config), "client", true, &lib_path) {
        Ok(sock) => {
            println!("Rust Client: Connected");
            
            // --- TEST 1: Basic Ping ---
            let payload = "ping-rust";
            sock.send(payload.as_bytes()).expect("Send failed");
            let response = sock.receive(1024).expect("Receive failed");
            let decoded = String::from_utf8_lossy(&response);
            if decoded != format!("echo:{}", payload) {
                eprintln!("Rust Client: FAILURE - Basic ping failed. Got: {}", decoded);
                process::exit(1);
            }

            // --- TEST 2: Metadata Verification ---
            println!("Rust Client: Verifying metadata...");
            sock.send(b"meta_request").expect("Send failed");
            let response = sock.receive(1024).expect("Receive failed");
            let decoded = String::from_utf8_lossy(&response);
            if !decoded.starts_with("meta:") {
                eprintln!("Rust Client: FAILURE - Metadata request failed");
                process::exit(1);
            }
            
            let parts: Vec<&str> = decoded[5..].split(',').collect();
            if parts[0] != "rust-matrix" || parts[2] != public_ip {
                eprintln!("Rust Client: FAILURE - Metadata mismatch. Expected name=rust-matrix, ip={}. Got: {}", public_ip, decoded);
                process::exit(1);
            }
            println!("Rust Client: Metadata verified (Host reported: {})", parts[1]);

            // --- TEST 3: Large Payload (1MB) ---
            println!("Rust Client: Testing 1MB payload...");
            let large_payload = vec![b'R'; 1024 * 1024];
            let start = Instant::now();
            sock.send(&large_payload).expect("Send failed");
            
            let response = sock.receive(1024 * 1024 + 10).expect("Receive failed");
            let duration = start.elapsed();
            
            if response.len() != large_payload.len() + 5 || !response.starts_with(b"echo:") {
                eprintln!("Rust Client: FAILURE - Large payload corrupted. Length={}", response.len());
                process::exit(1);
            }
            
            println!("Rust Client: 1MB payload verified in {:?}", duration);
            
            println!("Rust Client: ALL TESTS SUCCESS");
            process::exit(0);
        }
        Err(e) => {
            eprintln!("Rust Client: ERROR - Failed to create socket: {}", e);
            process::exit(1);
        }
    }
}
