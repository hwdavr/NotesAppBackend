# Notes App Backend Product Context

The backend provides authenticated tree-based notes, folders, sync-safe
mutations, sharing, and note block comments for the NotesTakingApp clients.

## Current capabilities

- Auth0 JWT-protected HTTP API with a public `/healthz` endpoint.
- PostgreSQL persistence with paired forward and rollback migrations.
- Version-aware note and folder mutations, soft-delete tombstones, sharing,
  and block comments.
- OpenAPI contract at [`openapi.yaml`](../../openapi.yaml).

## Roadmap

- Add deterministic repository integration coverage for each shared client flow.
- Add migration version tracking when schema rollout needs become multi-instance.

## Harness Feature Tracker

<!-- HARNESS_TRACKER_START -->
| ID | Feature | Workspace | Status | Updated | Notes |
|---|---|---|---|---|---|
| backend-harness | Backend agent harness foundation | [docs/product/backend-harness/](backend-harness/) | Complete | 2026-09-21 | Go/Postgres workflows, rules, templates, and executable quality gates are installed. |
<!-- HARNESS_TRACKER_END -->
