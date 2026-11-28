#!/usr/bin/env python
# coding:utf-8

"""
ESSENTIAL PROCESS:
Defines the SafeSocketError exception class for the safe-socket Python SDK.

DATA FLOW:
1. Input: Error message string retrieved from the libsafesocket C ABI.
2. Logic: Encapsulates transport, handshake, and lifecycle errors as a Python Exception.
3. Output: Raised SafeSocketError instances handled by Python SDK consumers.

KEY PARAMETERS:
- SafeSocketError: Base exception class for all safe-socket Python failures.
"""

# -----------------------------------------------------------------------------------------------


class SafeSocketError(Exception):
    """Base exception for SafeSocket operations."""

    Name = "SafeSocketError"
