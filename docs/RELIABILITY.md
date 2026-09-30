# Reliability and recovery

## Observation validity

Apply migration 6 before running this version. `observation_invalidations` has
one row per observation, a nonblank reason (at most 1000 characters), and the
database-recorded invalidation time. The original `price_observations` row is
never updated/deleted. Repeating an invalidation returns the first reason/time;
there is no undo, editing workflow, or automatic replacement/matching.

- Current Listing/Product prices and the public history API exclude invalidated
  observations. Historical summaries use that filtered history. If every
  observation is invalid, current price is missing and history is empty.
- Alert evaluation ignores invalid triggering observations and invalid earlier
  comparison observations. Evaluation and invalidation lock the same Listing
  row. If evaluation committed first, its event remains historical evidence;
  invalidating afterward does not retract it. No retroactive recalculation occurs.
- Promotion evidence and existing alert events are unchanged. New EffectivePrice
  requests use the currently selected valid observation; old facts remain auditable.
- Raw persistence audit reads retain all observations. Future collection can
  append valid observations normally, even with identical values. Invalidation
  does not blacklist a source or a value.

APIs use the existing data/error envelopes:

- `GET /listings/{id}/observations`: newest 100 audit records (all validity states),
  ordered by full observation timestamp then bytewise result ID, with `truncated`.
- `GET /observations/{id}`: inspect any exact observation ID, including older
  records outside the bounded list.
- `POST /observations/{id}/invalidate` with `{"reason":"wrong units reported"}`:
  returns 200 and the persisted audit record, including on a duplicate call.
  Invalid input returns 400; missing observation returns 404; failures return 500
  without internal error details.

The Product detail audit panel shows original facts and validity/reason/time,
and permits manual invalidation. Successful invalidation reloads the page so
current prices and previously loaded history cannot retain stale client results.
There is no auth model; these remain local-development APIs.

## Operational diagnosis

`GET /listings/{id}/price` and Product price responses expose the latest **valid
observation time** separately from process-local collection attempt/success times,
state, safe error code, and consecutive failure count. The count increments on
failed attempts and resets on success or backend restart; it is not a rolling
time-window total. Missing provider configuration remains `inactive`.

Backend logs use consistent event names and structured fields:
`collection_start`, `collection_result`, `ingestion_failure` (provider/validation/
persistence stage), `alert_evaluation_failure`, and `observation_invalidation`.
Only identities, states/codes, counters, and timestamps are logged. Raw provider
errors/payloads, credentials/headers, and free-text invalidation reasons are not.
Alert evaluation failure still leaves the observation committed; there is no
automatic evaluation retry service.

Diagnosis: check `/healthz` for process liveness, `/readyz` for DB connectivity,
then the Listing price response. Compare collection time with source observation
time rather than assuming a successful collection means a fresh source value.
Inspect safe log fields and the audit panel for a failed/missing price. Verify
the Listing has a startup collector configuration before using Refresh price.

## Query/index review

Current Listing/Product queries select one coherent observation with the existing
`price_observations_history_idx` on Listing, timestamp, nanosecond remainder, and
bytewise result ID. History and audit use the same ordering and explicit bounds.
Invalidation filtering uses `NOT EXISTS` with the new invalidation primary key;
an extra invalidation index would duplicate that key. Migration 6 adds
`listings_product_id_idx (product_id, id COLLATE "C")` for Product-to-Listing reads
and current-price ordering; the catalog previously had only primary/source-identity
indexes. Existing alert Listing/event indexes support their query shapes.
Drop comparison uses prior Listing/time order; historical-low comparison scans
earlier comparable observations to find a minimum. No measured bottleneck yet
justifies an additional price index or derived aggregate. Collection diagnostics
are an in-memory keyed lookup. This is a query-shape review, not a scale benchmark.
No caching, partitioning, retention deletion, time-series DB, or external search/
monitoring system is introduced. If volumes grow, measure representative plans
before changing this design.

## PostgreSQL backup and separate restore (PowerShell)

Run from the repository root with Docker Compose, `.env`, and Goose as documented
in README. Use the matching PostgreSQL tools bundled in the existing container.
The custom-format archive is PostgreSQL's standard format, not an application
format. It contains schema/data, indexes/constraints, and Goose metadata. Roles,
external configuration, logs, and process-local collector diagnostics are not
database data; preserve required configuration separately and never commit secrets.

Create a uniquely named backup of the configured database. Keep binary archives
out of PowerShell text pipelines; use container files and `docker compose cp`:

```powershell
docker compose up -d --wait postgres
New-Item -ItemType Directory -Force backups | Out-Null
$backupName = 'price_terminal_' + [guid]::NewGuid().ToString('N') + '.dump'
docker compose exec -T postgres sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc -f "$1"' sh "/tmp/$backupName"
if ($LASTEXITCODE -ne 0) { throw 'Backup failed' }
docker compose cp "postgres:/tmp/$backupName" "backups/$backupName"
if ($LASTEXITCODE -ne 0) { throw 'Backup copy failed' }
```

Restore only into a **new** disposable database. There is no `--clean` or restore
over the development database. Creation fails if the selected name already exists:

```powershell
$restoreDb = 'price_terminal_restore_' + [guid]::NewGuid().ToString('N')
docker compose exec -T postgres sh -c 'createdb -U "$POSTGRES_USER" --template template0 "$1"' sh $restoreDb
if ($LASTEXITCODE -ne 0) { throw 'Create disposable database failed' }
docker compose exec -T postgres sh -c 'pg_restore --exit-on-error --no-owner --no-acl -U "$POSTGRES_USER" -d "$1" "$2"' sh $restoreDb "/tmp/$backupName"
if ($LASTEXITCODE -ne 0) { throw 'Restore failed' }
goose -env .env -dir backend/migrations postgres "dbname=$restoreDb connect_timeout=5" status
goose -env .env -dir backend/migrations postgres "dbname=$restoreDb connect_timeout=5" up
docker compose exec -T postgres sh -c 'psql -U "$POSTGRES_USER" -d "$1" -c "TABLE goose_db_version;"' sh $restoreDb
```

For an existing host archive, copy it into the container first with
`docker compose cp "backups/$backupName" "postgres:/tmp/$backupName"`.
Use a compatible role with database-creation permission. `--no-owner --no-acl`
restores ownership to the local restoring role; this project has no app-user roles.
Keep the host backup; remove only the temporary container archive and the generated
disposable restore database after verification:

```powershell
docker compose exec -T postgres sh -c 'dropdb -U "$POSTGRES_USER" "$1"' sh $restoreDb
docker compose exec -T postgres rm -f "/tmp/$backupName"
```

### Repeatable seeded verification

After loading the PostgreSQL environment as in README, run from `backend/`:

```powershell
$env:BACKUP_VERIFY = '1'
go test -tags=integration -count=1 ./...
Remove-Item Env:BACKUP_VERIFY
```

`TestDockerBackupRestore` creates a fresh migrated source database, seeds catalog,
observations, promotions, alerts/events and invalidation state, then dumps it and
restores into a different empty database. It compares every column of every row
in all eight application tables plus Goose history, verifies the valid current
price, and runs migrate-up on the restored metadata. Test cleanup removes both
databases and the temporary archive even after failures. The normal development
database is used only for administrative database creation, never overwritten.
Collection attempt/success/error counters reset on restart and are deliberately
not claimed to survive pg_dump. No persistent collection table exists.
