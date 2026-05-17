import sys
import os
import time
import socket

# Add parent directory to sys.path to find the safesocket package
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from safesocket import safesocket

def run_client(addr):
    print(f"Python Client: Connecting to {addr}")
    try:
        # Create client with tcp-hello profile and custom metadata
        hostname = socket.gethostname()
        public_ip = "1.2.3.4-py"
        
        config = safesocket.SocketConfig(public_ip=public_ip)
        
        with safesocket.create_with_config("tcp-hello:python-matrix", addr, config, socket_type="client", auto_connect=True) as client:
            print("Python Client: Connected")
            
            # --- TEST 1: Basic Ping ---
            payload = "ping-python"
            print(f"Python Client: Sending {payload}")
            client.send(payload.encode())
            response = client.receive()
            if response.decode() != f"echo:{payload}":
                print(f"Python Client: FAILURE - Basic ping failed")
                return False

            # --- TEST 2: Metadata Verification ---
            print("Python Client: Verifying metadata...")
            client.send(b"meta_request")
            response = client.receive().decode()
            if not response.startswith("meta:"):
                print(f"Python Client: FAILURE - Metadata request failed")
                return False
            
            # format: meta:name,host,address
            parts = response[5:].split(",")
            if parts[0] != "python-matrix" or parts[2] != public_ip:
                print(f"Python Client: FAILURE - Metadata mismatch. Expected name=python-matrix, ip={public_ip}. Got: {response}")
                return False
            print(f"Python Client: Metadata verified (Host reported: {parts[1]})")

            # --- TEST 3: Large Payload (1MB) ---
            print("Python Client: Testing 1MB payload...")
            large_payload = b"A" * (1024 * 1024)
            start_time = time.time()
            client.send(large_payload)
            
            # The server prefixes with 'echo:'
            response = client.receive(len(large_payload) + 10)
            end_time = time.time()
            
            if len(response) != len(large_payload) + 5 or not response.startswith(b"echo:"):
                print(f"Python Client: FAILURE - Large payload corrupted. Length={len(response)}")
                return False
            
            print(f"Python Client: 1MB payload verified in {(end_time - start_time)*1000:.2f}ms")
            
            print("Python Client: ALL TESTS SUCCESS")
            return True
                
    except Exception as e:
        print(f"Python Client: ERROR - {e}")
        import traceback
        traceback.print_exc()
        return False

if __name__ == "__main__":
    address = "127.0.0.1:9999"
    if len(sys.argv) > 1:
        address = sys.argv[1]
    
    success = run_client(address)
    sys.exit(0 if success else 1)
