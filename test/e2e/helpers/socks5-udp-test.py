#!/usr/bin/env python3
"""
Minimal SOCKS5 UDP ASSOCIATE test client.
Requires: pip install PySocks
"""

import socket
import socks
import sys

def main():
    if len(sys.argv) < 5:
        print("Usage: python3 socks5-udp-test.py <socks_host> <socks_port> <udp_target_host> <udp_target_port>")
        sys.exit(1)

    socks_host = sys.argv[1]
    socks_port = int(sys.argv[2])
    target_host = sys.argv[3]
    target_port = int(sys.argv[4])

    # --- create UDP socket that talks via SOCKS5 ---
    s = socks.socksocket(socket.AF_INET, socket.SOCK_DGRAM)
    s.set_proxy(socks.SOCKS5, socks_host, socks_port)

    try:
        s.settimeout(5.0)
        message = b"test message\n"
        print(f"Sending {message!r} to {target_host}:{target_port} via SOCKS5 {socks_host}:{socks_port}")

        s.sendto(message, (target_host, target_port))

        data, addr = s.recvfrom(4096)
        print(f"Received: {data.decode('utf-8', errors='ignore')}")
        print("✓ UDP ASSOCIATE test successful!")
        return 0
    except Exception as e:
        print(f"✗ Error: {e}")
        print("SOCKS proxy may not support UDP ASSOCIATE or UDP server may be unreachable.")
        return 1
    finally:
        s.close()

if __name__ == "__main__":
    sys.exit(main())
