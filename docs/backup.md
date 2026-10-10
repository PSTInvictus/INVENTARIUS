# Backups

All data is in one SQLite file (`/data/inventarius.db` in the `data` volume) plus the item
photos in `/data/photos`.

## Database: Litestream

The `litestream` service streams every change to the backup location within seconds, and
keeps 30 days of history (`deploy/litestream.yml`). By default the backup goes to
`BACKUP_DIR`, which should be **another disk** (e.g. a USB drive mounted on the PC). For an
off-site copy, switch the replica to S3-compatible storage such as Cloudflare R2; the free tier
is enough (see the comment in `litestream.yml`).

## Photos

Litestream only covers the database. Copy photos nightly with cron on the PC:

```sh
# crontab -e
0 3 * * * docker run --rm -v deploy_data:/data:ro -v /mnt/backup:/backup alpine \
  sh -c 'mkdir -p /backup/photos && cp -ru /data/photos/. /backup/photos/'
```

## Restore

```sh
cd deploy
docker compose stop inventarius litestream
docker compose run --rm --no-deps --entrypoint litestream litestream \
  restore -config /etc/litestream.yml -o /data/restored.db /data/inventarius.db
docker compose run --rm --no-deps --entrypoint sh litestream -c \
  'cd /data && mv inventarius.db inventarius.db.broken && rm -rf inventarius.db-wal inventarius.db-shm .inventarius.db-litestream && mv restored.db inventarius.db && chown 65532:65532 inventarius.db'
docker compose up -d
```

Add `-timestamp 2026-10-01T12:00:00Z` to `restore` to go back to a point in time.

This procedure was tested on 2026-10-10: the restored database matched the live one,
including a change made while replication was running. Repeat the drill now and then.
