# Test plan

## Feature context

- Request: Notion account linking, status and disconnect.
- Acceptance criteria source: `implementation_plan_v1.md`, AC-1 through AC-5.

## Feature verification requirements matrix

| Verification type | Required for this feature? | Acceptance criterion(s) / reason | Generated test case ID(s) |
| --- | --- | --- | --- |
| Static quality checks | Required | AC-1–5 | TC-STATIC |
| Unit and business-rule tests | Required | AC-1–5 | TC-UNIT |
| Component/API behavior | Required | AC-1–5 | TC-API |
| API contract checks | Required | AC-1–5 | TC-CONTRACT |
| Database state and constraints | Required | AC-1–3, AC-5 | TC-DB |
| Transaction commit/rollback | Required | AC-3 | TC-TX |
| Kafka producer/consumer behavior | Not required | Linking produces or consumes no events | — |
| Idempotency/retry behavior | Not required | Contract promises no general retry or duplicate-suppression semantics; single-use state is covered by security scenarios | — |
| Broader regression | Required | AC-1–5 | TC-REG |

## Generated required test cases

Commands assume the named tests will be created during implementation; selectors must execute nonzero tests. Record actual counts, skips, source identity, mock boundaries and assertions.

| ID | Matrix row(s) | Acceptance criterion | Scenario and required assertions | Exact command/selector |
| --- | --- | --- | --- | --- |
| TC-STATIC | Static | AC-1–5 | Formatting, source boundaries, build, vet, migration and contract checks pass | `make fmt-check && go build ./... && make vet && bash harness/scripts/check-openapi.sh` |
| TC-UNIT | Unit | AC-1–5 | URL/state construction, expiry and replay, token encryption, cancellation, upstream and configuration failures; deterministic fake clock/Notion server as needed | `go test -json -count=1 ./internal/domain/... ./internal/pkg/... -run 'TestNotion'` |
| TC-API | Component/API | AC-1–5 | Real router with synthetic JWT/JWKS and mocked Notion/storage boundaries: success, unauthenticated requests, callback invalid/reused state, cancellation, failures, redirect, caller isolation and no secret disclosure | `go test -json -count=1 ./internal/http/... -run 'TestNotion'` |
| TC-CONTRACT | API contract | AC-1–5 | Static validity and compatibility against pre-change contract plus runtime response conformance for all changed operations and callback. Existing script only checks paths/operation IDs; establish suitable tooling or record BLOCKED | `bash harness/scripts/check-openapi.sh`; `redocly lint openapi.yaml --extends minimal`; reconstructed-baseline comparison; `NOTION_OPENAPI_PATH="$PWD/openapi.yaml" NOTION_NODE_MODULES=/opt/homebrew/lib/node_modules/appium/node_modules NOTION_CONFORMANCE_VALIDATOR="$PWD/docs/product/2026-11-09-notion-account-linking/runtime-contract-validator.cjs" go test -json -count=1 ./internal/http/... -run ^TestNotion` |
| TC-DB | Database | AC-1–3, AC-5 | Disposable PostgreSQL: state ownership, expiry and single consumption, encrypted token persistence, replacement and cross-user isolation, delete and migration rollback | `go test -json -count=1 -tags=integration ./internal/domain -run 'TestPostgresIntegrationNotion'` with isolated disposable DB and migrations |
| TC-TX | Transactions | AC-3 | Verify disconnect commits both deletions; induce a database failure to show rollback with no partial state; concurrent callback/disconnect cannot relink after disconnect | `go test -json -count=1 -tags=integration ./internal/domain -run 'TestPostgresIntegrationNotion'` with isolated disposable DB |
| TC-REG | Regression | AC-1–5 | Fresh broad untagged and relevant tagged tests, followed by project handoff gate | `go test -json -count=1 ./...`; `make test-integration` only if its disposable fixed-port environment is owned and isolated; `make check` |

Stop after an earlier mandatory failure or blocked gate. Use `harness/verification-evidence.md` for receipts; no test or capability declaration alone proves execution.
