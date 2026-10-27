#!/usr/bin/env python3
"""Enrich private BotWhatsApp panel labels from existing CRM read-only API.

Reads only local contact state and GET /api/v1/extension/customer-context?phone=.
Requires existing Sanctum ability extension:customer-read; never schedules a
proposal, changes CRM, sends WhatsApp or prints a customer's identity.
Only --apply writes private cache in the BotWhatsApp spool.
"""
from __future__ import annotations

import argparse
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import re
import stat
import sys
import tempfile
import time
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import HTTPRedirectHandler, ProxyHandler, Request, build_opener

ENDPOINT = "https://evydencia.com/api/v1/extension/customer-context?"
E164 = re.compile(r"^55[0-9]{10,11}$")


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def token_from_env(envpath: Path):
    if envpath.is_symlink() or not envpath.is_file() or stat.S_IMODE(envpath.stat().st_mode) & 0o007:
        raise ValueError("env privado nao existe ou esta publico")
    values = []
    for line in envpath.read_text(encoding="utf-8").splitlines():
        if line.startswith("EVIDENCIA_CRM_PROPOSAL_SANCTUM_TOKEN="):
            values.append(line.split("=", 1)[1].strip().strip("'\""))
    if len(values) != 1 or not re.fullmatch(r"[1-9][0-9]*\|[a-zA-Z0-9]{40,}", values[0]):
        raise ValueError("token Sanctum do CRM invalido ou ausente")
    return values[0]


def token_from_private_file(path: Path):
    """Use an existing extension token with customer-read, never expose value."""
    if path.is_symlink() or not path.is_file() or stat.S_IMODE(path.stat().st_mode) & 0o077:
        raise ValueError("arquivo da credencial de leitura nao e privado (exigir 0600)")
    value = path.read_text(encoding="utf-8").strip()
    if not re.fullmatch(r"[1-9][0-9]*\|[a-zA-Z0-9]{40,}", value):
        raise ValueError("credencial de leitura nao tem formato Sanctum")
    return value


def contact_numbers(spool: Path):
    contacts = spool / "contacts"
    if not contacts.is_dir() or spool.is_symlink():
        raise ValueError("spool/contacts ausente; nao criar volume falso")
    found = set()
    for file in contacts.glob("*.json"):
        if file.is_symlink() or file.stat().st_size > 32768:
            continue
        try:
            row = json.loads(file.read_text(encoding="utf-8"))
            phone = row.get("phone_e164")
            if isinstance(phone, str) and E164.fullmatch(phone):
                found.add(phone)
        except (OSError, ValueError):
            pass
    return sorted(found)


def lookup(opener, phone, token):
    url = ENDPOINT + urlencode({"phone": phone[2:]})
    req = Request(url, headers={
        "Authorization": "Bearer " + token,
        "Accept": "application/json",
        "User-Agent": "BotWhatsApp-Panel/1.0 (CTWA CRM Name Cache)",
    }, method="GET")
    try:
        with opener.open(req, timeout=6) as response:
            if response.status != 200:
                raise RuntimeError("unexpected response from CRM")
            raw = response.read(8192)
        data = json.loads(raw)
        if not isinstance(data, dict) or not isinstance(data.get("data"), dict):
            raise RuntimeError("schema CRM inesperado")
        obj = data["data"]
        state = (obj.get("lookup") or {}).get("status")
        if state != "matched":
            return None
        person = obj.get("customer") or {}
        name = person.get("display_name")
        if not isinstance(name, str):
            return None
        name = name.strip()
        if not name or name == "Cliente" or len(name) > 80 or any(ch in name for ch in "\r\n;"):
            return None
        return name
    except HTTPError as exc:
        if exc.code in (401, 403, 429, 503):
            raise RuntimeError("CRM nao autorizou leitura ou limitou taxa") from None
        if exc.code in (400, 422):
            return None
        raise RuntimeError("falha de consulta CRM") from None


def load_cache(path: Path):
    if not path.is_file() or path.is_symlink() or path.stat().st_size > 2 * 1024 * 1024:
        return {}
    if stat.S_IMODE(path.stat().st_mode) & 0o007:
        raise ValueError("cache CRM acessivel ao publico")
    raw = json.loads(path.read_text(encoding="utf-8"))
    rows = raw.get("entries", {})
    if not isinstance(rows, dict):
        return {}
    now = time.time()
    result = {}
    for phone, item in rows.items():
        if (E164.fullmatch(phone) and isinstance(item, dict) and
                isinstance(item.get("name"), str) and isinstance(item.get("at"), (int, float)) and
                now - item["at"] < 48 * 3600 and 0 <= now - item["at"]):
            result[phone] = item
    return result


def save_cache(path, entries):
    if path.is_symlink():
        raise ValueError("destino cache nao pode ser symlink")
    payload = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "names": {k: v["name"] for k, v in entries.items()},
        "entries": entries,
    }
    fd, tmp = tempfile.mkstemp(prefix=".ctwa-crm-name-", dir=path.parent)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as stream:
            json.dump(payload, stream, ensure_ascii=False, separators=(",", ":"))
            stream.flush()
            os.fsync(stream.fileno())
        os.chmod(tmp, 0o600)
        os.replace(tmp, path)
    finally:
        if os.path.exists(tmp):
            os.unlink(tmp)


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--env", required=True)
    p.add_argument("--spool", required=True)
    p.add_argument("--limit", type=int, default=40)
    p.add_argument("--read-token-file", type=Path,
                   help="token Sanctum EXISTENTE com extension:customer-read, em arquivo 0600")
    p.add_argument("--apply", action="store_true")
    args = p.parse_args()
    if args.limit < 1 or args.limit > 100:
        p.error("--limit entre 1 e 100")
    envpath, spool = Path(args.env), Path(args.spool)
    token = (token_from_private_file(args.read_token_file)
             if args.read_token_file is not None else token_from_env(envpath))
    numbers = contact_numbers(spool)
    cachepath = spool / "crm-names-cache.json"
    entries = load_cache(cachepath)
    pending = [phone for phone in numbers if phone not in entries]
    print("CTWA CRM NOMES: contatos=", len(numbers), " pendentes=", len(pending),
          " modo=", "WRITE_PRIVATE_CACHE" if args.apply else "PREVIEW_ONLY")
    if not args.apply:
        return 0
    client = build_opener(ProxyHandler({}), NoRedirect())
    updated = 0
    nonmatched = 0
    for phone in pending[:args.limit]:
        name = lookup(client, phone, token)
        if name:
            entries[phone] = {"name": name, "at": time.time()}
            updated += 1
        else:
            nonmatched += 1
        time.sleep(0.15)
    if entries:
        save_cache(cachepath, entries)
    print("RESULTADO: CRM nomes correspondentes=", updated, " sem vinculo unico=", nonmatched,
          " cache privado 0600; NENHUM POST enviado")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, ValueError, RuntimeError, URLError) as exc:
        print("NO_GO CRM NAME CACHE: " + type(exc).__name__ + " (credenciais/PII ocultos)", file=sys.stderr)
        sys.exit(2)
