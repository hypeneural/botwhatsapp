#!/usr/bin/env python3
"""CTWA panel forensic smoke test (GET/HEAD only; no WhatsApp/CRM POST).

Examples:
 python3 scripts/audit-ctwa-panel-readonly.py --env /srv/platform/stacks/botwhatsapp/.env.production
 python3 scripts/audit-ctwa-panel-readonly.py --env /secure/.env.production --base-url https://private.example
No credential is printed. A local .env is read privately and only the panel
password is used for Basic authentication on exactly the selected origin.
"""
from __future__ import annotations

import argparse
import base64
import json
import os
from pathlib import Path
import re
import stat
import sys
from urllib.error import HTTPError, URLError
from urllib.parse import urlsplit
from urllib.request import HTTPBasicAuthHandler, HTTPRedirectHandler, Request, build_opener


class RejectRedirects(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def read_panel_secret(envfile):
    target = Path(envfile)
    if target.is_symlink() or not target.is_file():
        raise ValueError("env de produção inexistente ou symlink")
    if stat.S_IMODE(target.stat().st_mode) & 0o007:
        raise ValueError("env de produção tem acesso world")
    matches = []
    for line in target.read_text(encoding="utf-8").splitlines():
        m = re.fullmatch(r"BOT_DASHBOARD_TOKEN=(.*)", line.strip())
        if m:
            matches.append(m.group(1).strip().strip("'\""))
    if len(matches) != 1 or len(matches[0]) < 32 or "\r" in matches[0]:
        raise ValueError("token painel ausente, duplicado ou curto")
    return matches[0]


def normalized_base(value):
    u = urlsplit(value)
    if (u.scheme not in ("http", "https") or not u.hostname or u.path not in ("", "/")
            or u.username is not None or u.password is not None or u.query or u.fragment):
        raise ValueError("base URL invalida; use somente origem scheme://host:port")
    if u.scheme == "http" and u.hostname not in ("localhost", "127.0.0.1", "::1"):
        raise ValueError("Basic Auth por HTTP remoto proibido; usar HTTPS privado")
    return value.rstrip("/")


def request_read(opener, base, path, token=None, method="GET"):
    headers = {"Accept": "application/json, text/html"}
    if token is not None:
        basic = base64.b64encode(("gestor:" + token).encode("utf-8")).decode("ascii")
        headers["Authorization"] = "Basic " + basic
    req = Request(base + path, method=method, headers=headers)
    try:
        resp = opener.open(req, timeout=7)
        with resp:
            return resp.status, resp.headers.get("Content-Type", ""), resp.read(1024 * 1024)
    except HTTPError as err:
        with err:
            return err.code, err.headers.get("Content-Type", ""), err.read(512)


def run(base, token):
    opener = build_opener(RejectRedirects())
    checks = []
    def check(label, condition):
        checks.append((label, bool(condition)))
        print(("PASS" if condition else "FAIL") + ": " + label)

    code, _, body = request_read(opener, base, "/healthz")
    check("healthz HTTP 200", code == 200 and body.strip() == b"ok")
    code, _, body = request_read(opener, base, "/readyz")
    check("readyz HTTP 200 (nao atesta conectividade externa)", code == 200 and body.strip() == b"ready")
    for route in ["/dashboard", "/painel", "/api/dashboard/events?limit=10", "/api/dashboard/contacts"]:
        code, _, _ = request_read(opener, base, route)
        check(route + " sem senha bloqueado (401)", code == 401)
    code, ct, body = request_read(opener, base, "/dashboard", token)
    html = body.decode("utf-8", errors="replace") if code == 200 else ""
    check("dashboard com Basic Auth HTTP 200 e HTML", code == 200 and "text/html" in ct and "<html" in html.lower())
    check("URLs absolutas same-origin no JS servido",
          "new URL(path, window.location.origin)" in html and "credentials:'same-origin'" in html)
    code, _, _ = request_read(opener, base, "/dashboard", token, method="HEAD")
    check("dashboard HEAD autenticado", code == 200)
    code, ct, body = request_read(opener, base, "/api/dashboard/events?limit=10", token)
    events_ok = False
    received = None
    if code == 200 and "json" in ct:
        try:
            payload = json.loads(body)
            events_ok = isinstance(payload.get("events"), list) and isinstance(payload.get("stats"), dict)
            received = payload["stats"].get("received") if events_ok else None
        except (ValueError, TypeError, KeyError):
            pass
    check("API eventos JSON com schema correto", events_ok)
    code, ct, body = request_read(opener, base, "/api/dashboard/contacts", token)
    contacts_ok = False
    if code == 200 and "json" in ct:
        try:
            payload = json.loads(body)
            contacts_ok = isinstance(payload.get("contacts"), list) and isinstance(payload.get("stats"), dict)
        except (ValueError, TypeError):
            pass
    check("API contatos JSON com schema correto", contacts_ok)
    print("INFO: eventos visiveis em amostra (quantidade apenas):", received if isinstance(received, int) else "indisponivel")
    ok = all(p for _, p in checks)
    print("RESULTADO:", "PASS_PANEL_HTTP" if ok else "NO_GO_PANEL_HTTP",
          "— ainda requer browser HTTPS privado, Evolucao e canario real.")
    return 0 if ok else 2


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--env", required=True, help="env privado atual do BotWhatsApp")
    parser.add_argument("--base-url", default="http://127.0.0.1:8788",
                        help="Origem única do painel; HTTP apenas localhost, externo exige HTTPS")
    options = parser.parse_args()
    base = normalized_base(options.base_url)
    secret = read_panel_secret(options.env)
    return run(base, secret)


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (ValueError, OSError, URLError) as exc:
        print("NO_GO_PANEL_HTTP: preflight ou conectividade falhou (" + type(exc).__name__ + ")",
              file=sys.stderr)
        sys.exit(2)
