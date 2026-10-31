#!/usr/bin/env python3
"""Read-only fail-closed comparison of private Evolution Go sample vs bot contract.

NO provider calls, messages, replays, notifications, token or PII output.
Both inputs must be root/operator-only (0600). Detects a common incompatibility:
Evolution main docs show MESSAGE/data.key, BotWhatsApp core requires
Message/data.Info and instanceId. DO NOT normalize in production without proof.
"""
import argparse
import json
from pathlib import Path
import stat
import sys

def read_private(path, *, json_content=False):
    p = Path(path)
    st = p.stat()
    if not stat.S_ISREG(st.st_mode) or (sys.platform != 'win32' and st.st_mode & 0o077) or st.st_size > 2 * 1024 * 1024:
        raise ValueError("private_sample_required")
    raw = p.read_text(encoding="utf-8")
    return json.loads(raw) if json_content else raw.strip()


def check(event, expected):
    data = event.get("data")
    info = data.get("Info", {}) if isinstance(data, dict) else {}
    message = data.get("Message", {}) if isinstance(data, dict) else {}
    legacy_key = data.get("key", {}) if isinstance(data, dict) else {}
    supported = (
        event.get("event") == "Message" and
        event.get("instanceId") == expected and
        isinstance(info, dict) and isinstance(message, dict) and
        bool(info.get("ID")) and
        bool(info.get("Chat")) and
        ("IsFromMe" in info) and
        bool(message.get("conversation") or
             (message.get("extendedTextMessage") or {}).get("text"))
    )
    return {
        "status": "MATCH" if supported else "BLOCKED_CONTRACT",
        "event_case_accepted": event.get("event") == "Message",
        "instance_uuid_accepted": event.get("instanceId") == expected,
        "id_shape_accepted": bool(info.get("ID")),
        "chat_shape_accepted": bool(info.get("Chat")),
        "from_me_shape_accepted": "IsFromMe" in info,
        "message_shape_accepted": bool(message),
        "has_documented_go_key_shape": bool(legacy_key),
        "relay_eventkey_accepted": bool(event.get("instanceId") == expected
                                       and event.get("event")),
        "webhook_delivery_proven": False,
    }


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--sample-file", required=True)
    p.add_argument("--expected-uuid-file", required=True)
    args = p.parse_args(argv)
    try:
        event = read_private(args.sample_file, json_content=True)
        expected = read_private(args.expected_uuid_file)
        if not isinstance(event, dict) or not expected:
            raise ValueError("invalid_sample")
        result = check(event, expected)
        print(json.dumps(result, sort_keys=True))
        return 0 if result["status"] == "MATCH" else 2
    except Exception as exc:
        print(json.dumps({"status": "BLOCKED_CONTRACT", "reason": type(exc).__name__}))
        return 2


if __name__ == "__main__":
    sys.exit(main())
