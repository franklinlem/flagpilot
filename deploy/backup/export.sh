#!/bin/bash
# Executado pela chave SSH restrita do destino. stdout contém somente ciphertext.
set -euo pipefail
[[ "${SSH_ORIGINAL_COMMAND:-backup}" == backup ]] || exit 64
ROOT="$HOME/flagpilot/homolog"
source "$ROOT/private/backup.env"
exec 9>"$ROOT/private/backup.lock"
flock -n 9 || exit 75
container=$(docker ps --filter "label=com.docker.compose.project=$FLAGPILOT_SERVICE_UUID" --filter label=com.docker.compose.service=db --format '{{.ID}}')
[[ "$container" =~ ^[a-f0-9]+$ ]] || { echo 'FlagPilot DB indisponivel ou ambiguo' >&2; exit 1; }
umask 077
file=$(mktemp "$ROOT/private/backup.XXXXXXXX.gpg")
trap 'rm -f "$file"' EXIT
# Sem dump em disco, senha em argv ou acesso SSH a outros comandos.
docker exec "$container" pg_dump -U flagpilot_owner -d flagpilot --format=custom |
  gpg --homedir "$ROOT/private/gnupg" --batch --trust-model always --recipient "$FLAGPILOT_GPG_RECIPIENT" --encrypt --output "$file" --yes
[[ -s "$file" ]] || exit 1
cat "$file"
