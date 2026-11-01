#!/usr/bin/env python3
"""Inspect relay spool safety without reading/printing WhatsApp payloads.

Requires explicit path to the existing relay spool directory. Read-only.
Always redact file names, phone numbers, event bodies and secret-bearing URLs.
"""
import argparse
from collections import Counter
from datetime import datetime, timezone
import json
from pathlib import Path
import stat
import sys

def inspect(root: Path, max_events=None, max_bytes=None):
    if not root.is_dir() or root.is_symlink():
        raise ValueError("invalid_spool")
    counts = Counter()
    oldest = None
    total_bytes = 0
    for path in root.iterdir():
        if path.name.startswith(".relay-tmp-"):
            counts["temporary_files"] += 1
            continue
        if path.suffix != ".json":
            counts["other_files"] += 1
            continue
        info = path.lstat()
        if not stat.S_ISREG(info.st_mode) or (sys.platform != 'win32' and info.st_mode & 0o077):
            counts["unsafe_permissions"] += 1
            continue
        counts["records"] += 1
        total_bytes += info.st_size
        if info.st_size > 36 * 1024 * 1024:
            counts["oversize"] += 1
            continue
        try:
            record = json.loads(path.read_text(encoding="utf-8"))
            if record.get("key", "") + ".json" != path.name:
                counts["corrupt"] += 1
                continue
            done = bool(record.get("completed_at"))
            status = bool(record.get("status_done"))
            bot = bool(record.get("bot_done"))
            body = bool(record.get("body"))
            if done:
                counts["completed"] += 1
                if body or not (status and bot):
                    counts["corrupt"] += 1
            else:
                counts["pending"] += 1
                counts["pending_status"] += int(not status)
                counts["pending_bot"] += int(not bot)
                if not body:
                    counts["corrupt"] += 1
                created = record.get("created_at")
                if created:
                    ts = datetime.fromisoformat(created.replace("Z", "+00:00"))
                    oldest = min(oldest, ts) if oldest else ts
        except (ValueError, UnicodeError, OSError, TypeError):
            counts["corrupt"] += 1
    age = None
    if oldest:
        age = int((datetime.now(timezone.utc) - oldest).total_seconds())
    risk = counts["corrupt"] + counts["unsafe_permissions"] + counts["oversize"]
    capacity = {}
    if max_events is not None and max_events > 0:
        capacity["event_count_limit"] = max_events
        capacity["event_count_used_ratio"] = round(counts["records"] / max_events, 4)
    if max_bytes is not None and max_bytes > 0:
        capacity["byte_limit"] = max_bytes
        capacity["bytes_used_ratio"] = round(total_bytes / max_bytes, 4)
    saturation = any(value >= 0.90 for k, value in capacity.items()
                     if k.endswith("_ratio"))
    status = "BLOCKED" if risk else "CAPACITY_WARNING" if saturation else "INVENTORY_ONLY"
    return {"status": status, "counts": dict(counts), "bytes": total_bytes,
            "capacity": capacity, "oldest_pending_age_seconds": age,
            "delivery_proven": False, "runtime_contract_proven": False}


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--spool", required=True)
    p.add_argument("--max-events", type=int, help="actual RELAY_MAX_EVENTS from runtime")
    p.add_argument("--max-bytes", type=int, help="actual RELAY_MAX_SPOOL_BYTES from runtime")
    args = p.parse_args(argv)
    try:
        result = inspect(Path(args.spool), args.max_events, args.max_bytes)
        print(json.dumps(result, sort_keys=True))
        return 0 if result["status"] == "INVENTORY_ONLY" else 2
    except Exception as exc:
        print(json.dumps({"status": "BLOCKED", "reason": type(exc).__name__}))
        return 2


if __name__ == "__main__":
    sys.exit(main())
