#!/usr/bin/env bash
# Offline, self-contained preflight contract test with intentionally fake secrets.
set -euo pipefail
umask 077
root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/spool"
cat > "$tmp/settings.env" <<'EOF'
BOT_ENABLED=true
BOT_OUTBOUND_ENABLED=true
BOT_OPERATIONAL_AUTHORIZATION=true
BOT_ADS_PHONE=5548996425287
BOT_GREETING_PHONE=
BOT_GREETING_INSTANCE_UUID=
BOT_ADS_INSTANCE_UUID=demo-ads-instance
BOT_ADS_WEBHOOK_SECRET=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
EVOLUTION_ADS_APIKEY=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
BOT_DASHBOARD_TOKEN=cccccccccccccccccccccccccccccccccccccccc
EVIDENCIA_CRM_PROPOSAL_SANCTUM_TOKEN=17|dddddddddddddddddddddddddddddddddddddddd
EVIDENCIA_CRM_BASE_URL=https://evydencia.com
EVOLUTION_GO_BASE_URL=http://127.0.0.1:12345
BOT_ADS_AUDIO_FILE=/app/assets/example.ogg
BOT_ADS_DOCUMENT_URL=http://127.0.0.1:8788/static/example.pdf
EOF
chmod 600 "$tmp/settings.env"
bash -n "$root/scripts/preflight-ctwa-production-readonly.sh"
bash "$root/scripts/preflight-ctwa-production-readonly.sh" "$tmp/settings.env" "$tmp/spool" > "$tmp/pass.log"
grep -q 'PASS_CONFIG' "$tmp/pass.log"
if grep -Eq 'aaaaaaaaaaaaaaaa|bbbbbbbbbbbbbbbb|cccccccccccccccc|dddddddddddddddd' "$tmp/pass.log"; then
  echo "FAIL: preflight leaked fake secret" >&2; exit 1
fi
# A second Greeting role is not part of the approved 5287 topology.
sed -i 's/^BOT_GREETING_PHONE=$/BOT_GREETING_PHONE=5548999999999/' "$tmp/settings.env"
if bash "$root/scripts/preflight-ctwa-production-readonly.sh" "$tmp/settings.env" "$tmp/spool" > "$tmp/fail.log" 2>&1; then
  echo "FAIL: unexpected separate role was accepted" >&2; exit 1
fi
sed -i 's/^BOT_GREETING_PHONE=.*/BOT_GREETING_PHONE=/' "$tmp/settings.env"
sed -i 's/^BOT_ADS_PHONE=.*/BOT_ADS_PHONE=554832200440/' "$tmp/settings.env"
if bash "$root/scripts/preflight-ctwa-production-readonly.sh" "$tmp/settings.env" "$tmp/spool" > "$tmp/fail.log" 2>&1; then
  echo "FAIL: legacy WhatsApp 0440 was accepted" >&2; exit 1
fi
sed -i 's/^BOT_ADS_PHONE=.*/BOT_ADS_PHONE=5548996425287/' "$tmp/settings.env"
# Wrong origin must fail, including HTTPS hosts where credentials could leak.
sed -i 's#^EVIDENCIA_CRM_BASE_URL=.*#EVIDENCIA_CRM_BASE_URL=https://example.invalid#' "$tmp/settings.env"
if bash "$root/scripts/preflight-ctwa-production-readonly.sh" "$tmp/settings.env" "$tmp/spool" > "$tmp/fail.log" 2>&1; then
  echo "FAIL: non-official CRM origin was accepted" >&2; exit 1
fi
sed -i 's#^EVIDENCIA_CRM_BASE_URL=.*#EVIDENCIA_CRM_BASE_URL=https://evydencia.com#' "$tmp/settings.env"
# A random string (even if long) cannot substitute for a CRM-issued Sanctum token.
sed -i 's/^EVIDENCIA_CRM_PROPOSAL_SANCTUM_TOKEN=.*/EVIDENCIA_CRM_PROPOSAL_SANCTUM_TOKEN=eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee/' "$tmp/settings.env"
if bash "$root/scripts/preflight-ctwa-production-readonly.sh" "$tmp/settings.env" "$tmp/spool" > "$tmp/fail.log" 2>&1; then
  echo "FAIL: fake Sanctum token format was accepted" >&2; exit 1
fi
sed -i 's/^EVIDENCIA_CRM_PROPOSAL_SANCTUM_TOKEN=.*/EVIDENCIA_CRM_PROPOSAL_SANCTUM_TOKEN=/' "$tmp/settings.env"
if bash "$root/scripts/preflight-ctwa-production-readonly.sh" "$tmp/settings.env" "$tmp/spool" > "$tmp/fail.log" 2>&1; then
  echo "FAIL: missing Sanctum credential was accepted" >&2; exit 1
fi
echo 'PASS: offline CTWA deployment preflight contract (no external calls)'
