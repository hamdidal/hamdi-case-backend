#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKUP_DIR="$SCRIPT_DIR/../backups"
CONTAINER="dpp-postgres"
DB_USER="dppuser"
DB_NAME="dppdb"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
BACKUP_FILE="$BACKUP_DIR/${DB_NAME}_${TIMESTAMP}.sql.gz"

mkdir -p "$BACKUP_DIR"

docker exec "$CONTAINER" pg_dump -U "$DB_USER" "$DB_NAME" | gzip > "$BACKUP_FILE"

echo "Backup saved: $BACKUP_FILE"

# Retain only the 7 most recent backups
find "$BACKUP_DIR" -name "${DB_NAME}_*.sql.gz" -type f | sort | head -n -7 | xargs -r rm --
