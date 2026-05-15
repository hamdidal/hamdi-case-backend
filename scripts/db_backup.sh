#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/../.env"

# Load .env when running from cron (no shell environment)
if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck source=/dev/null
  source "$ENV_FILE"
  set +a
fi

BACKUP_DIR="$SCRIPT_DIR/../backups"
CONTAINER="${BACKUP_CONTAINER:-dpp-postgres}"
DB_USER="${DB_USER:?DB_USER is not set}"
DB_NAME="${DB_NAME:?DB_NAME is not set}"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
BACKUP_FILE="$BACKUP_DIR/${DB_NAME}_${TIMESTAMP}.sql.gz"
BACKUP_NAME="${DB_NAME}_${TIMESTAMP}.sql.gz"

mkdir -p "$BACKUP_DIR"

docker exec "$CONTAINER" pg_dump -U "$DB_USER" "$DB_NAME" | gzip > "$BACKUP_FILE"

echo "Backup saved: $BACKUP_FILE"

# Retain only the 7 most recent backups
find "$BACKUP_DIR" -name "${DB_NAME}_*.sql.gz" -type f | sort | head -n -7 | xargs -r rm --

# Sync backup to cloud storage when Rclone is configured
if [[ -n "${RCLONE_REMOTE_NAME:-}" ]] && command -v rclone &>/dev/null; then
  rclone copy "$BACKUP_DIR/$BACKUP_NAME" "${RCLONE_REMOTE_NAME}:backups" --progress
  echo "Backup synced to remote: ${RCLONE_REMOTE_NAME}:backups"
fi
