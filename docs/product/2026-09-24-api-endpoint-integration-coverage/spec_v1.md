# Full API endpoint integration coverage

## Scope

Split the existing broad API lifecycle integration test into focused,
integration-tagged Go test files. Cover every HTTP operation currently
registered by `internal/http/router.go` through a local HTTP listener, local
RS256 JWKS server, real handlers/services/repositories, and the disposable
PostgreSQL 16 database.

The runtime route inventory contains 19 operations:

| Area | Operations |
| --- | --- |
| Health | `GET /healthz` |
| Items and debug identity | `GET /v1/items`, `GET /v1/debug/me`, `GET /v1/items/{itemID}`, `POST /v1/folders`, `POST /v1/notes`, six item mutation patches, and `DELETE /v1/items/{itemID}` |
| Note shares | List, create, update, and delete under `/v1/notes/{itemID}/shares` |
| Note block comments | List and create under `/v1/notes/{itemID}/blocks/{blockID}/comments` |

## Exclusions

- No production API, database schema, migration, or dependency changes.
- No live Auth0, email provider, shared database, or real user data.
- OpenAPI declares `PATCH` and `DELETE` comment-by-ID operations that the
  router does not expose. They cannot be covered as request-to-response
  endpoints without separately implementing those routes.
- The existing OpenAPI edit is user-owned and will not be changed by this test
  coverage work.

## Contract observation

The current runtime and OpenAPI inventories differ:

| Runtime route missing from OpenAPI | OpenAPI operation missing from runtime |
| --- | --- |
| `GET /v1/debug/me` | `PATCH /v1/notes/{itemID}/blocks/{blockID}/comments/{commentID}` |
|  | `DELETE /v1/notes/{itemID}/blocks/{blockID}/comments/{commentID}` |

This delivery will cover all 19 active router operations and preserve this
observation for a separate contract/feature decision.

## Acceptance criteria

1. Tests are split by health, items, note shares, and note block comments,
   with shared authenticated HTTP/fixture helpers.
2. Every active router operation has a successful API request-to-response
   assertion for its documented response shape and status.
3. Authenticated routes use locally signed JWTs; the suite verifies missing
   bearer-token rejection and an invalid-audience rejection.
4. State-changing requests verify durable behavior through a subsequent API
   response where applicable.
5. Tests use isolated deterministic fixture users and a silent email fake; no
   tokens, emails, note bodies, or database URLs are printed.
6. `make check` and `make test-integration` pass with non-zero executed tests.

## Rule Applicability

| ID | Rule | Decision | Evidence / reason |
| --- | --- | --- | --- |
| ARCH | Layer boundaries and dependency direction | Required | Test-only wiring must exercise the existing router → handler → domain → repository boundary without adding prohibited production dependencies. |
| API | OpenAPI and client contract | Required | The route inventory is cross-checked against `openapi.yaml`; observed drift is documented without changing the user-owned contract. |
| DB | Persistence and transaction semantics | Required | API mutations and reads run against the disposable PostgreSQL database with deterministic cleanup. |
| MIG | Migration and rollback safety | Not applicable — test coverage adds no schema or migration change | Existing compose bootstrap migrations remain unchanged. |
| TEST | Unit, HTTP, and integration tests | Required | Focused integration-tagged API tests assert status, headers, JSON fields, and persisted effects. |
| SEC | Authentication, authorization, and data handling | Required | Local RS256 JWKS validates verified identity only; negative authentication cases are included and test data contains no real identities. |
| OBS | Logging, errors, and operational signals | Required | Tests use no-op logging and a silent email fake; failures avoid echoing sensitive payloads. |
| PERF | Query and runtime performance | Not applicable — no performance behavior changes | The suite does not introduce production query or runtime behavior. |
| DEP | Dependency and build reproducibility | Required | Use only existing Go modules, Docker compose service, and Make targets. |
| DOC | Documentation and delivery records | Required | Versioned plan, test plan, summary, product tracker, and durable change record are maintained. |
