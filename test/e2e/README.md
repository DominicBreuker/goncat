# End-to-end tests (test/e2e)

This document is for maintainers who will run, extend, or debug the end-to-end (E2E) tests under `test/e2e/`.
It describes the exact runtime topology, how test artifacts interact, and explicit guidelines to avoid common mistakes.

## Top-level summary

- E2E tests run inside small Alpine-based containers that mount the repository `test/e2e/` into `/opt/tests` and the built `dist/` binaries into `/opt/dist`.
- Two compose configurations exist for the two runtime topologies the project supports:
  - `docker-compose.slave-listen.yml` — the slave runs in "listen" mode and the master actively connects to it.
  - `docker-compose.slave-connect.yml` — the master runs in "listen" mode and the slave actively connects to it (reverse connection).
- Each compose file defines these services: `master`, `master-companion`, `slave`, and `slave-companion`. Roles are symmetric between the two compose files (one side listens, the other connects).
- Test logic is implemented as per-test Expect scripts under `master-listen/` and `master-connect/`. The `master` service runs the `test-runner.sh` script which executes the test scripts matching `test-*.sh` in the configured test set directory.

## Key files and their roles

- `docker-compose.slave-listen.yml` and `docker-compose.slave-connect.yml`
  - Define the runtime graph and healthchecks. Important details:
    - Containers are built from the `test/e2e/Dockerfile` image tag `local/tests`.
    - The repository `test/e2e/` is mounted read-only into `/opt/tests` so tests run the live scripts from the workspace.
    - The built program binaries are mounted from the repo `dist/` into `/opt/dist` (readonly). This means you must run `make build` (or copy a working binary into `dist/`) before running the E2E tests.
    - Services use three networks: `common` (bridge), and two internal-only private networks `master_private` and `slave_private`. The companion echo servers are placed on the private networks to make sure port-forwarding is actually required for reachability.
    - Healthchecks look for listening ports using `netstat` and are relied on by `depends_on` to order startup.

- `Dockerfile`
  - Uses `alpine:3` and installs `expect`, `socat`, `python3`, and `PySocks` (via pip) which are required by the tests.
  - Important: the compose files mount `test/e2e/` and `dist/` instead of copying files into the image. Do not copy or rely on baked-in test artifacts in the image; the mount behavior is required for rapid iteration.

- `test-runner.sh`
  - Simple harness that accepts the transport string as first parameter (e.g. `tcp://` or `ws://`) then a list of test scripts. It runs each script and aggregates failures. Exit code 0 = all pass.
  - The `master` container runs this script. It passes the chosen $TRANSPORT env var through to each Expect script.

- `lib.tcl`
  - Helper library loaded by all Expect test scripts. Provides small helpers:
    - Expect::client_connected / server_connected — wait for the role-specific connection log lines.
    - Expect::shell_access_works — confirm an interactive exec (`--exec sh`) provides a working shell by running `id` and `hostname` and checking output.
    - Utils::wait_for — reusable expect loop with retries and configurable substring vs exact matching. Defaults to 5s timeout and 2 retries. Tests rely on this retry behavior when waiting for ephemeral output.
  - When editing tests, prefer to add helpers here rather than duplicating logic in multiple scripts.

- `helpers/` (helper programs used in tests)
  - `master-companion-echo.sh` and `slave-companion-echo.sh` — each starts a TCP echo server on port 9000 and a UDP echo server on port 9001. The TCP echo uses `socat` with fork semantics; the UDP server uses `socat UDP4-RECVFROM` with fork.
  - `echo-handler.sh` — single-line handler that echoes the incoming line back prefixed with the container hostname. Tests assert this prefix to ensure the message went to the expected side (master-companion vs slave-companion).
  - `socks5-udp-test.py` — minimal Python client that exercises SOCKS5 `UDP ASSOCIATE` via PySocks. Used by SOCKS tests that verify UDP relaying.

## How tests are organized

- There are two matching directories with the same set of scenarios: `master-listen/` and `master-connect/`. Each directory contains the same test names and semantics but the call pattern differs (master listening vs master connecting).
- Test names and purpose (each file is an Expect script):
  - `test-plain.sh` – Basic connectivity: start the relevant role and assert the other side connects.
  - `test-exec.sh` – Master starts with `--exec sh`; tests that shell execution works and that it exits with EOF as appropriate.
  - `test-exec-pty.sh` – Like `test-exec.sh` but with `--pty`, verifying PTY allocation works.
  - `test-local-forward.sh` – Starts goncat with `-L 7000:slave-companion:9000` and then connects from the master to `localhost:7000` to verify the forwarded connection reaches `slave-companion:9000`.
  - `test-remote-forward.sh` – Starts goncat with `-R 7000:master-companion:9000` and then executes a command on the remote side to connect to `localhost:7000` there. Verify response originates from `master-companion`.
  - `test-socks-associate.sh` – Starts goncat with `-D 127.0.0.1:1080` enabling a local SOCKS5 proxy and then runs the Python UDP test client to verify `UDP ASSOCIATE` support.
  - `test-socks-connect.sh` – Tests SOCKS TCP CONNECT functionality (script exists but keep parity with other tests; check its contents when changing SOCKS behaviour).

## Concrete prerequisites and gotchas

- Build step (required): `dist/` must contain the compiled goncat binaries. In practice run `make build-linux` or `make build` from the repository root. Without this the containers will run a non-existent binary and tests will fail.
- Network namespaces and port visibility:
  - Companion echo servers listen on private networks to ensure port forwarding is actually used to reach them across the other side. Do not move echo servers onto the `common` network or bind them on 0.0.0.0 in the `master`/`slave` containers – that will mask port-forwarding bugs.
  - The tests rely on specific ports: goncat uses 8080 for the control connection, companions use 9000/TCP and 9001/UDP, and SOCKS uses 1080. Avoid changing those ports unless you update all healthchecks and tests.
- Healthchecks and startup ordering:
  - Compose `depends_on` uses the `service_healthy` condition. If you modify healthchecks, ensure the `netstat` command used there still finds a listener (some minimal base images may not include `netstat` — the `Dockerfile` relies on Alpine's packages to provide it via `net-tools` or busybox variants). If healthchecks are flaky, the dependent side may attempt to run tests before the server is ready.
  - Expect scripts also use `Utils::wait_for` which retries; this is complementary to the compose-level healthchecks.
- Expect and PTY behaviour:
  - Tests use `expect` to spawn goncat and then interact with the spawned process. When changing output lines (log messages) in the goncat binary, update `lib.tcl` matchers (for example: `Connecting to`, `New * connection from`) which the helpers rely on.
  - Timeout defaults in `lib.tcl` are short (5s) to keep tests fast. If adding tests that need more time (slow startup or heavy crypto operations), increase `timeout` locally in the script using `set timeout <seconds>` or adjust `Utils::set_timeout`.

## How to run tests locally (quick checks)

1. Build the binaries (from repo root):

```bash
make build-linux
```

2. From `test/e2e/`, start the compose configuration you need. Examples:

```bash
# master listens, slave connects
TRANSPORT='tcp' TEST_SET='master-listen' docker compose -f docker-compose.slave-connect.yml up --build --abort-on-container-exit --exit-code-from master

# slave listens, master connects
TRANSPORT='tcp' TEST_SET='master-connect' docker compose -f docker-compose.slave-listen.yml up --build --abort-on-container-exit --exit-code-from master
```

Notes:
- `--abort-on-container-exit --exit-code-from master` makes the `docker compose up` command return the exit code of the `master` service (where the test-runner runs) so CI can fail fast.
- Use `docker compose logs -f master` to follow the master output while debugging.

## Debugging failures — step-by-step checklist

1. Re-run `make build-linux` and ensure `dist/goncat.elf` exists. The containers mount that path read-only.
2. Recreate the test image and bring up the compose stack with `docker compose ... up` (see above) without `--abort-on-container-exit` to keep containers running for debugging.
3. Inspect logs for `master`, `slave`, `master-companion`, `slave-companion`:
   - Companion logs show echo server activity (`socat` logs) and are useful to verify where messages were received.
4. Enter the `master` or `slave` container to run Expect scripts manually or run individual scripts from `/opt/tests/master-listen` or `/opt/tests/master-connect`.
   - Example: `docker compose exec master /bin/sh` then run `/opt/tests/master-listen/test-plain.sh tcp://` to run a single script.
5. If Expect times out waiting for a specific hostname string, verify goncat's logging lines haven't changed. Update matching strings in `lib.tcl` if necessary.
6. For SOCKS/UDP issues, run the Python client manually inside the `master` container to see detailed exceptions (it prints the caught exception).

## Adding new tests — concrete guidelines

1. Follow the existing naming pattern: add a script named `test-<feature>.sh` in both `master-connect/` and `master-listen/` to keep parity. The `master` runner uses `$TEST_SET` to pick which directory to run.
2. Prefer re-using `Utils::wait_for` and the helpers in `lib.tcl`. If you need new helpers, add them to `lib.tcl` and keep them small and deterministic.
3. Avoid non-deterministic waits (sleep without checks). Use `wait_for` with an explicit timeout and add only small sleeps (1–3s) when waiting for port-forwarding to become available — but always check the forwarded behavior with an actual socket operation (socat/netcat) rather than assuming success.
4. If your test needs additional tools, add them to the `Dockerfile` and keep the image small — prefer Alpine packages. Remember that the image is only a small test harness; tests mount the repo at runtime.
5. When testing new network features, place test servers on the appropriate private network (master_private or slave_private) to ensure the forwarding/proxying is actually exercised.

## CI considerations

- The testing CI runs `make build-linux` before starting compose. The pipeline relies on the `master` service exit code as the test result. Keep `test-runner.sh` exit codes meaningful (0 success, non-zero fail) and avoid masking errors in Expect scripts.
- Tests are designed to be fast: most have short timeouts (5–15s). If CI runs on slower hosts or under high load, only increase timeouts where strictly necessary and keep the timeout changes local to the failing test script.

## Common mistakes and how to avoid them

- Forgetting to build binaries: the most common cause of failures. Always run `make build` or `make build-linux` first; the compose file mounts `../../dist` and will fail silently if the binary is missing.
- Changing log message text in `cmd` or `pkg` without updating `lib.tcl`: Expect scripts depend on textual markers. When refactoring logs, update the TCL helpers.
- Moving companion servers to the `common` network: that invalidates port-forwarding tests because services would be reachable directly.
- Adding cryptography or long-running setup without increasing waits: keep any heavy setup outside the tight Expect timeouts or adjust timeout using `set timeout` in the script.

## Quick reference: ports used

- goncat control: 8080
- companion TCP echo: 9000
- companion UDP echo: 9001
- SOCKS proxy (tests): 1080

## Final notes

This E2E suite is intended to be small, fast, and deterministic. It emphasizes exercising network features (local/remote port forwarding and SOCKS TCP/UDP) by placing test servers on private networks so the code paths are enforced. When adding or changing tests, update `lib.tcl` and the compose healthchecks if you alter output, ports, or behavior.