#!/usr/bin/env python3
"""Coordinated rotation of CTWA Webhook Bearer Secrets for BotWhatsApp & Relay.

FAIL-CLOSED, OPERATOR-ONLY (POSIX 0600).
Rotates:
  1. BOT_ADS_WEBHOOK_SECRET & RELAY_BOT_URL (Relay -> Bot hop)
  2. RELAY_INGRESS_SECRET & Evolution Webhook (Evolution -> Relay hop)

NEVER prints raw secrets. Prints SHA256 fingerprints only.
Requires operator confirmation or --execute flag.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import secrets
import shutil
import stat
import subprocess
import sys
import time
import urllib.request

E164_BR = re.compile(r"^55[0-9]{10,11}$")


def check_private_file(path: Path) -> None:
    st = path.stat()
    if not stat.S_ISREG(st.st_mode):
        raise ValueError(f"{path} is not a regular file")
    if sys.platform != "win32" and (st.st_mode & 0o077):
        raise ValueError(f"{path} has insecure permissions (must be 0600)")


def fp(value: str) -> str:
    """Return 16-char SHA256 fingerprint of sensitive string."""
    return hashlib.sha256(value.encode("utf-8")).hexdigest()[:16]


def main(argv=None):
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--env-file", default="/srv/platform/stacks/botwhatsapp/.env.production")
    p.add_argument("--compose-override", default="/srv/platform/stacks/botwhatsapp/deploy/compose.override.yaml")
    p.add_argument("--provider-url", default="http://127.0.0.1:14000")
    p.add_argument("--execute", action="store_true", help="Apply rotation to files, containers and provider")
    args = p.parse_args(argv)

    env_path = Path(args.env_file)
    compose_path = Path(args.compose_override)

    if not env_path.exists() or not compose_path.exists():
        print(json.dumps({"status": "BLOCKED", "reason": "config_files_not_found"}))
        return 2

    try:
        check_private_file(env_path)
    except Exception as exc:
        print(json.dumps({"status": "BLOCKED", "reason": str(exc)}))
        return 2

    # Generate new cryptographically secure secrets
    new_relay_ingress_secret = secrets.token_hex(32)
    new_bot_ads_secret = secrets.token_hex(32)

    plan = {
        "status": "PLAN_READY",
        "new_relay_secret_sha": fp(new_relay_ingress_secret),
        "new_bot_secret_sha": fp(new_bot_ads_secret),
        "execute": bool(args.execute),
    }

    if not args.execute:
        print(json.dumps(plan, indent=2))
        return 0

    timestamp = time.strftime("%Y%m%d_%H%M%S")
    backup_env = env_path.with_name(f"{env_path.name}.bak_{timestamp}")
    backup_compose = compose_path.with_name(f"{compose_path.name}.bak_{timestamp}")

    shutil.copy2(env_path, backup_env)
    shutil.copy2(compose_path, backup_compose)
    if sys.platform != "win32":
        backup_env.chmod(0o600)
        backup_compose.chmod(0o600)

    # 1. Update .env.production
    env_lines = env_path.read_text(encoding="utf-8").splitlines()
    updated_env = []
    for line in env_lines:
        if line.startswith("BOT_ADS_WEBHOOK_SECRET="):
            updated_env.append(f"BOT_ADS_WEBHOOK_SECRET={new_bot_ads_secret}")
        elif line.startswith("RELAY_INGRESS_SECRET="):
            updated_env.append(f"RELAY_INGRESS_SECRET={new_relay_ingress_secret}")
        elif line.startswith("RELAY_GREETING_INGRESS_SECRET="):
            updated_env.append(f"RELAY_GREETING_INGRESS_SECRET={new_relay_ingress_secret}")
        elif line.startswith("RELAY_BOT_URL=") or line.startswith("RELAY_GREETING_BOT_URL="):
            prefix = line.split("=")[0]
            updated_env.append(f"{prefix}=http://bot:8788/webhook/ads/{new_bot_ads_secret}")
        else:
            updated_env.append(line)
    env_path.write_text("\n".join(updated_env) + "\n", encoding="utf-8")

    # 2. Update compose.override.yaml
    compose_text = compose_path.read_text(encoding="utf-8")
    compose_text = re.sub(
        r"RELAY_INGRESS_SECRET:\s+[a-f0-9]{32,}",
        f"RELAY_INGRESS_SECRET: {new_relay_ingress_secret}",
        compose_text,
    )
    compose_text = re.sub(
        r"RELAY_BOT_URL:\s+http://bot:8788/webhook/ads/[a-f0-9]{32,}",
        f"RELAY_BOT_URL: http://bot:8788/webhook/ads/{new_bot_ads_secret}",
        compose_text,
    )
    compose_path.write_text(compose_text, encoding="utf-8")

    # 3. Recreate bot and relay containers
    deploy_dir = compose_path.parent
    cmd_recreate = [
        "docker", "compose",
        "-f", "compose.staging.yaml", "-f", "compose.override.yaml",
        "up", "-d", "--no-deps", "bot", "relay-greeting"
    ]
    res_compose = subprocess.run(cmd_recreate, cwd=str(deploy_dir), capture_output=True, text=True)
    if res_compose.returncode != 0:
        # Rollback files on container failure
        shutil.copy2(backup_env, env_path)
        shutil.copy2(backup_compose, compose_path)
        print(json.dumps({"status": "FAILED_CONTAINER_RESTART", "error": res_compose.stderr[:300]}))
        return 2

    # 4. Fetch instance token from PostgreSQL to update Evolution GO
    res_token = subprocess.run([
        "docker", "exec", "evolution-go-postgres-1", "psql", "-U", "postgres", "-d", "evogo_users",
        "-t", "-A", "-c", "SELECT token FROM instances WHERE name='evydencia';"
    ], capture_output=True, text=True)
    inst_token = res_token.stdout.strip()
    if not inst_token:
        print(json.dumps({"status": "FAILED_GETTING_INSTANCE_TOKEN"}))
        return 2

    # 5. Update webhook on Evolution GO via /instance/connect
    new_webhook_url = f"http://botwhatsapp-stage-relay-greeting-1:8790/webhook/evolution/{new_relay_ingress_secret}"
    events = [
        "MESSAGE", "SEND_MESSAGE", "READ_RECEIPT", "PRESENCE",
        "HISTORY_SYNC", "CHAT_PRESENCE", "CALL", "CONNECTION",
        "LABEL", "CONTACT", "GROUP", "NEWSLETTER", "QRCODE",
        "BUTTON_CLICK", "PICTURE", "USER_ABOUT"
    ]
    payload = {
        "phone": "554896425287",
        "webhookUrl": new_webhook_url,
        "subscribe": events
    }
    req = urllib.request.Request(
        f"{args.provider_url.rstrip('/')}/instance/connect",
        data=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json", "apikey": inst_token}
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            if data.get("message") != "success":
                raise ValueError("provider_rejected_webhook_update")
    except Exception as exc:
        print(json.dumps({"status": "FAILED_PROVIDER_UPDATE", "error": str(exc)}))
        return 2

    plan["status"] = "ROTATION_COMPLETED"
    plan["backups"] = [str(backup_env), str(backup_compose)]
    print(json.dumps(plan, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
