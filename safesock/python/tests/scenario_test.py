#!/usr/bin/env python
# coding:utf-8

"""
=============================================================================
ESSENTIAL PROCESS:
Consolidated unit and scenario test suite for safe-socket Python bindings,
validating echo messaging, custom handshake parameters, and deadline timeouts.

DATA FLOW:
1. Input: Ephemeral localhost endpoints and custom SocketConfig timeouts.
2. Logic: Spawns Python SafeSocket servers and clients via FFI, performs
   bidirectional echo exchanges, and verifies tight read timeout enforcement.
3. Output: Unit test pass/fail status via unittest runner.

KEY PARAMETERS:
- Name: Test suite identifier used in structured stdout logs.
=============================================================================
"""

from os.path import abspath as osPathAbspath, dirname as osPathDirname
from socket import AF_INET as socketAF_INET, SOCK_STREAM as socketSOCK_STREAM, socket as socketCreate
from sys import path as sysPath
from threading import Thread as threadCreate
from time import sleep as timeSleep, time as timeNow
from unittest import TestCase as unittestTestCase, main as unittestMain

sysPath.append(osPathDirname(osPathDirname(osPathAbspath(__file__))))

from safesocket import safesocket


# -----------------------------------------------------------------------------------------------


def _allocate_ephemeral_addr() -> str:
    """Allocate an available localhost TCP port for isolated test runs."""
    sock = socketCreate(socketAF_INET, socketSOCK_STREAM)
    try:
        sock.bind(("127.0.0.1", 0))
        host, port = sock.getsockname()
        return f"{host}:{port}"
    finally:
        sock.close()


# -----------------------------------------------------------------------------------------------


class TestScenarioCustomParameters(unittestTestCase):
    """
    Validation suite for Python SafeSocket FFI echo exchange and timeout scenarios.
    """

    Name = "TestScenarioCustomParameters"

    # -------------------------------------------------------------------------------------------

    def test_basic_echo_exchange(self) -> None:
        addr = _allocate_ephemeral_addr()
        server_errors = []

        with safesocket.create("tcp-hello:py-echo-srv", addr, socket_type="server", auto_connect=True) as server:
            def _serve() -> None:
                try:
                    with server.accept() as conn:
                        data = conn.receive()
                        if data != b"Hello from Python!":
                            server_errors.append(f"unexpected payload: {data!r}")
                            return
                        conn.send(b"Hello back from Server!")
                except Exception as exc:
                    server_errors.append(str(exc))

            srv_thread = threadCreate(target=_serve, daemon=True)
            srv_thread.start()
            timeSleep(0.05)

            with safesocket.create("tcp-hello:py-echo-cli", addr, socket_type="client", auto_connect=True) as client:
                client.send(b"Hello from Python!")
                reply = client.receive()
                self.assertEqual(reply, b"Hello back from Server!")

            srv_thread.join(timeout=2.0)
            self.assertEqual(server_errors, [])

    # -------------------------------------------------------------------------------------------

    def test_scenario_custom_parameters(self) -> None:
        addr = _allocate_ephemeral_addr()
        config = safesocket.SocketConfig(
            handshake_timeout_ms=150,
            deadline_ms=100,
            heartbeat_interval_ms=500,
        )

        print("{0} : starting scenario test with custom config: Handshake={1}ms, Deadline={2}ms".format(
            self.Name, config.handshake_timeout_ms, config.deadline_ms))

        with safesocket.create_with_config("tcp-hello:py-server", addr, config, socket_type="server", auto_connect=True) as server:
            client = safesocket.create_with_config("tcp-hello:py-client", addr, config, socket_type="client", auto_connect=False)
            start = timeNow()
            client.open()
            print("{0} : handshake completed in {1:.2f}ms".format(self.Name, (timeNow() - start) * 1000))

            try:
                with server.accept() as conn:
                    print("{0} : server accepted connection, waiting for timeout...".format(self.Name))
                    conn.receive(10)
                    self.fail("Expected timeout but receive succeeded")
            except safesocket.SafeSocketError as err:
                self.assertIn("timeout", str(err).lower())
                print("{0} : successfully triggered tight deadline error: {1}".format(self.Name, err))
            finally:
                client.close()


# -----------------------------------------------------------------------------------------------

if __name__ == "__main__":
    unittestMain()
