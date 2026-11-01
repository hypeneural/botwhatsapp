#!/usr/bin/env python3
"""Offline contract test of read-only runtime panel audit; no real credentials."""
import importlib.util
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import os
from pathlib import Path
import stat
import tempfile
from threading import Thread
import unittest

SCRIPT = Path(__file__).with_name("audit-ctwa-panel-readonly.py")
spec = importlib.util.spec_from_file_location("ctwa_panel_probe", SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

DEMO_TOKEN = "x" * 64

class FakePanel(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        pass

    def do_HEAD(self):
        self.respond()

    def do_GET(self):
        self.respond()

    def respond(self):
        if self.path in ("/healthz", "/readyz"):
            self.send_response(200)
            self.end_headers()
            if self.command == "GET":
                self.wfile.write(b"ok" if self.path == "/healthz" else b"ready")
            return
        if self.path in ("/dashboard", "/painel", "/api/dashboard/events?limit=10", "/api/dashboard/contacts"):
            import base64
            token = self.headers.get("Authorization", "")
            required = "Basic " + base64.b64encode(("gestor:" + DEMO_TOKEN).encode()).decode()
            if token != required:
                self.send_response(401)
                self.end_headers()
                return
            self.send_response(200)
            if self.path in ("/painel", "/dashboard"):
                self.send_header("Content-Type", "text/html")
                payload = b"<html><script>new URL(path, window.location.origin); credentials:'same-origin'</script></html>"
            else:
                self.send_header("Content-Type", "application/json")
                payload = (b'{"events":[],"stats":{"received":0}}' if "events" in self.path
                           else b'{"contacts":[],"stats":{}}')
            self.end_headers()
            if self.command == "GET":
                self.wfile.write(payload)
            return
        self.send_response(404)
        self.end_headers()


class PanelProbeOfflineTests(unittest.TestCase):
    def test_url_policy(self):
        self.assertEqual(module.normalized_base("http://127.0.0.1:8788"), "http://127.0.0.1:8788")
        self.assertEqual(module.normalized_base("https://private.example/"), "https://private.example")
        for bad in ("http://example.com", "https://user:secret@example.com",
                    "https://example.com/a", "https://example.com/?token=abc"):
            with self.assertRaises(ValueError):
                module.normalized_base(bad)

    def test_secret_must_not_be_world_readable(self):
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / ".env.production"
            path.write_text("BOT_DASHBOARD_TOKEN=" + DEMO_TOKEN + "\n", encoding="utf-8")
            os.chmod(path, 0o600)
            self.assertEqual(module.read_panel_secret(path), DEMO_TOKEN)
            os.chmod(path, 0o644)
            with self.assertRaises(ValueError):
                module.read_panel_secret(path)

    def test_smoke_with_synthetic_local_server(self):
        server = ThreadingHTTPServer(("127.0.0.1", 0), FakePanel)
        thread = Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            self.assertEqual(module.run(f"http://127.0.0.1:{server.server_port}", DEMO_TOKEN), 0)
            self.assertEqual(module.run(f"http://127.0.0.1:{server.server_port}", "not-correct"), 2)
        finally:
            server.shutdown()
            server.server_close()
            thread.join(timeout=2)


if __name__ == "__main__":
    unittest.main()
