#!/usr/bin/env python3
"""Private, READ-ONLY inventory of legacy HTTP 422 proposal tasks.

No network, writes, retry, CRM scheduling or customer PII in output.
A match in the local spool does NOT establish CRM's actual commercial status.
"""
from __future__ import annotations
from collections import Counter, defaultdict
import argparse
import json
from pathlib import Path
import sys


def audit(spool: Path):
    tasks_dir = spool / "tasks"
    if not tasks_dir.is_dir() or spool.is_symlink() or tasks_dir.is_symlink():
        raise ValueError("spool/tasks ausente ou inseguro")
    counts = Counter()
    historical = set()
    confirmed_by_phone = set()
    uncertain_by_phone = set()
    for path in tasks_dir.glob("*.json"):
        if path.is_symlink() or not path.is_file() or path.stat().st_size > (1 << 20):
            counts["files_invalid_or_oversized"] += 1
            continue
        try:
            rec = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, ValueError):
            counts["files_unreadable"] += 1
            continue
        if not isinstance(rec, dict) or rec.get("step") != "schedule_natal_crm":
            continue
        counts["crm_tasks"] += 1
        phone = str(rec.get("phone_e164") or "")
        state = str(rec.get("state") or "")
        err = str(rec.get("error") or "").lower()
        if state == "confirmed" and rec.get("external_id") and phone:
            confirmed_by_phone.add(phone)
        if state in ("uncertain", "attempt_started") and phone:
            uncertain_by_phone.add(phone)
        if "http 422" in err and ("native_entry_required" in err or "campaign" in err):
            counts["historical_422_candidate_tasks"] += 1
            if phone:
                historical.add(phone)
            else:
                counts["historical_422_without_phone"] += 1

    counts["historical_422_unique_phones"] = len(historical)
    counts["historical_422_with_some_confirmed_task_same_phone"] = len(historical & confirmed_by_phone)
    counts["historical_422_with_some_uncertain_task_same_phone"] = len(historical & uncertain_by_phone)
    counts["historical_422_without_confirmed_task_in_this_spool"] = len(historical - confirmed_by_phone)
    return dict(sorted(counts.items()))


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--spool", required=True, type=Path)
    args = p.parse_args()
    result = audit(args.spool)
    print(json.dumps(result, ensure_ascii=False, sort_keys=True))
    print("READ_ONLY: nao comprova situacao no banco CRM nem autoriza reenviar qualquer lead.")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, ValueError) as err:
        print("NO_GO_LEGACY_AUDIT: " + type(err).__name__, file=sys.stderr)
        sys.exit(2)
