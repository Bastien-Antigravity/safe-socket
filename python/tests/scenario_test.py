import time
import unittest
import sys
import os

# Add the parent directory to sys.path to find the safesocket package
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from safesocket import safesocket

class TestScenarioCustomParameters(unittest.TestCase):
    def test_scenario_custom_parameters(self):
        addr = "127.0.0.1:9998"
        
        # --- CUSTOM PARAMETERS ---
        # Very tight timeouts for high responsiveness testing
        config = safesocket.SocketConfig(
            handshake_timeout_ms=150,
            deadline_ms=100,
            heartbeat_interval_ms=500
        )
        
        print(f"Starting Python scenario test with custom config: Handshake={config.handshake_timeout_ms}ms, Deadline={config.deadline_ms}ms")
        
        try:
            # 1. Create Server using new create_with_config entry point
            with safesocket.create_with_config("tcp-hello:py-server", addr, config, socket_type="server", auto_connect=True) as server:
                
                # 2. Create Client
                # We don't auto-connect to measure handshake time
                client = safesocket.create_with_config("tcp-hello:py-client", addr, config, socket_type="client", auto_connect=False)
                
                start = time.time()
                client.open()
                print(f"Handshake completed in {(time.time() - start) * 1000:.2f}ms")
                
                # 3. Verify Deadline enforcement
                # Client does nothing, server should timeout in ~100ms
                try:
                    conn = server.accept()
                    with conn:
                        print("Server accepted connection, waiting for timeout...")
                        # This should raise SafeSocketError with "i/o timeout"
                        conn.receive(10)
                        self.fail("Expected timeout but receive succeeded")
                except safesocket.SafeSocketError as e:
                    if "timeout" in str(e).lower():
                        print(f"Successfully triggered tight deadline error: {e}")
                    else:
                        self.fail(f"Expected timeout error, got: {e}")
                
                client.close()
                
        except Exception as e:
            self.fail(f"Scenario test failed with unexpected error: {e}")

if __name__ == '__main__':
    unittest.main()
