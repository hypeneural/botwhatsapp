#!/usr/bin/env bash
# Single fast P0 contract gate. Offline by default: NO SSH, NO POST, NO deploy.
# Optional: --env /path/private/.env.production --spool /actual/volume/path
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"
if [[ "$#" -ne 0 && ( "$#" -ne 4 || "$1" != "--env" || "$3" != "--spool" ) ]]; then
  echo "USO: bash scripts/validate-ctwa-p0-credentials.sh [--env ENV_FILE --spool SPOOL_DIR]" >&2
  exit 2
fi
echo "CTWA P0 | Security gate (sem envio externo ou alteracao de ambiente)"
bash -n scripts/preflight-ctwa-production-readonly.sh
bash -n scripts/preflight-ctwa-production-readonly.test.sh
bash scripts/preflight-ctwa-production-readonly.test.sh
if command -v python3 >/dev/null 2>&1; then
  python_cmd=(python3)
elif command -v py >/dev/null 2>&1; then
  python_cmd=(py -3)
elif command -v python >/dev/null 2>&1; then
  python_cmd=(python)
else
  echo "NO_GO: Python ausente para validar o provisionador de credenciais" >&2
  exit 2
fi
PYTHONDONTWRITEBYTECODE=1 "${python_cmd[@]}" scripts/configure-ctwa-secrets.test.py -v
PYTHONDONTWRITEBYTECODE=1 "${python_cmd[@]}" scripts/audit-ctwa-panel-readonly.test.py -v
if command -v php >/dev/null 2>&1; then
  php -l scripts/issue-ctwa-sanctum-on-crm.php
  php -l scripts/verify-ctwa-sanctum-on-crm.php
else
  echo "SKIP: PHP lint so pode ser comprovado no host CRM autorizado"
fi
if [[ "$#" -eq 4 ]]; then
  bash scripts/preflight-ctwa-production-readonly.sh "$2" "$4"
else
  echo "PENDENTE: preflight do .env.production real e volume confirmado do container"
fi
echo "PASS_OFFLINE_SECURITY: nao significa token Sanctum autenticado, producao ou canario."
