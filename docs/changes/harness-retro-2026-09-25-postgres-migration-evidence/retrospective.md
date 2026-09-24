# Harness retrospective: PostgreSQL migration evidence

## Incident

The note-comment API hardening manually verified `make migrate` against the
disposable PostgreSQL database, but the normal `make test-integration` target
only bootstrapped PostgreSQL with `docker-entrypoint-initdb.d` files. The
required integration gate therefore did not prove migration adoption or repeat
safety. The static migration check also did not detect a newly added migration
missing from either Compose bootstrap profile.

## Classification and root cause

- Classification: `TEST_EVIDENCE_GAP`
- Root cause: the integration runner exercised application persistence against a
  fresh schema, while the migration runner was an independent Make target with
  no required contract test or gate integration.

## Invariant

An applicable PostgreSQL migration claim passes only when a disposable database
proves both upgrade from an existing schema and a repeat migration run that
performs no additional changes.

## Harness changes

- Added `harness/scripts/tests/check-migration-runner.sh`. It removes only the
  disposable test database's migration metadata and newest columns, runs the
  real `make migrate` command, asserts schema and metadata state, then runs it
  again and requires five skips with zero applications.
- Added `harness/scripts/tests/psql-test-client.sh` so migration files are
  streamed into the disposable PostgreSQL container instead of passing a host
  path that the container cannot read.
- Updated `Makefile` so `make test-integration` runs that check before tagged Go
  tests, recreates the disposable database for each run, and always removes it
  with an exit trap.
- Added `make check-migration-runner` for the focused migration contract.
- Tightened `harness/scripts/check-migrations.sh` so every `.up.sql` migration
  must be explicitly mounted in both Compose bootstrap profiles. Its contract
  fixture proves that a missing mount is rejected.

## Verification

- `bash -n harness/scripts/tests/check-migration-runner.sh` — PASS
- `bash harness/scripts/check-migrations.sh` — PASS
- `make check-migration-runner` — PASS
- `make test-integration` — PASS, including the migration runner check and
  tagged PostgreSQL/API tests
- `make check` — PASS
- `git diff --check` — PASS

## Routed items outside harness scope

The production Compose stack still uses `docker-entrypoint-initdb.d`, which
only runs for a newly initialized persistent volume. Automatically applying
new migrations during deployment requires a deployment/startup decision and is
not changed by this harness repair.

## Remaining risk

Migration checksums, transactional multi-instance locking, and automated
rollback verification remain future migration-system work. The new check proves
the current single-runner adoption and repeat behavior only.
