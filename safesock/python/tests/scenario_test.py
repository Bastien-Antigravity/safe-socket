#!/usr/bin/env python
# coding:utf-8

"""
ESSENTIAL PROCESS:
Scenario testing for safe-socket Python bindings. Verifies custom configuration, 
handshake performance, and deadline enforcement.

DATA FLOW:
Initializes Server -> Initializes Client -> Performs Handshake -> Measures Latency -> Validates Timeouts.

KEY PARAMETERS:
- addr: Localhost testing address.
- config: High-responsiveness SocketConfig for stress testing.
"""

import time as timeTime
import unittest
import sys as sysSys
import os as osOs

# Add the parent directory to sys.path to find the safesocket package
sysSys.path.append(osOs.path.dirname(osOs.path.dirname(osOs.path.abspath(__file__))))

from safesocket import safesocket

class TestScenarioCustomParameters(unittest.TestCase):
    """
    Validation suite for high-responsiveness scenarios.
    """

    Name = "TestScenarioCustomParameters"

    # -----------------------------------------------------------------------------------------------

    def test_scenario_custom_parameters(self) -> None:
        addr = "127.0.0.1:9998"
        
        # --- CUSTOM PARAMETERS ---
        # Very tight timeouts for high responsiveness testing
        config = safesocket.SocketConfig(
            handshake_timeout_ms=150,
            deadline_ms=100,
            heartbeat_interval_ms=500
        )
        
        print("{0} : starting scenario test with custom config: Handshake={1}ms, Deadline={2}ms".format(
            self.Name, config.handshake_timeout_ms, config.deadline_ms))
        
        try:
            # 1. Create Server using new create_with_config entry point
            with safesocket.create_with_config("tcp-hello:py-server", addr, config, socket_type="server", auto_connect=True) as server:
                
                # 2. Create Client
                # We don't auto-connect to measure handshake time
                client = safesocket.create_with_config("tcp-hello:py-client", addr, config, socket_type="client", auto_connect=False)
                
                start = timeTime.time()
                client.open()
                print("{0} : handshake completed in {1:.2f}ms".format(self.Name, (timeTime.time() - start) * 1000))
                
                # 3. Verify Deadline enforcement
                # Client does nothing, server should timeout in ~100ms
                try:
                    conn = server.accept()
                    with conn:
                        print("{0} : server accepted connection, waiting for timeout...".format(self.Name))
                        # This should raise SafeSocketError with "i/o timeout"
                        conn.receive(10)
                        self.fail("Expected timeout but receive succeeded")
                except safesocket.SafeSocketError as e:
                    if "timeout" in str(e).lower():
                        print("{0} : successfully triggered tight deadline error: {1}".format(self.Name, e))
                    else:
                        self.fail("Expected timeout error, got: {0}".format(e))
                
                client.close()
                
        except Exception as e:
            self.fail("Scenario test failed with unexpected error: {0}".format(e))

# -----------------------------------------------------------------------------------------------

if __name__ == '__main__':
    unittest.main()
