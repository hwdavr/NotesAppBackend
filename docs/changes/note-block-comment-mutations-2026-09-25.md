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
