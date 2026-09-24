# Implementation plan

## Feature context

- Request, issue, or feature reference: Implement the note block comment mutation API added by the last `openapi.yaml` commit.
- Scope and exclusions: Register and implement `PATCH /v1/notes/{itemID}/blocks/{blockID}/comments/{commentID}` and `DELETE` for the same resource. Persist and return the contract's `parentCommentId` and `mentions` fields for comment creation, listing, and update. Do not add messaging, retry/idempotency guarantees, or undocumented moderation behavior.
- Impacted layers, requirements, and acceptance criteria:
  - `REQ-CMT-MUT-001`: The authenticated author can update a comment's body and mentions, while note/block/comment identity and author identity remain server-controlled.
  - `REQ-CMT-MUT-002`: The authenticated author can delete their comment and receives `204 No Content`.
  - `REQ-CMT-AUTH-001`: Non-authors receive `403`; missing or inaccessible resources receive `404`; unauthenticated requests remain covered by the existing middleware.
  - `REQ-CMT-DATA-001`: Comment parent relationships and mention references survive create, list, update, and delete operations with no partial mutation.
  - `AC-1`: Both documented operations are registered with the existing JWT-protected router and use the documented statuses and JSON shapes.
  - `AC-2`: Update trims and validates the body, validates mention references, checks the comment's note/block scope, and permits only its author to mutate it.
  - `AC-3`: Delete checks the same scope and author ownership, removes exactly the requested comment, and returns `204` with no response body.
  - `AC-4`: Comment persistence has a forward and rollback migration for nullable self-parenting and JSONB mentions; reads return an empty mention array rather than `null`.
  - `AC-5`: Focused API integration scenarios cover success, malformed input, missing comment, wrong author, parent/mention persistence, delete visibility, and auth behavior.
- Parent spec (complex harness-planning only): None; this is a single-slice feature.

## Implementation approach

Implement the feature as one coherent change across the persistence, domain,
transport, and integration-test layers. Add the paired comment migration and
domain operations first, then register the existing contract operations in the
handler/router and cover the complete authenticated HTTP behavior with the
planned PostgreSQL scenarios. The change is intentionally not divided into
independently tracked slices.

## Verification environment

- Project capability profile and triggered optional rules: `docs/product/project-capabilities.json`; HTTP contract rules, security rules, PostgreSQL testing strategy, and messaging/idempotency applicability policies. Kafka is not applicable because this API emits no events. Idempotency is not required because the contract promises neither duplicate suppression nor safe retry semantics.
- Existing executable checks vs new verification work explicitly in scope: Existing `bash harness/scripts/check-openapi.sh`, `go build ./...`, `go vet ./...`, `make test`, `make test-integration`, and `make check` remain in scope. New focused integration scenarios are required because the existing comment test covers only list/create and cannot prove the new mutation contract.
- Missing required tooling/tests/infrastructure and resolution: The project has no automated OpenAPI schema/runtime conformance validator and no isolated dynamic-port integration runner. Record those as verification gaps; use the existing disposable PostgreSQL Compose profile for the required integration scenarios and report any unavailable runtime as `BLOCKED`.
- Gate ordering, targeted selectors, and final regression: Run static checks first, then targeted package tests, then tagged API/PostgreSQL integration tests, then the broader regression and final `make check`. Record command, source identity, counts, cleanup, and scenario assertions in the versioned summary.

## Approval

- Status: Approved
- Approved by/date: User / 2026-09-25
