import sys
import os
import time
import threading

# Add the current directory to sys.path so we can import safesocket
sys.path.append(os.path.join(os.getcwd(), 'python'))

from safesocket import safesocket

def run_server():
    try:
        print("[Server] Creating socket...")
        server = safesocket.create(profile="tcp", address="127.0.0.1:9999", socket_type="server")
        print("[Server] Listening...")
        server.listen()
        
        print("[Server] Waiting for connection...")
        conn = server.accept()
        print("[Server] Accepted connection!")
        
        data = conn.receive()
        print(f"[Server] Received: {data.decode('utf-8')}")
        
        response = f"Echo: {data.decode('utf-8')}"
        conn.send(response.encode('utf-8'))
        print("[Server] Response sent!")
        
        conn.close()
        server.close()
        print("[Server] Closed.")
    except Exception as e:
        print(f"[Server] Error: {e}")

def run_client():
    try:
        time.sleep(1) # Wait for server to start
        print("[Client] Creating socket...")
        client = safesocket.create(profile="tcp", address="127.0.0.1:9999", socket_type="client")
        print("[Client] Opening...")
        client.open()
        
        msg = "Hello from Python!"
        print(f"[Client] Sending: {msg}")
        client.send(msg.encode('utf-8'))
        
        data = client.receive()
        print(f"[Client] Received: {data.decode('utf-8')}")
        
        client.close()
        print("[Client] Closed.")
    except Exception as e:
        print(f"[Client] Error: {e}")

if __name__ == "__main__":
    server_thread = threading.Thread(target=run_server)
    client_thread = threading.Thread(target=run_client)
    
    server_thread.start()
    client_thread.start()
    
    server_thread.join()
    client_thread.join()
    
    print("Test finished.")
