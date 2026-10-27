#!/usr/bin/env bash
# CTWA 5287 preflight: READ ONLY. Does not contact CRM/Evolution or restart anything.
# Usage: bash scripts/preflight-ctwa-production-readonly.sh /secure/path/.env.production /existing/spool
set -euo pipefail
umask 077
if [[ "$#" -ne 2 ]]; then
  echo "USO: bash scripts/preflight-ctwa-production-readonly.sh ENV_FILE SPOOL_DIR" >&2
  exit 2
fi
env_file="$1"
spool_dir="$2"
failures=0
pass() { printf 'PASS: %s\n' "$1"; }
fail() { printf 'FAIL: %s\n' "$1" >&2; failures=$((failures+1)); }
warn() { printf 'REVISAR: %s\n' "$1"; }
if [[ ! -f "$env_file" || ! -r "$env_file" ]]; then
  echo "FAIL: arquivo de configuracao privado inexistente ou ilegivel" >&2
  exit 2
fi
# The script never sources/evals an env file or prints any variable values.
# Last literal KEY= wins. Quotes on the edges are stripped for presence checks.
value() {
  local key="$1" line
  line="$(grep -E "^([[:space:]]*)${key}=" "$env_file" | tail -n 1 || true)"
  line="${line#*=}"
  line="${line%$'\r'}"
  if [[ "${line:0:1}" == '"' && "${line: -1}" == '"' ]]; then
    line="${line:1:${#line}-2}"
  fi
  if [[ "${line:0:1}" == "'" && "${line: -1}" == "'" ]]; then
    line="${line:1:${#line}-2}"
  fi
  printf '%s' "$line"
}
check_true() {
  local key="$1" v
  v="$(value "$key")"
  if [[ "${v,,}" == "true" ]]; then pass "$key ativo"; else fail "$key nao esta true"; fi
}
check_secret() {
  local key="$1" min="$2" v
  v="$(value "$key")"
  if (( ${#v} >= min )); then pass "$key definido (valor oculto)"; else fail "$key ausente ou curto (valor oculto)"; fi
}
mode="$(stat -c '%a' "$env_file")"
# A service env file may be 600 or 640; prohibit any world access.
if (( (8#$mode & 7) == 0 )); then pass "arquivo de configuracao nao legivel pelo publico"; else fail "arquivo de configuracao possui permissao para outros"; fi
if [[ "$(value BOT_ADS_PHONE)" == "5548996425287" ]]; then pass "WhatsApp de anuncios final 5287"; else fail "BOT_ADS_PHONE incorreto"; fi
if [[ -z "$(value BOT_GREETING_PHONE)" && -z "$(value BOT_GREETING_INSTANCE_UUID)" ]]; then
  pass "sem segundo papel Greeting no 5287"
else
  fail "para esta implantacao CTWA 5287, BOT_GREETING_PHONE e BOT_GREETING_INSTANCE_UUID devem estar vazios"
fi
check_true BOT_ENABLED
check_true BOT_OUTBOUND_ENABLED
check_true BOT_OPERATIONAL_AUTHORIZATION
check_secret BOT_ADS_INSTANCE_UUID 8
check_secret BOT_ADS_WEBHOOK_SECRET 32
check_secret EVOLUTION_ADS_APIKEY 8
check_secret BOT_DASHBOARD_TOKEN 32
check_secret EVIDENCIA_CRM_PROPOSAL_SANCTUM_TOKEN 32
if [[ "$(value EVIDENCIA_CRM_BASE_URL)" =~ ^https://[^[:space:]]+$ ]]; then pass "CRM acessado via HTTPS"; else fail "EVIDENCIA_CRM_BASE_URL precisa ser HTTPS"; fi
if [[ "$(value EVOLUTION_GO_BASE_URL)" =~ ^https?://[^[:space:]]+$ ]]; then pass "URL Evolution definida"; else fail "EVOLUTION_GO_BASE_URL ausente"; fi
if [[ -n "$(value BOT_ADS_AUDIO_FILE)" ]]; then pass "caminho do audio configurado (verificar montagem no container)"; else fail "BOT_ADS_AUDIO_FILE nao configurado"; fi
if [[ -n "$(value BOT_ADS_DOCUMENT_FILE)" || -n "$(value BOT_ADS_DOCUMENT_URL)" ]]; then pass "fonte do PDF configurada (verificar acesso real no container)"; else fail "PDF nao configurado"; fi
if [[ -d "$spool_dir" && -r "$spool_dir" && -w "$spool_dir" ]]; then
  pass "journal existente, legivel e gravavel pelo operador (verificar UID do container)"
else
  fail "journal persistente nao encontrado ou sem acesso (NUNCA criar volume vazio automaticamente)"
fi
if (( failures>0 )); then
  printf '\nNO_GO: %d bloqueio(s). Nenhum segredo exposto, sem qualquer escrita.\n' "$failures" >&2
  exit 1
fi
printf '\nPASS_CONFIG: somente arquivo apontado e permissoes.\n'
printf 'PENDENTE: confrontar environment/volume REAL do container, health, perfis, canal WhatsApp, recibo CRM e canario unico.\n'
printf 'NAO execute docker compose down, down -v, up com volume novo ou replay em massa.\n'
