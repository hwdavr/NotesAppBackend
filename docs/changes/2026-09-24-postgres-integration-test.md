# PostgreSQL and API integration coverage

Added `internal/domain/repository_integration_test.go` with the `integration`
build tag. The test runs the real domain service and repository against the
PostgreSQL schema initialized by `build/docker-compose.test.yml` and verifies:

- folder and nested note creation;
- root and child listing;
- note content update with version increment; and
- soft-delete visibility and tombstone retrieval.

Added a focused API end-to-end suite with the same `integration` build tag:

- `internal/http/health_e2e_integration_test.go` verifies `/healthz`, verified
  JWT subject propagation, and missing-token/invalid-audience rejection.
- `internal/http/items_e2e_integration_test.go` verifies all item operations:
  list, get, create folder/note, rename, move, reorder, favorite, both content
  updates, and delete.
- `internal/http/shares_e2e_integration_test.go` verifies note-share list,
  create, update, and delete operations with a silent email fake.
- `internal/http/comments_e2e_integration_test.go` verifies note-block-comment
  list and create operations.

Shared support in `internal/http/api_e2e_integration_test.go` starts the
production router with a real PostgreSQL repository and a local RS256 JWKS
server. The suite now covers every active router operation (19 total) through
the HTTP boundary. OpenAPI-only comment update/delete operations remain out of
scope because the router does not register them.

Verification:

- `make check` — passed after adding the focused API suites.
- `go test -tags=integration -run '^$' ./internal/domain ./internal/http` —
  compiled both integration packages successfully (the zero-test selector is
  intentional and does not claim runtime coverage).
- `go test -tags=integration -list '^Test' ./internal/domain ./internal/http`
  — passed and listed seven executed test functions: one repository lifecycle
  test, four focused API E2E groups, and two existing router tests.
- `make test-integration` — passed (exit 0). The disposable PostgreSQL 16
  container became healthy, then all seven listed tests passed.
