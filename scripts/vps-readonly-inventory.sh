#!/usr/bin/env bash
# AUDITORIA SOMENTE LEITURA. Revisar localmente antes de executar por SSH autorizado.
set -u
export LC_ALL=C
section(){ printf '\n### %s\n' "$1"; }
section "Sistema"
uname -srmo 2>/dev/null || true
if command -v hostnamectl >/dev/null 2>&1; then hostnamectl 2>/dev/null | grep -E '^ *(Operating System|Kernel|Architecture|Virtualization):' || true; fi
section "Recursos"
uptime 2>/dev/null || true
free -h 2>/dev/null || true
df -h -x tmpfs -x devtmpfs 2>/dev/null | head -25 || true
section "Binarios"
for c in docker go python3 ffmpeg; do if command -v "$c" >/dev/null 2>&1; then printf '%s: present\n' "$c"; else printf '%s: absent\n' "$c"; fi; done
if command -v docker >/dev/null 2>&1; then
  section "Docker version"
  docker version --format 'client={{.Client.Version}} server={{.Server.Version}}' 2>/dev/null || true
  docker compose version 2>/dev/null || true
  section "Containers: name | image | status | ports"
  docker ps --format '{{.Names}} | {{.Image}} | {{.Status}} | {{.Ports}}' 2>/dev/null || true
  section "Docker networks (names only)"
  docker network ls --format '{{.Name}} | {{.Driver}}' 2>/dev/null || true
fi
section "TCP listener port numbers ONLY"
if command -v ss >/dev/null 2>&1; then ss -lntH 2>/dev/null | awk '{print $4}' | grep -Eo '[0-9]+$' | sort -un | head -100 || true; fi
section "Active service NAMES ONLY"
if command -v systemctl >/dev/null 2>&1; then systemctl list-units --type=service --state=running --no-legend --no-pager 2>/dev/null | awk '{print $1}' | head -100 || true; fi
section "FIM: não enviar saída bruta; revisar e redigir antes de salvar"
