#!/usr/bin/env python3
"""CTWA ↔ Laravel contract drift gate. Static, read-only, no HTTP/CRM credentials.

By default checks only the bot's Go client and canonic audit document.
Optionally provide a local read-only CRM checkout via --crm-root to
cross-check the Laravel source before operator runs any live canary.
"""
from __future__ import annotations
import argparse
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
NATIVE = "api/v1/extension/proposals/schedule"
CONTEXT = "api/v1/extension/customer-context"


def validate(bot_root: Path, crm_root: Path | None = None):
    errors = []
    required = {
        "internal/crm/native.go": [
            '/api/v1/extension/proposals/schedule',
            'json:"request_id"',
            'json:"phone"',
            'json:"name"',
            'json:"resend_mode"',
            'X-Idempotency-Key',
            'http.MethodPost',
            'receipt.Data.Status != "scheduled"',
        ],
        "scripts/sync-ctwa-crm-names-readonly.py": [
            '/api/v1/extension/customer-context?',
            'method="GET"',
            'extension:customer-read',
        ],
        "docs/AUDITORIA-VALIDACAO-CHATGPT-20261027.md": [
            "**CORREÇÃO P0:**",
            "**POST**",
            "/api/v1/extension/proposals/schedule",
            "/api/v1/extension/customer-context",
            "HTTP 201",
            "HTTP 200",
            "NÃO",
        ],
    }
    for file, markers in required.items():
        path = bot_root / file
        if not path.is_file():
            errors.append("bot path absent: " + file)
            continue
        body = path.read_text(encoding="utf-8")
        errors.extend("bot missing %s in %s" % (marker, file)
                      for marker in markers if marker not in body)

    if crm_root is not None:
        laravel = {
            "routes/extension-api.php": [
                "proposals/schedule",
                "customer-context",
                "ExtensionAbilities::PROPOSAL_WRITE",
                "ExtensionAbilities::CUSTOMER_READ",
                "throttle:extension-write",
                "throttle:extension-read",
            ],
            "app/Http/Requests/Api/Extension/ProposalScheduleRequest.php": [
                "'request_id' => ['required', 'uuid']",
                "'phone' => ['required'",
                "'name' => ['present', 'nullable'",
                "X-Idempotency-Key",
            ],
            "app/Http/Controllers/Api/Extension/ProposalScheduleController.php": [
                "'status' => 'scheduled'",
                "'dispatch_uuid'",
                "isReplay() ? 200 : 201",
            ],
            "app/Services/Proposals/ScheduleProposal.php": [
                "where('request_id'",
                "lockForUpdate()",
                "duplicate_recent_dispatch",
                "replayOrThrow",
            ],
        }
        for file, markers in laravel.items():
            path = crm_root / file
            if not path.is_file():
                errors.append("CRM checkout path absent: " + file)
                continue
            body = path.read_text(encoding="utf-8")
            errors.extend("CRM missing %s in %s" % (marker, file)
                          for marker in markers if marker not in body)
    return errors


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--crm-root", type=Path, help="checkout Laravel local somente leitura")
    args = p.parse_args()
    problems = validate(ROOT, args.crm_root)
    if problems:
        for item in problems:
            print("FAIL_CONTRACT: " + item)
        print("NO_GO: auditar contrato real antes de deploy/canario")
        return 2
    print("PASS_STATIC_CONTRACT: Bot routes/request/response and canonical docs consistent"
          + (" with local Laravel checkout" if args.crm_root else "; Laravel repo not inspected by this check"))
    print("INFO: static gate does not confirm database, token ability, deployment or delivery")
    return 0


if __name__ == "__main__":
    sys.exit(main())
