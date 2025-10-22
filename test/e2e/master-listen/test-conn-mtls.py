#!/usr/bin/env python3
import sys

# allow "from lib import MasterRunner"
sys.path.insert(0, "/opt/tests")
from lib import MasterRunner  # noqa: E402

# ---- arg parsing (transport REQUIRED) ----
if len(sys.argv) < 2 or not sys.argv[1].strip():
    print("usage: test-conn-mtls.py <transport>", file=sys.stderr)
    sys.exit(2)
transport = sys.argv[1].strip()

# ---- constants matching your compose setup ----
PORT_PLAIN = 8080
PORT_TLS   = 8081
PORT_MTLS  = 8082

KEY_GOOD = "testsecret"
KEY_BAD  = "wrongsecret"

def main() -> int:
    r = MasterRunner(transport=transport)

    fails = 0
    # 1) mTLS master vs plain slave -> reject
    fails += 0 if r.run_negative(port=PORT_PLAIN,
                                 name="Test 1: mTLS master -> plain slave",
                                 use_ssl=True, key=KEY_GOOD, require_error=False) else 1
    # 2) mTLS master vs TLS-only slave -> reject
    fails += 0 if r.run_negative(port=PORT_TLS,
                                 name="Test 2: mTLS master -> TLS-only slave",
                                 use_ssl=True, key=KEY_GOOD, require_error=False) else 1
    # 3) mTLS master vs mTLS slave (matching key) -> accept
    fails += 0 if r.run_positive(port=PORT_MTLS,
                                 name="Test 3: mTLS master -> mTLS slave (matching key)",
                                 use_ssl=True, key=KEY_GOOD) else 1
    # 4) mTLS master vs mTLS slave (wrong key) -> reject
    fails += 0 if r.run_negative(port=PORT_MTLS,
                                 name="Test 4: mTLS master -> mTLS slave (WRONG key)",
                                 use_ssl=True, key=KEY_BAD, require_error=False) else 1

    print(f"\nFails: {fails}", flush=True)
    print("SUCCESS" if fails == 0 else "FAIL", flush=True)
    return 0 if fails == 0 else 1

if __name__ == "__main__":
    sys.exit(main())
