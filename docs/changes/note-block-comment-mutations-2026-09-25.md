# Note block comment mutations

Implemented the API operations added in the latest `openapi.yaml` contract:

- `PATCH /v1/notes/{itemID}/blocks/{blockID}/comments/{commentID}`
- `DELETE /v1/notes/{itemID}/blocks/{blockID}/comments/{commentID}`

The implementation adds author-only mutation checks, scoped comment lookup,
body/mention validation, parent comment persistence, JSONB mention storage,
and a rollback-paired migration. Both Compose bootstrap profiles mount the new
forward migration.

Verification passed with focused unit tests, PostgreSQL integration tests,
`make test-integration`, `make test`, and `make check`. The disposable
integration database was cleaned up after testing.

Follow-up hardening added stable public error messages, strict single-value JSON
decoding, mention span bounds, removal of the unused comment-update `409`
contract, and a configurable/idempotent migration runner with schema adoption
for existing databases. The remaining migration roadmap item is checksum and
rollback automation for multi-instance deployments.
