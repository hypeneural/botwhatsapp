"""Run with python3 -m unittest scripts/test_relay_forensic_readonly.py -v."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent


def load_script(file, module_name):
    spec = importlib.util.spec_from_file_location(module_name, ROOT / file)
    m = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(m)
    return m


spool = load_script("audit_relay_spool_readonly.py", "relay_spool_audit")
contract = load_script("preflight_evolution_relay_contract_readonly.py", "relay_contract_audit")


class RelayForensicTests(unittest.TestCase):
    def test_spool_pending_and_completed(self):
        with tempfile.TemporaryDirectory() as directory:
            d = Path(directory)
            pending = "a" * 64
            complete = "b" * 64
            (d / (pending + ".json")).write_text(json.dumps({
                "key": pending, "body": {"event": "Message"},
                "created_at": "2026-10-31T10:00:00Z",
                "status_done": True, "bot_done": False}), encoding="utf-8")
            (d / (complete + ".json")).write_text(json.dumps({
                "key": complete, "status_done": True, "bot_done": True,
                "completed_at": "2026-10-31T11:00:00Z"}), encoding="utf-8")
            for f in d.iterdir():
                f.chmod(0o600)
            result = spool.inspect(d)
            self.assertEqual(result["status"], "INVENTORY_ONLY")
            self.assertEqual(result["counts"]["pending_bot"], 1)
            self.assertEqual(result["counts"]["completed"], 1)
            self.assertFalse(result["delivery_proven"])
            self.assertNotIn("Message", str(result))

    def test_spool_rejects_insecure_or_corrupt_files(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / ("a" * 64 + ".json")
            path.write_text("{}", encoding="utf-8")
            path.chmod(0o600)
            self.assertEqual(spool.inspect(Path(directory))["status"], "BLOCKED")
            path.chmod(0o644)
            self.assertEqual(spool.inspect(Path(directory))["counts"]["unsafe_permissions"], 1)

    def test_evolution_072_shape_and_documented_main_shape(self):
        uuid = "uuid-test"
        good = {"event": "Message", "instanceId": uuid,
                "data": {"Info": {
                    "ID": "id1", "Chat": "5511999999999@s.whatsapp.net",
                    "IsFromMe": False},
                    "Message": {"conversation": "Ola"}}}
        self.assertEqual(contract.check(good, uuid)["status"], "MATCH")
        main_docs = {"event": "MESSAGE", "instance": "evydencia",
                     "data": {"key": {"id": "id1",
                                      "remoteJid": "5511999999999@s.whatsapp.net",
                                      "fromMe": False},
                              "message": {"conversation": "Ola"}}}
        result = contract.check(main_docs, uuid)
        self.assertEqual(result["status"], "BLOCKED_CONTRACT")
        self.assertTrue(result["has_documented_go_key_shape"])
        self.assertFalse(result["relay_eventkey_accepted"])

    def test_private_sample_permissions(self):
        with tempfile.TemporaryDirectory() as directory:
            file = Path(directory) / "event.json"
            file.write_text("{}", encoding="utf-8")
            file.chmod(0o644)
            with self.assertRaises(ValueError):
                contract.read_private(str(file), json_content=True)
            file.chmod(0o600)
            self.assertEqual(contract.read_private(str(file), json_content=True), {})


if __name__ == "__main__":
    unittest.main()
