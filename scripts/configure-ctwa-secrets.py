#!/usr/bin/env python3
"""Provisionamento LOCAL de segredos CTWA; nunca imprime tokens ou contata APIs.

Modo default e apenas plano. --apply altera SOMENTE o env_file explicitamente
indicado; cria backup 0600 e troca atomica. Nunca gera token Sanctum ficticio.
"""
from __future__ import annotations

import argparse
from datetime import datetime, timezone
import os
from pathlib import Path
import re
import secrets
import stat
import sys
import tempfile

KEY_DASH = "BOT_DASHBOARD_TOKEN"
KEY_CRM = "EVIDENCIA_CRM_PROPOSAL_SANCTUM_TOKEN"
SANCTUM_FORMAT = re.compile(r"^[1-9][0-9]*\|[a-zA-Z0-9]{40,}$")


def read_sanctum_file(path: Path) -> str:
    if path.is_symlink() or not path.is_file():
        raise ValueError("Arquivo Sanctum privado ausente ou symlink")
    mode = stat.S_IMODE(path.stat().st_mode)
    if mode & 0o077:
        raise ValueError("Arquivo Sanctum deve estar protegido (0600 ou mais restrito)")
    raw = path.read_text(encoding="utf-8").strip()
    if not SANCTUM_FORMAT.fullmatch(raw):
        raise ValueError("Token nao tem o formato Laravel Sanctum ID|segredo; NAO gerar valor aleatorio")
    return raw


def parse_env(content: str):
    entries = {}
    for line in content.splitlines():
        match = re.match(r"^([A-Z][A-Z0-9_]*)=(.*)$", line)
        if not match:
            continue
        key, value = match.groups()
        if key in (KEY_DASH, KEY_CRM):
            if key in entries:
                raise ValueError("Variavel duplicada no ambiente: " + key)
            entries[key] = value.strip().strip("'\"")
    return entries


def prepare(original: str, sanctum: str | None, rotate_dashboard: bool, rotate_sanctum: bool):
    current = parse_env(original)
    dashboard_set = bool(current.get(KEY_DASH))
    crm_set = bool(current.get(KEY_CRM))
    changes = {}
    if rotate_dashboard or not dashboard_set:
        changes[KEY_DASH] = secrets.token_hex(32)
    if sanctum is not None and (rotate_sanctum or not crm_set):
        changes[KEY_CRM] = sanctum
    if crm_set and sanctum is None:
        if not SANCTUM_FORMAT.fullmatch(current[KEY_CRM]):
            raise ValueError("Token Sanctum existente nao confere formato: requer emissao CRM")
    elif crm_set and sanctum is not None and not rotate_sanctum:
        # Nao substituir silenciosamente token produtivo ja instalado.
        if not SANCTUM_FORMAT.fullmatch(current[KEY_CRM]):
            raise ValueError("Token Sanctum existente invalido: -rotate-sanctum explicito necessario")
    result = original
    for key, value in changes.items():
        pattern = re.compile(r"^" + re.escape(key) + r"=.*$", re.MULTILINE)
        if pattern.search(result):
            result = pattern.sub(lambda _: key + "=" + value, result)
        else:
            result = result.rstrip("\r\n") + "\n" + key + "=" + value + "\n"
    return result, {
        "dashboard": "GENERATE_NEW" if KEY_DASH in changes else ("PRESENT" if dashboard_set else "MISSING"),
        "sanctum": ("IMPORT_SCOPED" if KEY_CRM in changes else "PRESENT" if crm_set else "MISSING_EXTERNAL_CRM_TOKEN"),
        "changed": bool(changes),
    }


def apply_atomic(path: Path, new_text: str, old_bytes: bytes) -> Path:
    parent = path.parent
    when = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    backup = parent / (path.name + ".backup-ctwa-" + when)
    backup_fd = os.open(backup, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600)
    try:
        with os.fdopen(backup_fd, "wb") as stream:
            stream.write(old_bytes)
            stream.flush()
            os.fsync(stream.fileno())
    except BaseException:
        backup.unlink(missing_ok=True)
        raise
    descriptor, temp_name = tempfile.mkstemp(prefix=".ctwa-env-", dir=parent)
    try:
        with os.fdopen(descriptor, "wb") as stream:
            stream.write(new_text.encode("utf-8"))
            stream.flush()
            os.fsync(stream.fileno())
        original_stat = path.stat()
        os.chmod(temp_name, 0o600)
        if hasattr(os, "chown") and os.geteuid() == 0:
            os.chown(temp_name, original_stat.st_uid, original_stat.st_gid)
        os.replace(temp_name, path)
        return backup
    finally:
        if os.path.exists(temp_name):
            os.unlink(temp_name)


def main():
    ap = argparse.ArgumentParser(description="Configura CTWA sem vazar valores de tokens")
    ap.add_argument("env_file", type=Path, help="Caminho real da .env.production")
    ap.add_argument("--sanctum-file", type=Path, help="Arquivo privado ja emitido pelo CRM")
    ap.add_argument("--apply", action="store_true", help="Confirma gravacao local atomica e backup")
    ap.add_argument("--rotate-dashboard", action="store_true")
    ap.add_argument("--rotate-sanctum", action="store_true")
    args = ap.parse_args()
    if args.rotate_sanctum and args.sanctum_file is None:
        ap.error("--rotate-sanctum exige --sanctum-file")
    env = args.env_file.resolve(strict=True)
    if args.env_file.is_symlink() or not env.is_file():
        raise ValueError("Ambiente deve ser arquivo existente, nao symlink")
    if stat.S_IMODE(env.stat().st_mode) & 0o007:
        raise ValueError("Ambiente com acesso world; corrija as permissoes antes")
    old_bytes = env.read_bytes()
    original = old_bytes.decode("utf-8")
    sanctum = read_sanctum_file(args.sanctum_file) if args.sanctum_file else None
    updated, state = prepare(original, sanctum, args.rotate_dashboard, args.rotate_sanctum)
    state["mode"] = "PLAN_ONLY"
    if args.apply and state["changed"]:
        backup = apply_atomic(env, updated, old_bytes)
        state["mode"] = "APPLIED"
        state["backup_file"] = backup.name  # Filename only, NEVER token or contents.
    elif args.apply:
        state["mode"] = "NO_CHANGE"
    print("CTWA CONFIG: " + " | ".join(f"{k}={v}" for k, v in state.items()))
    if state["sanctum"] == "MISSING_EXTERNAL_CRM_TOKEN":
        print("NO_GO: gerar Sanctum no CRM por usuario autorizado e passar via arquivo 0600.")
        return 2
    print("CONFIG_ONLY: ainda validar ability, permissao do usuario e runtime antes do deploy.")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, UnicodeError, ValueError) as err:
        # Exception descriptions contain no token; do not print env lines/values.
        print("NO_GO: configuracao invalida ou operacao de arquivo falhou (" + type(err).__name__ + ")", file=sys.stderr)
        sys.exit(2)
