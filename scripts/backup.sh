#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
umask 077
mkdir -p .deploy/backups
backup_file=".deploy/backups/ai-buddy-$(date -u +%Y%m%dT%H%M%SZ).sql.gz"
tmp_file=$(mktemp .deploy/backups/.dump.XXXXXX)
trap 'rm -f "$tmp_file" "$tmp_file.gz"' EXIT HUP INT TERM
docker compose exec -T db pg_dump -U ai_buddy ai_buddy </dev/null > "$tmp_file"
gzip -c "$tmp_file" > "$tmp_file.gz"
gzip -t "$tmp_file.gz"
mv "$tmp_file.gz" "$backup_file"
find .deploy/backups -name '*.sql.gz' -mtime +14 -delete
printf 'Backup saved: %s\n' "$backup_file"
