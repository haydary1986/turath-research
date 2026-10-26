#!/bin/bash
# Nightly backup of the Turath Research stack: PostgreSQL dump + uploaded files.
# Usage: ./scripts/backup.sh [target-dir]   (default ./backups) — keeps the last 14.
set -euo pipefail
cd "$(dirname "$0")/.."
TARGET="${1:-./backups}"
STAMP="$(date +%Y%m%d-%H%M%S)"
mkdir -p "$TARGET"

docker compose exec -T postgres pg_dump -U turath -d turath --no-owner \
  | gzip > "$TARGET/db-$STAMP.sql.gz"

docker run --rm \
  -v "$(docker compose ps -q backend | xargs docker inspect -f '{{ range .Mounts }}{{ if eq .Destination "/data/uploads" }}{{ .Name }}{{ end }}{{ end }}')":/uploads:ro \
  -v "$(cd "$TARGET" && pwd)":/backup alpine \
  tar czf "/backup/uploads-$STAMP.tar.gz" -C /uploads .

ls -1t "$TARGET"/db-*.sql.gz 2>/dev/null | tail -n +15 | xargs rm -f --
ls -1t "$TARGET"/uploads-*.tar.gz 2>/dev/null | tail -n +15 | xargs rm -f --
echo "backup complete: $TARGET/db-$STAMP.sql.gz + uploads-$STAMP.tar.gz"
