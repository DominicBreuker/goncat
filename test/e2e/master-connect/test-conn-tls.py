#!/usr/bin/env python3
import sys

# allow "from lib import MasterConnector"
sys.path.insert(0, "/opt/tests")
from lib import MasterConnector  # noqa: E402

# ---- arg parsing (transport REQUIRED) ----
if len(sys.argv) < 2 or not sys.argv[1].strip():
    print("usage: test-conn-tls.py <transport>", file=sys.stderr)
    sys.exit(2)
transport = sys.argv[1].strip()

# ---- constants matching your compose setup ----
HOST_PLAIN = "slave"       # plain server container (no --ssl)
HOST_TLS   = "slave-tls"   # TLS server container     (--ssl)
HOST_MTLS  = "slave-mtls"  # mTLS server container    (--ssl --key testsecret)
PORT       = 8080

CLIENT_TLS = ["--ssl"]

def main() -> int:
    r = MasterConnector(transport=transport)

    fails = 0
    # Test 1: TLS client -> Plain server (should fail; expect error)
    # If you observe clean EOFs here in your environment, switch require_error=False.
    fails += 0 if r.run_negative(host=HOST_PLAIN, port=PORT,
                                 name="Test 1: TLS client -> Plain server",
                                 require_error=True,
                                 extra_args=CLIENT_TLS) else 1

    # Test 2: TLS client -> TLS server (should succeed)
    fails += 0 if r.run_positive(host=HOST_TLS, port=PORT,
                                 name="Test 2: TLS client -> TLS server",
                                 extra_args=CLIENT_TLS) else 1

    # Test 3: TLS client -> mTLS server (should fail; expect error)
    fails += 0 if r.run_negative(host=HOST_MTLS, port=PORT,
                                 name="Test 3: TLS client -> mTLS server",
                                 require_error=True,
                                 extra_args=CLIENT_TLS) else 1

    print(f"\nFails: {fails}", flush=True)
    print("SUCCESS" if fails == 0 else "FAIL", flush=True)
    return 0 if fails == 0 else 1

if __name__ == "__main__":
    sys.exit(main())
