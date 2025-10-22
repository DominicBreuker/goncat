#!/usr/bin/env python3
"""
Minimal test helpers for goncat e2e tests.

Provides:
  - MasterRunner   : start 'master listen ...' and assert outcomes
  - MasterConnector: start 'master connect ...' and assert outcomes
"""

import re
import time
import subprocess
import threading
import queue
from typing import Optional

# Log patterns (tolerant to prefixes/ANSI)
SESSION_RE = re.compile(r"Session with .* established")
ERROR_RE   = re.compile(r"Error:")

def _ts() -> str:
    return time.strftime("%H:%M:%S")

def _kill_best_effort(p: subprocess.Popen) -> None:
    if p.poll() is None:
        try: p.terminate()
        except Exception: pass
        time.sleep(0.6)
    if p.poll() is None:
        try: p.kill()
        except Exception: pass

def _tee(proc: subprocess.Popen, out_q: "queue.Queue[str]") -> None:
    try:
        for line in iter(proc.stdout.readline, ""):
            print(line.rstrip("\n"), flush=True)  # mirror to compose logs
            out_q.put(line)
    finally:
        try: proc.stdout.close()
        except Exception: pass

def _wait_for_outcome(proc: subprocess.Popen, wait_secs: int) -> str:
    """
    Stream output up to wait_secs.
    Returns: 'session' | 'error' | 'timeout' | 'eof'
    """
    ql: "queue.Queue[str]" = queue.Queue()
    threading.Thread(target=_tee, args=(proc, ql), daemon=True).start()
    saw_error = False
    deadline = time.time() + wait_secs

    while time.time() < deadline:
        try:
            line = ql.get(timeout=0.2)
        except queue.Empty:
            line = None

        if line is not None:
            if SESSION_RE.search(line):
                return "session"
            if ERROR_RE.search(line):
                saw_error = True

        if proc.poll() is not None:
            return "error" if saw_error else "eof"

    return "error" if saw_error else "timeout"

# ---------------------------------------------------------------------------

class MasterRunner:
    """
    Encapsulates starting/stopping a master listener and asserting outcomes.
    """

    def __init__(
        self,
        transport: str,
        bin_path: str = "/opt/dist/goncat.elf",
        wait_pos: int = 30,
        wait_neg: int = 12,
        deadline_pos: int = 35,
        deadline_neg: int = 14,
    ) -> None:
        if not transport:
            raise ValueError("transport is required")
        self.transport = transport
        self.bin = bin_path
        self.wait_pos = wait_pos
        self.wait_neg = wait_neg
        self.deadline_pos = deadline_pos
        self.deadline_neg = deadline_neg

    def _start_master(self, *, port: int, use_ssl: bool = False, key: str = "") -> subprocess.Popen:
        args = [self.bin, "master", "listen", f"{self.transport}://:{port}", "--timeout", "2000"]
        if use_ssl:
            args += ["--ssl"]
        if key:
            args += ["--key", key]
        print(f"{_ts()} -> spawn {' '.join(args)}", flush=True)
        return subprocess.Popen(
            args,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            bufsize=1,
            universal_newlines=True,
        )

    def _watchdog(self, proc: subprocess.Popen, seconds: int) -> None:
        time.sleep(seconds)
        _kill_best_effort(proc)

    def run_positive(self, *, port: int, name: str, use_ssl: bool = False, key: str = "") -> bool:
        print(f"\n=== {name} (port {port}) ===", flush=True)
        p = self._start_master(port=port, use_ssl=use_ssl, key=key)
        threading.Thread(target=self._watchdog, args=(p, self.deadline_pos), daemon=True).start()
        outcome = _wait_for_outcome(p, self.wait_pos)
        ok = (outcome == "session")
        print(("✓ " if ok else "✗ ") + f"{name}: "
              + ("session established" if ok else f"expected session, got {outcome}"), flush=True)
        _kill_best_effort(p)
        try: p.wait(timeout=1.0)
        except Exception: pass
        return ok

    def run_negative(
        self,
        *,
        port: int,
        name: str,
        use_ssl: bool = False,
        key: str = "",
        require_error: bool = False,
    ) -> bool:
        print(f"\n=== {name} (port {port}) ===", flush=True)
        p = self._start_master(port=port, use_ssl=use_ssl, key=key)
        threading.Thread(target=self._watchdog, args=(p, self.deadline_neg), daemon=True).start()
        outcome = _wait_for_outcome(p, self.wait_neg)

        if outcome == "session":
            print(f"✗ {name}: session should NOT be established", flush=True)
            ok = False
        elif require_error and outcome != "error":
            print(f"✗ {name}: expected an error message, got {outcome}", flush=True)
            ok = False
        else:
            note = "error seen" if outcome == "error" else "timeout"
            print(f"✓ {name}: no session ({note})", flush=True)
            ok = True

        _kill_best_effort(p)
        try: p.wait(timeout=1.0)
        except Exception: pass
        return ok

# ---------------------------------------------------------------------------

class MasterConnector:
    """
    Encapsulates starting/stopping a connecting master and asserting outcomes:
        /opt/dist/goncat.elf master connect <transport>://<host>:<port> --timeout 2000
    """

    def __init__(
        self,
        transport: str,
        bin_path: str = "/opt/dist/goncat.elf",
        wait_pos: int = 30,
        wait_neg: int = 12,
        deadline_pos: int = 35,
        deadline_neg: int = 14,
    ) -> None:
        if not transport:
            raise ValueError("transport is required")
        self.transport = transport
        self.bin = bin_path
        self.wait_pos = wait_pos
        self.wait_neg = wait_neg
        self.deadline_pos = deadline_pos
        self.deadline_neg = deadline_neg

    def _start_connect(self, *, host: str, port: int, extra_args: Optional[list] = None) -> subprocess.Popen:
        url = f"{self.transport}://{host}:{port}"
        args = [self.bin, "master", "connect", url, "--timeout", "2000"]
        if extra_args:
            args += extra_args
        print(f"{_ts()} -> spawn {' '.join(args)}", flush=True)
        return subprocess.Popen(
            args,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            bufsize=1,
            universal_newlines=True,
        )

    def _watchdog(self, proc: subprocess.Popen, seconds: int) -> None:
        time.sleep(seconds)
        _kill_best_effort(proc)

    def run_positive(self, *, host: str, port: int, name: str, extra_args: Optional[list] = None) -> bool:
        """Expect a session to be established."""
        print(f"\n=== {name} ({host}:{port}) ===", flush=True)
        p = self._start_connect(host=host, port=port, extra_args=extra_args)
        threading.Thread(target=self._watchdog, args=(p, self.deadline_pos), daemon=True).start()
        outcome = _wait_for_outcome(p, self.wait_pos)
        ok = (outcome == "session")
        print(("✓ " if ok else "✗ ") + f"{name}: "
              + ("session established" if ok else f"expected session, got {outcome}"), flush=True)
        _kill_best_effort(p)
        try: p.wait(timeout=1.0)
        except Exception: pass
        return ok

    def run_negative(
        self,
        *,
        host: str,
        port: int,
        name: str,
        require_error: bool = True,
        extra_args: Optional[list] = None,
    ) -> bool:
        """Expect NO session. If require_error=True, must see an error line (not just timeout)."""
        print(f"\n=== {name} ({host}:{port}) ===", flush=True)
        p = self._start_connect(host=host, port=port, extra_args=extra_args)
        threading.Thread(target=self._watchdog, args=(p, self.deadline_neg), daemon=True).start()
        outcome = _wait_for_outcome(p, self.wait_neg)

        if outcome == "session":
            print(f"✗ {name}: session should NOT be established", flush=True)
            ok = False
        elif require_error and outcome != "error":
            print(f"✗ {name}: expected an error message, got {outcome}", flush=True)
            ok = False
        else:
            note = "error seen" if outcome == "error" else "timeout"
            print(f"✓ {name}: no session ({note})", flush=True)
            ok = True

        _kill_best_effort(p)
        try: p.wait(timeout=1.0)
        except Exception: pass
        return ok
