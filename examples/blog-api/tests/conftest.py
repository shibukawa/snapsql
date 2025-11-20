import os
import socket
import subprocess
import sys
import time
from pathlib import Path

import httpx
import pytest


APP_DIR = Path(__file__).resolve().parents[1]


def _reserve_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def _wait_until_server_ready(port: int, timeout: float = 15.0) -> None:
    deadline = time.time() + timeout
    url = f"http://127.0.0.1:{port}/health"
    while time.time() < deadline:
        try:
            response = httpx.get(url, timeout=1.0)
        except Exception:
            time.sleep(0.2)
            continue

        if response.status_code == 200:
            return

        time.sleep(0.2)

    raise RuntimeError("Blog API server failed to start within timeout")


@pytest.fixture(scope="session")
def live_server_url() -> str:
    """Run the FastAPI app via uvicorn and yield its base URL."""

    port = _reserve_port()
    env = os.environ.copy()
    python_path = env.get("PYTHONPATH")
    env["PYTHONPATH"] = (
        str(APP_DIR)
        if not python_path
        else os.pathsep.join([str(APP_DIR), python_path])
    )

    cmd = [
        sys.executable,
        "-m",
        "uvicorn",
        "app.main:app",
        "--host",
        "127.0.0.1",
        f"--port={port}",
    ]

    process = subprocess.Popen(
        cmd,
        cwd=APP_DIR,
        env=env,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )

    try:
        _wait_until_server_ready(port)
        yield f"http://127.0.0.1:{port}"
    finally:
        process.terminate()
        try:
            process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait()
