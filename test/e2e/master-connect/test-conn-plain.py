#!/usr/bin/env python3
import sys

# allow "from lib import MasterConnector"
sys.path.insert(0, "/opt/tests")
from lib import MasterConnector  # noqa: E402

# ---- arg parsing (transport REQUIRED) ----
if len(sys.argv) < 2 or not sys.argv[1].strip():
    print("usage: test-conn-plain.py <transport>", file=sys.stderr)
    sys.exit(2)
transport = sys.argv[1].strip()

# ---- constants matching your compose setup ----
HOST_PLAIN = "slave"       # plain server container
HOST_TLS   = "slave-tls"   # TLS server container (listening with --ssl)
HOST_MTLS  = "slave-mtls"  # mTLS server container (listening with --ssl --key)
PORT       = 8080          # all slaves listen on 8080

def main() -> int:
    r = MasterConnector(transport=transport)

    fails = 0
    # Test 1: Plain client -> Plain server (should succeed)
    fails += 0 if r.run_positive(host=HOST_PLAIN, port=PORT,
                                 name="Test 1: Plain client -> Plain server") else 1
    # Test 2: Plain client -> TLS server (should fail; no error but we should see EOF)
    fails += 0 if r.run_negative(host=HOST_TLS, port=PORT,
                                 name="Test 2: Plain client -> TLS server",
                                 require_error=False) else 1
    # Test 3: Plain client -> mTLS server (should fail; require error)
    fails += 0 if r.run_negative(host=HOST_MTLS, port=PORT,
                                 name="Test 3: Plain client -> mTLS server",
                                 require_error=True) else 1

    print(f"\nFails: {fails}", flush=True)
    print("SUCCESS" if fails == 0 else "FAIL", flush=True)
    return 0 if fails == 0 else 1

if __name__ == "__main__":
    sys.exit(main())
