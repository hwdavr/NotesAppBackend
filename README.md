# Notes App Backend

This project is a Go backend for a mobile notes application. It follows the same layered architecture as `energy-usage-api-starter`: HTTP router and handlers, domain service, repository, and PostgreSQL persistence.

## Stack

- Go
- Chi router
- PostgreSQL
- Auth0 JWT validation
- SQL migrations

## Developer Harness

This repository includes a backend-adapted agent harness based on the
NotesTakingApp workflow. Initialize its entry points and run the complete
quality gate with:

```bash
make harness-init
make check
```

The harness rules and workflows live under `.harness/.agents/`, templates and
checks under `.harness/harness/`, and active delivery artifacts under
`docs/current/`. `make test-integration` uses a disposable PostgreSQL 16
container when Docker is available.

## Environment

Required environment variables:

- `ADDR` server bind address, default `0.0.0.0:8080` (use `0.0.0.0` for access from AVD/other devices)
- `DATABASE_URL` PostgreSQL connection string
- `AUTH0_ISSUER` default `https://dev-9sa8k5kv.us.auth0.com/`
- `AUTH0_AUDIENCE` default `https://notes-app.api`
- `AUTH0_JWKS_URL` default `https://dev-9sa8k5kv.us.auth0.com/.well-known/jwks.json`

Notion account linking additionally requires `NOTION_CLIENT_ID`,
`NOTION_CLIENT_SECRET`, `NOTION_REDIRECT_URL` (the public URL ending in
`/v1/notion/callback`, registered with Notion), and `NOTION_ENCRYPTION_KEY`
(a base64-encoded, randomly generated 32-byte key). Keep the key and client
secret in secret storage. The same encryption key must be retained across
restarts to preserve access to stored credentials. Without these settings,
starting a link returns 503; the rest of the API remains available.


## API

Authenticated routes:

- `GET /v1/items`
- `GET /v1/items/{itemID}`
- `POST /v1/folders`
- `POST /v1/notes`
- `PATCH /v1/items/{itemID}/rename`
- `PATCH /v1/items/{itemID}/move`
- `PATCH /v1/items/{itemID}/reorder`
- `PATCH /v1/notes/{itemID}/content`
- `DELETE /v1/items/{itemID}`

`GET /v1/items` supports:

- `type=folder|note`
- `parentId=<item-id>`
- `rootOnly=true|false`
- `q` search over `name` and note `content`
- `sinceVersion=<number>` incremental sync
- `includeDeleted=true|false` include tombstones

## Tree Model

Folders and notes are stored in one table as independent items, not as one large tree document:

```json
{
  "id": "item_123",
  "type": "folder",
  "parentId": "folder_001",
  "name": "Work",
  "sortKey": "a0",
  "version": 8,
  "deletedAt": null,
  "updatedAt": "2026-04-26T10:00:00Z"
}
```

For notes, the same item also carries `content`.

## Conflict Rules

- Rename uses version-based conflict detection on `name`.
- Note content edits use version-based conflict detection on `content`.
- Move updates `parentId` only.
- Reorder updates `sortKey` only.
- Delete uses soft-delete tombstones via `deletedAt`.
- Every accepted mutation increments `version`.
- Ownership is bound to the Auth0 `sub` claim.

Folder updates are not treated as one big object update. The API separates:

- rename folder
- move folder
- reorder folder
- delete folder
- create note
- move note

## Example Run

```bash
export ADDR=0.0.0.0:8080
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/notes_app?sslmode=disable
export AUTH0_ISSUER=https://dev-9sa8k5kv.us.auth0.com/
export AUTH0_AUDIENCE=https://notes-app.api
export AUTH0_JWKS_URL=https://dev-9sa8k5kv.us.auth0.com/.well-known/jwks.json

go run ./cmd/server
```

The API expects an Auth0 access token in `Authorization: Bearer <token>`. Validation rules:

- issuer must be `https://dev-9sa8k5kv.us.auth0.com/`
- audience must be `https://notes-app.api`
- algorithm must be `RS256`
- signature must validate against the Auth0 JWKS public key
- `exp` must not be expired

## NEON PostgresDB deployment

```
export DATABASE_URL='postgresql://USER:PASSWORD@HOST/DATABASE?sslmode=require'
psql "$DATABASE_URL"

// migrate the DB
make -e migrate
```