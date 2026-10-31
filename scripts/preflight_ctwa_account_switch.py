#!/usr/bin/env python3
"""Fail-closed offline preflight for switching the CTWA Ads business account.

Reads operator-only 0600 files; only --verify-live-status makes one read-only\nloopback GET /instance/status; it never sends or changes anything. The
Evolution inventory file must be a privately captured GET /instance/all JSON
from the target VPS and version. Nothing sensitive is printed, including
numbers, UUIDs, JIDs, URLs, tokens and ad copy.
"""
import argparse
import hashlib
import hmac
import json
from pathlib import Path
import re
import stat
import sys
from urllib.parse import urlsplit
from urllib.request import Request, urlopen

E164_BR = re.compile(r"^55[0-9]{10,11}$")
UUID_RE = re.compile(r"^[A-Za-z0-9-]{8,128}$")


def read_private(path, *, as_json=False):
    p = Path(path)
    info = p.stat()
    if not stat.S_ISREG(info.st_mode) or info.st_mode & 0o077 or info.st_size > 4 * 1024 * 1024:
        raise ValueError("private_file_permissions_or_size")
    raw = p.read_text(encoding="utf-8")
    return json.loads(raw) if as_json else raw


def parse_env(source):
    result = {}
    for line in source.splitlines():
        stripped = line.strip()
        if not stripped or stripped.startswith("#") or "=" not in stripped:
            continue
        key, val = stripped.split("=", 1)
        key = key.strip()
        if key and re.fullmatch(r"[A-Z][A-Z0-9_]*", key):
            result[key] = val.strip().strip("'").strip('"')
    return result


def from_jid(value):
    if not isinstance(value, str) or not value.endswith("@s.whatsapp.net"):
        return ""
    raw = value.split("@", 1)[0]
    parts = raw.split(":")
    if len(parts) > 2 or (len(parts) == 2 and
            (not re.fullmatch(r"[0-9]{1,3}", parts[1]))):
        return ""
    number = parts[0]
    return number if E164_BR.fullmatch(number) else ""


def verify_live_instance_token(base_url, token, expected_phone):
    """Read-only GET authenticated with candidate token; only local Evolution."""
    u = urlsplit(base_url)
    if (u.scheme != "http" or u.hostname not in ("127.0.0.1", "localhost")
            or u.username or u.password or u.query or u.fragment):
        raise ValueError("provider_url_must_be_local")
    req = Request(base_url.rstrip("/") + "/instance/status",
                  headers={"apikey":token, "Accept":"application/json"},
                  method="GET")
    with urlopen(req, timeout=6) as response:
        if response.status != 200:
            return False
        content = json.load(response)
    status = content.get("data", content) if isinstance(content, dict) else {}
    if not isinstance(status, dict):
        return False
    return (status.get("connected") is True
            and status.get("loggedIn", True) is True
            and from_jid(status.get("myJid") or status.get("jid") or "") == expected_phone)


def verify_inventory(inventory, bot, relay, previous=None, provider_auth_proof=False):
    checks = {}
    phone = bot.get("BOT_ADS_PHONE", "")
    uuid = bot.get("BOT_ADS_INSTANCE_UUID", "")
    api_key = bot.get("EVOLUTION_ADS_APIKEY", "")
    bot_secret = bot.get("BOT_ADS_WEBHOOK_SECRET", "")
    checks["phone_format_valid"] = bool(E164_BR.fullmatch(phone))
    checks["uuid_present"] = bool(UUID_RE.fullmatch(uuid))
    checks["bot_key_present"] = len(api_key) >= 16
    checks["bot_webhook_secret_present"] = len(bot_secret) >= 32
    checks["greeting_number_distinct"] = (not bot.get("BOT_GREETING_PHONE") or
                                          phone != bot.get("BOT_GREETING_PHONE"))
    checks["greeting_uuid_distinct"] = (not bot.get("BOT_GREETING_INSTANCE_UUID") or
                                        uuid != bot.get("BOT_GREETING_INSTANCE_UUID"))

    instances = inventory.get("data") if isinstance(inventory, dict) else None
    if not isinstance(instances, list):
        instances = []
    matching = [
        item for item in instances if isinstance(item, dict)
        and any(item.get(key) == uuid for key in ("id", "instanceId", "uuid"))
    ]
    checks["unique_provider_uuid"] = len(matching) == 1
    inst = matching[0] if len(matching) == 1 else {}
    paired_jid = bot.get("BOT_ADS_PAIRED_JID", "")
    paired_phone = from_jid(paired_jid) if paired_jid else phone
    checks["paired_jid_valid"] = (not paired_jid or bool(from_jid(paired_jid)))
    # The provider's registered JID may differ from the advertised E.164
    # in Brazil. Only an explicit, verified paired JID can bridge them.
    checks["provider_phone_matches"] = (
        bool(paired_phone) and
        from_jid(inst.get("jid") or inst.get("myJid") or "") == paired_phone
    )
    checks["paired_alias_separate_from_greeting"] = (
        not bot.get("BOT_GREETING_PHONE") or
        paired_phone != bot.get("BOT_GREETING_PHONE")
    )
    checks["provider_connected"] = inst.get("connected") is True
    provider_key = inst.get("token")
    inventory_key_verified = isinstance(provider_key, str) and bool(api_key) and hmac.compare_digest(
        hashlib.sha256(provider_key.encode()).digest(),
        hashlib.sha256(api_key.encode()).digest()
    )
    checks["api_key_verified"] = inventory_key_verified or bool(provider_auth_proof)
    # Some provider builds deliberately omit the token in GET /instance/all.
    # A missing value does NOT prove the credential is valid.
    checks["relay_uuid_matches"] = relay.get("RELAY_EXPECTED_INSTANCE_UUID") == uuid
    relay_bot_url = relay.get("RELAY_BOT_URL", "")
    try:
        u = urlsplit(relay_bot_url)
        checks["relay_bot_destination_valid"] = (
            u.scheme == "http" and bool(u.hostname)
            and not u.username and not u.password and not u.query and not u.fragment
            and u.path == "/webhook/ads/" + bot_secret
        )
    except ValueError:
        checks["relay_bot_destination_valid"] = False
    checks["relay_ingress_secret_present"] = len(relay.get("RELAY_INGRESS_SECRET", "")) >= 32
    checks["relay_status_destination_present"] = bool(relay.get("RELAY_STATUS_URL"))
    checks["relay_enabled"] = relay.get("RELAY_ENABLED") == "true"
    if previous is not None:
        checks["phone_change_requested"] = previous.get("BOT_ADS_PHONE") != phone
        checks["uuid_change_requested"] = previous.get("BOT_ADS_INSTANCE_UUID") != uuid
        # A phone change without a new provider UUID might indicate spoofing
        # or a changed JID; provider inventory still decides eligibility.
    result = {
        "status": ("PASS_BINDINGS_ONLY" if all(
            value for key, value in checks.items()
            if key not in ("phone_change_requested", "uuid_change_requested")
        ) else "BLOCKED_ACCOUNT_BINDING"),
        "checks": checks,
        "provider_webhook_present": bool(inst.get("webhook")),
        "provider_event_subscription_present": bool(inst.get("events") or inst.get("webhookEvents")),
        "changes_performed": False,
        "provider_delivery_proven": False,
        "crm_dispatch_proven": False,
    }
    return result


def inspect_previous_outbox(tasks_dir, old_uuid):
    """Count unresolved tasks from the former Ads instance, without PII."""
    root = Path(tasks_dir)
    if not root.is_dir() or root.is_symlink():
        raise ValueError("outbox_directory_invalid")
    unresolved = 0
    scanned = 0
    for item in root.iterdir():
        if item.suffix != ".json":
            continue
        info = item.lstat()
        if not stat.S_ISREG(info.st_mode) or (sys.platform != 'win32' and info.st_mode & 0o077) or info.st_size > 512 * 1024:
            raise ValueError("outbox_record_unsafe")
        row = json.loads(item.read_text(encoding="utf-8"))
        scanned += 1
        if row.get("instance_id") == old_uuid and row.get("state") in (
                "pending", "attempt_started", "uncertain"):
            unresolved += 1
    return {"unresolved_previous_ads_tasks": unresolved,
            "tasks_scanned": scanned, "old_outbox_clear": unresolved == 0}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--bot-env", required=True)
    p.add_argument("--relay-env", required=True)
    p.add_argument("--inventory-json", required=True)
    p.add_argument("--previous-bot-env")
    p.add_argument("--verify-live-status", action="store_true",
                   help="GET /instance/status using candidate APIKEY on local Evolution, never POST")
    p.add_argument("--provider-base-url", default="http://127.0.0.1:14000")
    p.add_argument("--outbox-tasks-dir",
                   help="Required with --previous-bot-env for an account change; read-only.")
    args = p.parse_args(argv)
    try:
        bot = parse_env(read_private(args.bot_env))
        relay = parse_env(read_private(args.relay_env))
        inventory = read_private(args.inventory_json, as_json=True)
        prev = parse_env(read_private(args.previous_bot_env)) if args.previous_bot_env else None
        provider_auth_proof = False
        if args.verify_live_status:
            provider_auth_proof = verify_live_instance_token(
                args.provider_base_url,
                bot.get("EVOLUTION_ADS_APIKEY", ""),
                from_jid(bot.get("BOT_ADS_PAIRED_JID", "")) or bot.get("BOT_ADS_PHONE", ""),
            )
        result = verify_inventory(
            inventory, bot, relay, previous=prev, provider_auth_proof=provider_auth_proof)
        changed = prev is not None and (
            prev.get("BOT_ADS_PHONE") != bot.get("BOT_ADS_PHONE") or
            prev.get("BOT_ADS_INSTANCE_UUID") != bot.get("BOT_ADS_INSTANCE_UUID"))
        if changed:
            if not args.outbox_tasks_dir:
                result["status"] = "BLOCKED_PREVIOUS_OUTBOX_NOT_AUDITED"
            else:
                queue = inspect_previous_outbox(
                    args.outbox_tasks_dir, prev.get("BOT_ADS_INSTANCE_UUID", ""))
                result.update(queue)
                if not queue["old_outbox_clear"]:
                    result["status"] = "BLOCKED_PREVIOUS_OUTBOX_UNRESOLVED"
        # Identity/preflight success never authorizes a webhook change by itself.
        result["deployment_authorized"] = False
        print(json.dumps(result, sort_keys=True))
        return 0 if result["status"] == "PASS_BINDINGS_ONLY" else 2
    except Exception as e:
        print(json.dumps({"status": "BLOCKED_ACCOUNT_BINDING", "reason": type(e).__name__}))
        return 2


if __name__ == "__main__":
    sys.exit(main())
