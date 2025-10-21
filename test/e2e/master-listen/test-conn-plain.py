#!/usr/bin/env python3
import sys

# allow "from lib import MasterRunner"
sys.path.insert(0, "/opt/tests")
from lib import MasterRunner  # noqa: E402

# ---- arg parsing (transport REQUIRED) ----
if len(sys.argv) < 2 or not sys.argv[1].strip():
    print("usage: test-conn-plain.py <transport>", file=sys.stderr)
    sys.exit(2)
transport = sys.argv[1].strip()

# ---- constants matching your compose setup ----
PORT_PLAIN = 8080   # plain client container connects here
PORT_TLS   = 8081   # TLS-only client container connects here
PORT_MTLS  = 8082   # mTLS client container connects here

def main() -> int:
    r = MasterRunner(transport=transport)

    fails = 0
    # Test 1: Plain server -> Plain client (should succeed)
    fails += 0 if r.run_positive(port=PORT_PLAIN,
                                 name="Test 1: Plain server -> Plain client",
                                 use_ssl=False, key="") else 1
    # Test 2: Plain server -> TLS client (should fail; expect error)
    fails += 0 if r.run_negative(port=PORT_TLS,
                                 name="Test 2: Plain server -> TLS client",
                                 use_ssl=False, key="", require_error=True) else 1
    # Test 3: Plain server -> mTLS client (should fail; expect error)
    fails += 0 if r.run_negative(port=PORT_MTLS,
                                 name="Test 3: Plain server -> mTLS client",
                                 use_ssl=False, key="", require_error=False) else 1

    print(f"\nFails: {fails}", flush=True)
    print("SUCCESS" if fails == 0 else "FAIL", flush=True)
    return 0 if fails == 0 else 1

if __name__ == "__main__":
    sys.exit(main())
