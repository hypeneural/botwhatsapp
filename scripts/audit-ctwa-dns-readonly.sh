#!/usr/bin/env bash
# Post-DNS validation for bot.etijucas.com.br. GET/HEAD only; no credentials.
# Expected IPv4 MUST be verified against the VPS external interface first.
set -euo pipefail
HOST="bot.etijucas.com.br"
expected="${1:-}"
if [[ -n "$expected" && ! "$expected" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]]; then
  echo "NO_GO: expected IPV4 invalido" >&2; exit 2
fi
if ! command -v getent >/dev/null 2>&1 || ! command -v curl >/dev/null 2>&1; then
  echo "NO_GO: requer getent e curl com TLS" >&2; exit 2
fi
ips="$(getent ahostsv4 "$HOST" | awk '{print $1}' | sort -u || true)"
if [[ -z "$ips" ]]; then
  echo "NO_GO: DNS A nao propagado ou nao visivel neste resolvedor" >&2; exit 2
fi
echo "DNS bot.etijucas.com.br -> $(echo "$ips" | paste -sd , -)"
if [[ -n "$expected" ]] && ! grep -Fxq "$expected" <<< "$ips"; then
  echo "NO_GO: host nao resolve para IP publico esperado" >&2; exit 2
fi
echo "INFO: compare IPs obtidos com WAN da VPS e com a zona DNS autoritativa"
for path in /dashboard /painel '/api/dashboard/events?limit=1' /api/dashboard/contacts; do
  code="$(curl --noproxy '*' --silent --show-error --connect-timeout 5 --max-time 12 \
    --output /dev/null --write-out '%{http_code}' --head "https://$HOST$path")" || {
    echo "NO_GO: TLS/rede do endereco HTTPS nao disponivel em $path" >&2; exit 2
  }
  if [[ "$code" != 401 ]]; then
    echo "NO_GO: $path deveria retornar 401 anonimo, mas retornou $code" >&2
    exit 2
  fi
  echo "PASS: HTTPS $path bloqueia anonimo (401)"
done
for path in /static/ /static/Proposta_Natal.pdf; do
  code="$(curl --noproxy '*' --silent --show-error --connect-timeout 5 --max-time 12 \
    --output /dev/null --write-out '%{http_code}' --head "https://$HOST$path")" || {
    echo "NO_GO: TLS/rede indisponivel" >&2; exit 2
  }
  if [[ "$code" != 404 ]]; then
    echo "NO_GO: $path deveria estar oculto ao publico, mas retornou $code" >&2; exit 2
  fi
  echo "PASS: $path nao foi exposto publicamente"
done
echo "PASS_PUBLIC_EDGE_ONLY: ainda executar smoke autenticado e confirmar TLS/Caddy/canario"
