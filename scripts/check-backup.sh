#!/bin/sh
# Restores a freshly created dump into a new temporary database, never production.
set -eu
cd "$(dirname "$0")/.."
./scripts/backup.sh </dev/null
AI_BUDDY_RESTORE_DB="buddy_restore_check_$$"
AI_BUDDY_BACKUP_FILE=$(find .deploy/backups -name '*.sql.gz' -type f | sort | tail -1)
AI_BUDDY_RESTORE_SQL=$(mktemp)
trap 'rm -f "$AI_BUDDY_RESTORE_SQL"' EXIT HUP INT TERM
gzip -dc "$AI_BUDDY_BACKUP_FILE" > "$AI_BUDDY_RESTORE_SQL"
docker compose exec -T db createdb -U ai_buddy "$AI_BUDDY_RESTORE_DB" </dev/null
trap 'docker compose exec -T db dropdb --if-exists -U ai_buddy "$AI_BUDDY_RESTORE_DB" </dev/null; rm -f "$AI_BUDDY_RESTORE_SQL"' EXIT HUP INT TERM
docker compose exec -T db psql -v ON_ERROR_STOP=1 -U ai_buddy -d "$AI_BUDDY_RESTORE_DB" < "$AI_BUDDY_RESTORE_SQL" > /tmp/ai-buddy-restore-check.log
AI_BUDDY_TABLES=$(docker compose exec -T db psql -U ai_buddy -d "$AI_BUDDY_RESTORE_DB" -tAc "SELECT count(*) FROM information_schema.tables WHERE table_schema='public'" </dev/null)
test "$AI_BUDDY_TABLES" -ge 10
printf 'Restore verified in isolated database: %s tables\n' "$AI_BUDDY_TABLES"
