# Implementation plan — full API endpoint integration coverage

Rule decisions: [spec_v1.md#rule-applicability](spec_v1.md#rule-applicability)

1. Extract the existing test's reusable local-JWKS, router, disposable-database,
   authenticated-request, JSON assertion, and fixture-cleanup helpers into a
   shared integration test support file. Replace the logging email mock with a
   silent test fake for share coverage.
2. Add a health/auth test file for `GET /healthz`, missing bearer-token
   rejection, invalid-audience rejection, and `GET /v1/debug/me` identity
   propagation.
3. Replace the broad item lifecycle test with focused item endpoint tests:
   create/list/get, rename, move, reorder, favorite, generic item content,
   note content, and delete. Each mutation asserts its response and a
   follow-up API read where state should persist.
4. Add note-share endpoint tests for create/list/update/delete using the real
   service/repository with deterministic share fixtures.
5. Add note-block-comment endpoint tests for create/list using the real
   service/repository with deterministic note and block fixtures.
6. Keep the two OpenAPI-only comment update/delete operations out of this
   suite because no router endpoint exists; retain the documented contract
   observation without modifying user-owned `openapi.yaml`.
7. Run formatting, the full quality gate, and the real Docker-backed
   integration target. Record exact commands, statuses, and executed test
   count in `summary_v1.md`; then update product and change records.

