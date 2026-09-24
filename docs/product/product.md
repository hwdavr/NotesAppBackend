# Notes App Backend Product Context

The backend provides authenticated tree-based notes, folders, sync-safe
mutations, sharing, and note block comments for the NotesTakingApp clients.

## Current capabilities

- Auth0 JWT-protected HTTP API with a public `/healthz` endpoint.
- PostgreSQL persistence with paired forward and rollback migrations.
- Version-aware note and folder mutations, soft-delete tombstones, sharing,
  and block comments.
- A PostgreSQL integration test covers the core folder/note lifecycle, nested
  listing, content versioning, and soft-delete tombstones.
- Focused API end-to-end suites cover all 21 active router operations across
  health/authentication, items, note shares, and note block comments through
  JWT validation, handlers, service, and PostgreSQL.
- OpenAPI contract at [`openapi.yaml`](../../openapi.yaml).

## Roadmap

The [harness environment](../../.harness/README.md) provides ordered verification
planning, conditional Kafka/idempotency guidance for future projects, capability
inventories, and structured evidence templates. These are agent instructions and
planning artifacts; they do not add tests or runtime capabilities to this service.

- Expand deterministic repository coverage for additional shared-client and
  conflict scenarios.
- Add migration checksum and rollback automation when schema rollout needs become multi-instance.

## Harness Feature Tracker

<!-- HARNESS_TRACKER_START -->
| ID | Feature | Workspace | Status | Updated | Notes |
|---|---|---|---|---|---|
| backend-harness | Backend agent harness foundation | [docs/product/backend-harness/](backend-harness/) | Complete | 2026-09-21 | Go/Postgres workflows, rules, templates, and executable quality gates are installed. |
<!-- HARNESS_TRACKER_END -->
