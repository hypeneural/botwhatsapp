#!/usr/bin/env python3
"""Read-only aggregate identity audit of BotWhatsApp ingress spool.

No network access, no writes, never prints customer names, phones or LIDs.
"""
import argparse
from collections import Counter, defaultdict
import json
from pathlib import Path
import re
import sys

PHONE = re.compile(r"^55[0-9]{10,11}$")

def audit(spool):
    root = Path(spool)
    if not root.is_dir() or root.is_symlink():
        raise ValueError("spool absent or symlink")
    stats = Counter()
    lid_pairs = defaultdict(set)
    unresolved = set()
    for path in root.glob("*.json"):
        if path.is_symlink() or not path.is_file() or path.stat().st_size > (1 << 20):
            stats["skipped_nonstandard"] += 1
            continue
        try:
            row = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, ValueError):
            stats["skipped_invalid"] += 1
            continue
        decision = row.get("decision")
        if not isinstance(decision, dict) or not decision.get("message_id"):
            continue
        stats["events"] += 1
        phone = str(decision.get("phone_e164") or "")
        alias = str(decision.get("lid_alias") or decision.get("reply_jid") or "")
        name = str(decision.get("name") or "")
        if name and name.strip():
            stats["pushname_present"] += 1
        if PHONE.fullmatch(phone):
            stats["phone_verified_in_webhook"] += 1
            if alias.endswith("@lid"):
                lid_pairs[alias].add(phone)
        elif alias.endswith("@lid"):
            stats["lid_unresolved"] += 1
            unresolved.add(alias)
        else:
            stats["other_no_phone"] += 1
    for lid in unresolved:
        pairs = lid_pairs.get(lid, set())
        if len(pairs) == 1:
            stats["lid_eligible_for_exact_historical_recovery"] += 1
        elif len(pairs) > 1:
            stats["lid_conflicting_do_not_recover"] += 1
    return dict(sorted(stats.items()))


def main():
    p=argparse.ArgumentParser()
    p.add_argument("--spool",required=True)
    args=p.parse_args()
    stats=audit(args.spool)
    print("CTWA IDENTIDADE — auditoria somente leitura; agregados, nenhum telefone/nome/LID")
    print(json.dumps(stats,sort_keys=True,ensure_ascii=False))
    print("LID nao resolvido NUNCA deve ser convertido em telefone por seus digitos.")
    return 0


if __name__=="__main__":
    try:
        sys.exit(main())
    except (OSError,ValueError) as exc:
        print("NO_GO_IDENTIDADE: "+type(exc).__name__,file=sys.stderr)
        sys.exit(2)
