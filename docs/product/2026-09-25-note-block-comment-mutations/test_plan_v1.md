# Test plan

## Feature context

- Request, issue, or feature reference: Implement the note block comment mutation API added by the last `openapi.yaml` commit.
- Parent spec (complex harness-planning only): None.
- Acceptance criteria source: `implementation_plan_v1.md` in this workspace.

## Feature verification requirements matrix

| Verification type | Required for this feature? | Acceptance criterion(s) / reason | Generated test case ID(s) |
| --- | --- | --- | --- |
| Static quality checks | Required | AC-1, AC-4; changed Go, migration, router, and contract wiring must compile and pass repository rules. | TC-STATIC-001 |
| Unit and business-rule tests | Required | AC-2, AC-3; validation, author ownership, resource scoping, and domain error mapping are business rules. | TC-UNIT-001 |
| Component/API behavior | Required | AC-1, AC-2, AC-3, AC-5; real router, middleware, handler, service, and repository wiring must expose the documented operations. | TC-API-001, TC-API-002 |
| API contract checks | Required | AC-1, AC-5; operation paths, methods, operationIds, statuses, and response/request fields must match `openapi.yaml`. | TC-CONTRACT-001 |
| Database state and constraints | Required | AC-4, AC-5; parent/mention data must persist, comment scope must be enforced, and deletion must remove the targeted row. | TC-DB-001 |
| Transaction commit/rollback | Not required — each mutation is one SQL statement and the feature does not introduce a multi-write transaction. |  |  |
| Kafka producer/consumer behavior | Not required — the API has no event producer, consumer, outbox, or broker contract. |  |  |
| Idempotency/retry behavior | Not required — the OpenAPI contract makes no safe-retry or duplicate-suppression promise. |  |  |
| Dependency failure/resilience | Not required — no new external dependency or retry/fallback behavior is introduced; repository errors must still map to controlled 500 responses. |  |  |
| Broader regression | Required | Existing authenticated item/share/comment behavior and repository wiring must remain intact after schema and router changes. | TC-REG-001 |

## Generated required test cases

| ID | Matrix row(s) | Acceptance criterion | Scenario and required assertions | Exact command/selector |
| --- | --- | --- | --- | --- |
| TC-STATIC-001 | Static quality checks | AC-1, AC-4 | Formatting, forbidden-dependency rules, build, vet, migration pairing/wiring, and OpenAPI path/operationId checks pass. | `gofmt -l ...`; `bash harness/scripts/check-go-rules.sh`; `go build ./...`; `go vet ./...`; `bash harness/scripts/check-migrations.sh`; `bash harness/scripts/check-openapi.sh` |
| TC-UNIT-001 | Unit and business-rule tests | AC-2, AC-3 | Valid body/mentions are accepted; blank body and invalid mention shapes are rejected; wrong author maps to `ErrUnauthorized`; missing scoped comment maps to `ErrItemNotFound`. | `go test -json -count=1 ./internal/domain ./internal/http/handlers -run 'Test(Comment|NoteBlockComment)'` |
| TC-API-001 | Component/API behavior | AC-1, AC-2, AC-5 | Authenticated author patches body and mentions and receives a contract-shaped `200`; malformed JSON/blank body receives `400`; unauthenticated request receives `401`. | `go test -json -count=1 -tags=integration ./internal/http -run 'TestAPINoteBlockCommentMutationEndpoints'` |
| TC-API-002 | Component/API behavior | AC-1, AC-3, AC-5 | Non-author receives `403`; missing or wrong-scope comment receives `404`; author delete receives `204` and subsequent list omits the comment. | `go test -json -count=1 -tags=integration ./internal/http -run 'TestAPINoteBlockCommentMutationEndpoints'` |
| TC-CONTRACT-001 | API contract checks | AC-1, AC-5 | New methods and operationIds are present and unique; response/request fields include parent and mentions semantics. | `bash harness/scripts/check-openapi.sh` plus focused integration response assertions |
| TC-DB-001 | Database state and constraints | AC-4, AC-5 | Create/list/update preserve `parentCommentId` and mentions; update changes only mutable fields; delete removes exactly the scoped comment; cleanup removes fixtures. | `go test -json -count=1 -tags=integration ./internal/domain ./internal/http -run 'Test(PostgresIntegrationNoteBlockCommentMutations|APINoteBlockCommentMutationEndpoints)'` |
| TC-REG-001 | Broader regression | Existing behavior | All untagged tests, tagged integration tests, and repository final check pass after the focused scenarios. | `make test-integration`; `make test`; `make check` |

