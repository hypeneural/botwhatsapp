#!/usr/bin/env python3
"""Fixture-only regression for legacy 422 read-only audit; no CRM/WhatsApp."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("legacy422", HERE/"audit-ctwa-legacy-422-readonly.py")
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class Legacy422ForensicTest(unittest.TestCase):
    def test_phone_with_legacy_failure_and_new_receipt_is_not_replay_permission(self):
        with tempfile.TemporaryDirectory() as d:
            root=Path(d)
            tasks=root/"tasks"
            tasks.mkdir()
            rows=[
                {"step":"schedule_natal_crm","state":"suppressed","error":"crm: HTTP 422 natal_proposal_native_entry_required","phone_e164":"5548999990001"},
                {"step":"schedule_natal_crm","state":"confirmed","external_id":"test-only-dispatch","phone_e164":"5548999990001"},
                {"step":"schedule_natal_crm","state":"suppressed","error":"crm: HTTP 422 natal_proposal_native_entry_required","phone_e164":"5548999990002"}
            ]
            for n,row in enumerate(rows):
                (tasks/f"fixture-{n}.json").write_text(json.dumps(row),encoding="utf-8")
            r=module.audit(root)
            self.assertEqual(r["historical_422_candidate_tasks"],2)
            self.assertEqual(r["historical_422_unique_phones"],2)
            self.assertEqual(r["historical_422_with_some_confirmed_task_same_phone"],1)
            self.assertEqual(r["historical_422_without_confirmed_task_in_this_spool"],1)

    def test_missing_spool_fails_closed(self):
        with tempfile.TemporaryDirectory() as d:
            with self.assertRaises(ValueError):
                module.audit(Path(d))


if __name__ == "__main__":
    unittest.main()
