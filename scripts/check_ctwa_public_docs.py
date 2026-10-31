#!/usr/bin/env python3
"""Offline documentation leak gate for the P0 CTWA GitHub runbook.

Production webhook paths can contain bearer capabilities. The exact ingress
secret must never be printed in Git source or reports. This gate scans the
published operational reports (not private runtime environment).
"""
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
DOCUMENTS = [
    "docs/AUDITORIA_VALIDACAO_CHATGPT_P0_20261031.md",
    "docs/RECIBO_ANTIGRAVITY_CTWA_NUMBER_SWITCH_20261031.md",
    "docs/CTWA_POST_CUTOVER_GATES_20261031.md",
]
SECRET_URL = re.compile(r"/webhook/(?:evolution|ads)/[0-9a-fA-F]{32,}")


def check(root=ROOT):
    unsafe = []
    for relpath in DOCUMENTS:
        file = root / relpath
        if not file.exists():
            # New post-cutover report may not be published on older revisions.
            continue
        if SECRET_URL.search(file.read_text(encoding="utf-8")):
            unsafe.append(relpath)
    return unsafe


if __name__ == "__main__":
    bad = check()
    if bad:
        print("BLOCKED_PUBLIC_WEBHOOK_CAPABILITY: redaction required in " + ", ".join(bad))
        sys.exit(2)
    print("PASS_PUBLIC_DOCS_NO_RAW_WEBHOOK_CAPABILITIES")
