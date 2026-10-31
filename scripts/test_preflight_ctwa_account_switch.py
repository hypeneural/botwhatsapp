"""Run with python3 -m unittest scripts/test_preflight_ctwa_account_switch.py -v."""
import importlib.util
import contextlib
import io
import json
import os
from pathlib import Path
import tempfile
import unittest

source = Path(__file__).with_name("preflight_ctwa_account_switch.py")
spec = importlib.util.spec_from_file_location("ctwa_switch", source)
switch = importlib.util.module_from_spec(spec)
spec.loader.exec_module(switch)


def fixture():
    uuid = "00000000-0000-4000-8000-000000000001"
    phone = "5548999991234"
    key = "A" * 35
    secret = "S" * 40
    bot = {
        "BOT_ADS_PHONE": phone, "BOT_ADS_INSTANCE_UUID": uuid,
        "EVOLUTION_ADS_APIKEY": key, "BOT_ADS_WEBHOOK_SECRET": secret,
        "BOT_GREETING_PHONE": "", "BOT_GREETING_INSTANCE_UUID": "",
    }
    relay = {
        "RELAY_ENABLED":"true", "RELAY_EXPECTED_INSTANCE_UUID":uuid,
        "RELAY_BOT_URL":"http://bot:8788/webhook/ads/"+secret,
        "RELAY_STATUS_URL":"http://status:8050/webhook/evolution/example",
        "RELAY_INGRESS_SECRET":"R"*40,
    }
    inv = {"data":[{"name":"ads-example","id":uuid,"jid":phone+"@s.whatsapp.net",
                   "connected":True,"token":key,"events":"MESSAGE,READ_RECEIPT",
                   "webhook":"http://relay:8790/webhook/evolution/private"}]}
    return bot,relay,inv


class AccountSwitchTests(unittest.TestCase):
    def test_complete_binding_is_ready_without_disclosing_secrets(self):
        bot,relay,inv=fixture()
        out=switch.verify_inventory(inv,bot,relay)
        self.assertEqual(out["status"],"PASS_BINDINGS_ONLY")
        self.assertFalse(out["changes_performed"])
        self.assertNotIn(bot["BOT_ADS_PHONE"],str(out))
        self.assertNotIn(bot["EVOLUTION_ADS_APIKEY"],str(out))
        self.assertFalse(out["provider_delivery_proven"])

    def test_registered_jid_device_alias_requires_explicit_evidence(self):
        bot, relay, inv = fixture()
        bot["BOT_ADS_PHONE"] = "5548996425287"
        inv["data"][0]["jid"] = "554896425287:15@s.whatsapp.net"
        out=switch.verify_inventory(inv,bot,relay)
        self.assertEqual(out["status"],"BLOCKED_ACCOUNT_BINDING")
        bot["BOT_ADS_PAIRED_JID"] = "554896425287:15@s.whatsapp.net"
        out=switch.verify_inventory(inv,bot,relay)
        self.assertEqual(out["status"],"PASS_BINDINGS_ONLY")
        self.assertTrue(out["checks"]["paired_jid_valid"])
        self.assertFalse(out["changes_performed"])
        bot["BOT_ADS_PAIRED_JID"] = "554896425287:15:extra@s.whatsapp.net"
        out=switch.verify_inventory(inv,bot,relay)
        self.assertEqual(out["status"],"BLOCKED_ACCOUNT_BINDING")

    @unittest.skipIf(os.name == "nt", "Private mode 0600 gate runs on POSIX/WSL")
    def test_successful_cli_returns_zero_and_never_logs_phone(self):
        bot,relay,inv=fixture()
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            def private_file(name,data):
                path=root/name
                path.write_text(data,encoding="utf-8")
                path.chmod(0o600)
                return str(path)
            a=private_file("bot.env","\\n".join(f"{k}={v}" for k,v in bot.items()))
            b=private_file("relay.env","\\n".join(f"{k}={v}" for k,v in relay.items()))
            j=private_file("instances.json",json.dumps(inv))
            output=io.StringIO()
            with contextlib.redirect_stdout(output):
                code=switch.main(["--bot-env",a,"--relay-env",b,"--inventory-json",j])
            self.assertEqual(code,0)
            self.assertEqual(json.loads(output.getvalue())["status"],"PASS_BINDINGS_ONLY")
            self.assertNotIn(bot["BOT_ADS_PHONE"],output.getvalue())
            self.assertNotIn(bot["EVOLUTION_ADS_APIKEY"],output.getvalue())

    def test_switch_to_other_valid_line_requires_matching_inventory_uuid(self):
        bot,relay,inv=fixture()
        previous=dict(bot,BOT_ADS_PHONE="5548996425287",
                      BOT_ADS_INSTANCE_UUID="00000000-0000-4000-8000-000000000002")
        out=switch.verify_inventory(inv,bot,relay,previous)
        self.assertTrue(out["checks"]["phone_change_requested"])
        self.assertTrue(out["checks"]["uuid_change_requested"])
        self.assertEqual(out["status"],"PASS_BINDINGS_ONLY")
        bot["BOT_ADS_PHONE"]="5548999999999"
        bad=switch.verify_inventory(inv,bot,relay,previous)
        self.assertEqual(bad["status"],"BLOCKED_ACCOUNT_BINDING")
        self.assertFalse(bad["checks"]["provider_phone_matches"])

    def test_wrong_api_key_or_relay_or_shared_greeting_fails_closed(self):
        bot,relay,inv=fixture()
        for change in ("key","relay_uuid","relay_secret","same_greeting"):
            b=dict(bot)
            r=dict(relay)
            if change=="key":b["EVOLUTION_ADS_APIKEY"]="B"*35
            if change=="relay_uuid":r["RELAY_EXPECTED_INSTANCE_UUID"]="other"
            if change=="relay_secret":r["RELAY_BOT_URL"]="http://bot:8788/webhook/ads/wrong"
            if change=="same_greeting":b["BOT_GREETING_PHONE"]=b["BOT_ADS_PHONE"]
            got=switch.verify_inventory(inv,b,r)
            self.assertEqual(got["status"],"BLOCKED_ACCOUNT_BINDING",change)

    def test_live_authenticated_status_can_prove_token_when_inventory_omits_it(self):
        bot,relay,inv=fixture()
        del inv["data"][0]["token"]
        assert switch.verify_inventory(inv,bot,relay)["status"]=="BLOCKED_ACCOUNT_BINDING"
        result=switch.verify_inventory(inv,bot,relay,provider_auth_proof=True)
        self.assertEqual(result["status"],"PASS_BINDINGS_ONLY")
        self.assertTrue(result["checks"]["api_key_verified"])

    def test_live_provider_url_must_be_loopback(self):
        with self.assertRaises(ValueError):
            switch.verify_live_instance_token("http://example.com:14000","key","5548996425287")

    def test_missing_provider_token_cannot_fake_verification(self):
        bot,relay,inv=fixture()
        del inv["data"][0]["token"]
        got=switch.verify_inventory(inv,bot,relay)
        self.assertEqual(got["status"],"BLOCKED_ACCOUNT_BINDING")
        self.assertFalse(got["checks"]["api_key_verified"])

    def test_duplicate_instance_uuid_is_blocked(self):
        bot,relay,inv=fixture()
        inv["data"].append(dict(inv["data"][0]))
        got=switch.verify_inventory(inv,bot,relay)
        self.assertFalse(got["checks"]["unique_provider_uuid"])

    def test_old_outbox_blocks_switch_until_reconciled(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            uuid="00000000-0000-4000-8000-000000000002"
            task=root/("a"*64+".json")
            task.write_text(__import__("json").dumps({
                "instance_id":uuid,"state":"attempt_started",
                "phone_e164":"5548999991234",
                "step":"schedule_natal_crm"}),encoding="utf-8")
            task.chmod(0o600)
            report=switch.inspect_previous_outbox(root,uuid)
            self.assertFalse(report["old_outbox_clear"])
            self.assertEqual(report["unresolved_previous_ads_tasks"],1)
            self.assertNotIn("5548999991234",str(report))
            task.write_text(__import__("json").dumps({
                "instance_id":uuid,"state":"confirmed"}),encoding="utf-8")
            report=switch.inspect_previous_outbox(root,uuid)
            self.assertTrue(report["old_outbox_clear"])

    @unittest.skipIf(os.name == "nt", "POSIX 0600 permission check: use WSL/VPS for private preflight")
    def test_private_input_refuses_world_readable_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            path=Path(tmp)/"env"
            path.write_text("BOT_ADS_PHONE=5548999991234",encoding="utf-8")
            path.chmod(0o644)
            with self.assertRaises(ValueError):
                switch.read_private(path)
            path.chmod(0o600)
            self.assertIn("BOT_ADS_PHONE",switch.read_private(path))

    def test_env_parser_ignores_comments_and_invalid_keys(self):
        raw="# comment\nBOT_ADS_PHONE=5548999991234\nnot-a-key=secret\n"
        out=switch.parse_env(raw)
        self.assertEqual(out,{"BOT_ADS_PHONE":"5548999991234"})


if __name__=="__main__":
    unittest.main()
