#!/usr/bin/env python3
import sys

# allow "from lib import MasterConnector"
sys.path.insert(0, "/opt/tests")
from lib import MasterConnector  # noqa: E402

# ---- arg parsing (transport REQUIRED) ----
if len(sys.argv) < 2 or not sys.argv[1].strip():
    print("usage: test-conn-mtls.py <transport>", file=sys.stderr)
    sys.exit(2)
transport = sys.argv[1].strip()

# ---- constants matching your compose setup ----
HOST_PLAIN = "slave"       # plain server container (listening without --ssl)
HOST_TLS   = "slave-tls"   # TLS server container     (listening with --ssl)
HOST_MTLS  = "slave-mtls"  # mTLS server container    (listening with --ssl --key testsecret)
PORT       = 8080

CLIENT_MTLS_GOOD = ["--ssl", "--key", "testsecret"]
CLIENT_MTLS_BAD  = ["--ssl", "--key", "wrongsecret"]

def main() -> int:
    r = MasterConnector(transport=transport)

    fails = 0
    # Test 1: mTLS client -> Plain server (should fail; expect an error)
    # If your environment sometimes yields EOF instead, flip require_error to False.
    fails += 0 if r.run_negative(host=HOST_PLAIN, port=PORT,
                                 name="Test 1: mTLS client -> Plain server",
                                 require_error=True,  # set to False if you observe clean EOFs only
                                 extra_args=CLIENT_MTLS_GOOD) else 1

    # Test 2: mTLS client -> TLS server (should fail; expect an error)
    fails += 0 if r.run_negative(host=HOST_TLS, port=PORT,
                                 name="Test 2: mTLS client -> TLS server",
                                 require_error=True,
                                 extra_args=CLIENT_MTLS_GOOD) else 1

    # Test 3: mTLS client -> mTLS server with matching key (should succeed)
    fails += 0 if r.run_positive(host=HOST_MTLS, port=PORT,
                                 name="Test 3: mTLS client -> mTLS server (matching key)",
                                 extra_args=CLIENT_MTLS_GOOD) else 1

    # Test 4: mTLS client -> mTLS server with wrong key (should fail; expect an error)
    fails += 0 if r.run_negative(host=HOST_MTLS, port=PORT,
                                 name="Test 4: mTLS client -> mTLS server (wrong key)",
                                 require_error=True,
                                 extra_args=CLIENT_MTLS_BAD) else 1

    print(f"\nFails: {fails}", flush=True)
    print("SUCCESS" if fails == 0 else "FAIL", flush=True)
    return 0 if fails == 0 else 1

if __name__ == "__main__":
    sys.exit(main())
