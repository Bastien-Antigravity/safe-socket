#!/usr/bin/env python
# coding:utf-8

"""
ESSENTIAL PROCESS:
Defines the SocketConfig data model for configuring safe-socket instances in Python,
mirroring the Go models.SocketConfig timeout and identity attributes.

DATA FLOW:
1. Input: Public IP, idle deadline (ms), heartbeat interval (ms), and handshake timeout (ms).
2. Logic: Stores validated configuration values passed to SafeSocket_CreateExtended.
3. Output: SocketConfig object consumed by SafeSocket and factory helpers.

KEY PARAMETERS:
- public_ip: Optional advertised node IP address.
- deadline_ms: Idle timeout in milliseconds (0 uses hybrid defaults; < 0 disables timeouts).
- heartbeat_interval_ms: Heartbeat cadence in milliseconds.
- handshake_timeout_ms: Connection/handshake timeout in milliseconds.
"""

# -----------------------------------------------------------------------------------------------


class SocketConfig:
    """
    SocketConfig mirrors the Go models.SocketConfig struct.
    It allows full customization of responsiveness and identity.
    """

    Name = "SocketConfig"

    def __init__(
        self,
        public_ip: str = "",
        deadline_ms: int = 0,
        heartbeat_interval_ms: int = 0,
        handshake_timeout_ms: int = 0,
    ) -> None:
        self.public_ip = public_ip
        self.deadline_ms = deadline_ms
        self.heartbeat_interval_ms = heartbeat_interval_ms
        self.handshake_timeout_ms = handshake_timeout_ms
