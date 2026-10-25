#!/bin/bash
set -euo pipefail
ROOT="$HOME/backups/flagpilot/homolog"
exec 9>"$ROOT/pull.lock"
flock -n 9 || exit 75
umask 077
file=$(mktemp "$ROOT/.incoming.XXXXXXXX")
trap 'rm -f "$file"' EXIT
ssh -T -o BatchMode=yes -o ConnectTimeout=20 -o ServerAliveInterval=15 -o ServerAliveCountMax=3 \
  -o StrictHostKeyChecking=yes -o UserKnownHostsFile="$ROOT/known_hosts" \
  -o IdentitiesOnly=yes -i "$ROOT/pull_ed25519" franklin@158.220.120.2 backup > "$file"
[[ -s "$file" ]] || exit 1
name="flagpilot-homolog-$(date -u +%Y%m%dT%H%M%SZ).dump.gpg"
# ln falha se o nome já existe; não sobrescrever backup anterior.
ln "$file" "$ROOT/$name"
sha256sum "$ROOT/$name" > "$ROOT/$name.sha256"
date -u +%FT%TZ > "$ROOT/last-success.txt"
# Retenção restrita aos arquivos gerados pelo próprio FlagPilot.
find "$ROOT" -maxdepth 1 -type f -name 'flagpilot-homolog-*.dump.gpg*' -mtime +14 -delete
printf '%s backup armazenado: %s\n' "$(date -u +%FT%TZ)" "$name"
