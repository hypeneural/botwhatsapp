#!/usr/bin/env python3
"""Synthetic GET CRM customer context tests. No real CRM access or client PII."""
import importlib.util
import io
import json
import os
from pathlib import Path
import tempfile
import unittest

SCRIPT = Path(__file__).with_name("sync-ctwa-crm-names-readonly.py")
spec = importlib.util.spec_from_file_location("ctwa_name_sync", SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

class FakeHTTP:
    def __init__(self, response):
        self.response = response
        self.status = 200
    def read(self, _limit):
        return self.response
    def __enter__(self):
        return self
    def __exit__(self, *unused):
        return False

class FakeOpener:
    def __init__(self, obj):
        self.obj = obj
        self.paths = []
    def open(self, request, timeout=0):
        self.paths.append(request.full_url)
        if request.get_method() != "GET":
            raise AssertionError("only GET allowed")
        return FakeHTTP(json.dumps({"data":self.obj}).encode())

class NamesReadOnlyTest(unittest.TestCase):
    def test_unique_customer_name_with_scope_contract(self):
        opener = FakeOpener({"lookup":{"status":"matched"}, "customer":{"display_name":"Nome CRM Teste"}})
        result = module.lookup(opener, "5548999990001", "fixture-token")
        self.assertEqual(result,"Nome CRM Teste")
        self.assertEqual(opener.paths, ["https://evydencia.com/api/v1/extension/customer-context?phone=48999990001"])

    def test_ambiguous_is_not_guessed(self):
        for state in ("ambiguous","not_found"):
            opener = FakeOpener({"lookup":{"status":state},"customer":{"display_name":"Nao usar"}})
            self.assertIsNone(module.lookup(opener,"5548999990001","fixture-token"))

    def test_phone_must_come_from_e164_contact_state(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            (root/"contacts").mkdir()
            for idx, phone in enumerate(("5548999990001","123456@lid","5548999990002")):
                (root/"contacts"/f"{idx}.json").write_text(json.dumps({"phone_e164":phone}),encoding="utf-8")
            self.assertEqual(module.contact_numbers(root), ["5548999990001","5548999990002"])

    def test_cache_written_privately(self):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp)/"crm-names-cache.json"
            module.save_cache(path,{"5548999990001":{"name":"Nome CRM Teste","at":module.time.time()}})
            data=json.loads(path.read_text(encoding="utf-8"))
            self.assertEqual(data["names"]["5548999990001"],"Nome CRM Teste")
            self.assertEqual(len(module.load_cache(path)),1)
            self.assertEqual(os.stat(path).st_mode & 0o777,0o600)

if __name__ == "__main__":
    unittest.main()
