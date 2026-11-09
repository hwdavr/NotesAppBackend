# Implementation plan

## Feature context

- Request: implement Notion account linking using the existing OpenAPI contract.
- Scope: `POST /v1/notion/link`, `GET` and `DELETE /v1/notion/connection`, and the OAuth browser callback necessary to complete linking. The already documented page, folder and note-reading operations are outside account linking and remain outside this change.
- AC-1: An authenticated user can start linking and receive a Notion authorization URL; the backend binds a cryptographically random, expiring, single-use state to that verified user.
- AC-2: The public callback validates and consumes state, exchanges a successful authorization code with Notion using server-held credentials and bounded I/O, persists the user's connection, and redirects to the documented app deep link with `linked`, `cancelled`, or `failed`. Invalid, expired, reused or missing state cannot link an account; failure never exposes credentials or the code.
- AC-3: Connection status reports only the caller's linked state and optional workspace/account name; disconnect removes that caller's stored credentials and pending authorization, without affecting other users. A successful subsequent link replaces that user's existing connection.
- AC-4: Missing configuration, storage failure, and upstream failure are handled safely, including documented 503 responses where applicable. Bearer authentication protects all three app-facing operations; callback uses validated state rather than bearer authentication.
- AC-5: Credentials and OAuth parameters are never logged or returned to the app; database migrations have matching rollback and bootstrap support. The callback's public route, parameters, redirect and error behavior are documented in OpenAPI.

## Implementation approach

Add a small Notion integration service and HTTP client for authorization URL creation and token exchange, configured by environment (client ID, client secret, registered callback URL, and a dedicated encryption key). Require valid configuration when linking; keep the existing server usable without it. Generate random state with `crypto/rand`, store only a hash with user ID and expiry in PostgreSQL, and atomically consume it before handling a callback. Encrypt access tokens at rest with authenticated encryption using the configured key; do not expose or log token responses, raw state, authorization codes, secrets, or database URLs. Make the deep link a fixed server-controlled value. Use a bounded request timeout and context for Notion exchange; account name comes from the returned workspace name. Define explicit behavior for cancellation and failed or malformed callbacks and use the existing response error conventions.

Add PostgreSQL repository methods and paired migration(s) for state and per-user connection, unique ownership and cleanup, updating migration bootstrap and runner as needed. A disconnect invalidates pending state and deletes the connection atomically; ensure a callback cannot restore a disconnected connection after that deletion. Do not modify generated Bob files by hand. Add router/handlers and configuration wiring; document the callback in `openapi.yaml` without overwriting existing user changes. Add focused unit, HTTP and real PostgreSQL tests; update product documentation and delivery notes on completion.

## Verification environment

- `docs/product/project-capabilities.json`; API, security, PostgreSQL, idempotency and messaging policy rules apply. No event or explicit retry guarantee is specified. State single-use is a security requirement, tested independently of a general idempotency promise.
- Existing checks: `go build`, `go vet`, `make check`, existing tests and tagged PostgreSQL tests. New focused tests and fixture-safe PostgreSQL execution are required.
- The inventory identifies missing OpenAPI schema compatibility and runtime conformance tooling and isolated integration orchestration. Required evidence remains BLOCKED if it cannot be established; the fixed-port shared Compose setup must not be used unless isolated ownership and cleanup are established.
- Ordered static → unit → component/API → integration → regression gates and selectors are in the test plan. Run `make check` before handoff, recording failures and unavailable dependencies honestly.

## Approval

- Status: Approved
- Approved by/date: User / 2026-11-09
