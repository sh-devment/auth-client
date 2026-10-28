#!/usr/bin/env bash

set -euo pipefail

# Ночной бэкап SQLite: /opt/$APP/$APP.db → /backup/$APP, храним 7 последних.
# Ставится deploy.sh в /opt/$APP/backup.sh и запускается из crontab:
#   0 1 * * * flock /var/lock/backups.lock /opt/$APP/backup.sh $APP >> /backup/$APP/backup.log 2>&1
# Время у всех приложений одно — общий lock выстраивает их в очередь.

APP="${1:?usage: backup.sh <app>}"

[[ "$APP" =~ ^[a-z0-9][a-z0-9_-]*$ ]] || {
  echo "Invalid APP name: '$APP'" >&2
  exit 1
}

DB="/opt/$APP/$APP.db"
BACKUP_DIR="/backup/$APP"
KEEP=7

DATE=$(date +%F_%H-%M-%S)
BACKUP_FILE="$BACKUP_DIR/$APP-$DATE.db"

# Без проверки sqlite3 молча создал бы пустую базу на месте $DB.
if [ ! -f "$DB" ]; then
  echo "$DATE no database at $DB, skipping"
  exit 0
fi

mkdir -p "$BACKUP_DIR"

# Пишем во временный файл: недописанный бэкап не должен попасть в ротацию.
sqlite3 "$DB" ".backup '$BACKUP_FILE.tmp'"

# убедиться что файл создан и не пустой
[ -s "$BACKUP_FILE.tmp" ]
mv "$BACKUP_FILE.tmp" "$BACKUP_FILE"

# оставить только последние $KEEP успешных бэкапов
find "$BACKUP_DIR" -maxdepth 1 -name "$APP-*.db" -type f |
  sort -r |
  tail -n +$((KEEP + 1)) |
  xargs -r rm -f

echo "$DATE backup ok: $BACKUP_FILE"
