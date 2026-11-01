#!/usr/bin/env python3
"""Synthetic-only security regression tests; never access real env/CRM."""
import importlib.util
import io
import os
from pathlib import Path
import stat
import tempfile
import unittest
from contextlib import redirect_stdout
from unittest.mock import patch

SCRIPT = Path(__file__).resolve().parent / "configure-ctwa-secrets.py"
spec = importlib.util.spec_from_file_location("ctwa_secrets", SCRIPT)
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)
FAKE = "17|" + ("a" * 40)


class ConfigSecretsTest(unittest.TestCase):
    def test_generates_dashboard_not_fake_crm(self):
        data, result = m.prepare("BOT_ENABLED=true\nBOT_DASHBOARD_TOKEN=\n", None, False, False)
        self.assertIn("BOT_DASHBOARD_TOKEN=", data)
        self.assertEqual(result["dashboard"], "GENERATE_NEW")
        self.assertEqual(result["sanctum"], "MISSING_EXTERNAL_CRM_TOKEN")
        self.assertNotIn(m.KEY_CRM + "=", data)
        self.assertEqual(len(m.parse_env(data)[m.KEY_DASH]), 64)

    def test_keeps_existing_secrets_idempotently(self):
        src = f"BOT_DASHBOARD_TOKEN={'c' * 64}\n{m.KEY_CRM}={FAKE}\n"
        result, meta = m.prepare(src, None, False, False)
        self.assertEqual(result, src)
        self.assertFalse(meta["changed"])

    def test_rejects_duplicate_keys_and_invalid_credential(self):
        with self.assertRaises(ValueError):
            m.prepare("BOT_DASHBOARD_TOKEN=a\nBOT_DASHBOARD_TOKEN=b\n", None, False, False)
        with self.assertRaises(ValueError):
            m.prepare(m.KEY_CRM + "=not-a-real-sanctum\n", None, False, False)

    def test_read_token_file_must_be_private_and_correct_format(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "token"
            path.write_text(FAKE + "\n", encoding="utf-8")
            os.chmod(path, 0o600)
            self.assertEqual(m.read_sanctum_file(path), FAKE)
            os.chmod(path, 0o644)
            with self.assertRaises(ValueError):
                m.read_sanctum_file(path)
            os.chmod(path, 0o600)
            path.write_text("i-am-random-not-crm-token", encoding="utf-8")
            with self.assertRaises(ValueError):
                m.read_sanctum_file(path)

    def test_atomic_backup_permissions_and_no_output(self):
        with tempfile.TemporaryDirectory() as tmp:
            target = Path(tmp) / ".env.production"
            original = "BOT_ENABLED=true\nBOT_DASHBOARD_TOKEN=\n"
            target.write_text(original, encoding="utf-8")
            os.chmod(target, 0o600)
            changed, _ = m.prepare(original, FAKE, False, False)
            buf = io.StringIO()
            with redirect_stdout(buf):
                backup = m.apply_atomic(target, changed, original.encode("utf-8"))
            self.assertEqual(buf.getvalue(), "")
            self.assertEqual(backup.read_text(encoding="utf-8"), original)
            self.assertEqual(stat.S_IMODE(target.stat().st_mode), 0o600)
            self.assertEqual(stat.S_IMODE(backup.stat().st_mode), 0o600)
            self.assertIn(FAKE, target.read_text(encoding="utf-8"))
            self.assertNotIn(FAKE, backup.read_text(encoding="utf-8"))


if __name__ == "__main__":
    unittest.main()
