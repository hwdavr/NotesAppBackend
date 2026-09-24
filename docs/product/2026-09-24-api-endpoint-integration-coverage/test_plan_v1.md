# Test plan — full API endpoint integration coverage

Rule decisions: [spec_v1.md#rule-applicability](spec_v1.md#rule-applicability)

## Test matrix

| Test group | Runtime operations | Required assertions |
| --- | --- | --- |
| Health and authentication | `GET /healthz`, missing-token rejection, invalid-audience rejection, `GET /v1/debug/me` | Status, plain-text or JSON content type, verified subject propagation, and 401 rejections. |
| Item retrieval and creation | `GET /v1/items`, `GET /v1/items/{itemID}`, `POST /v1/folders`, `POST /v1/notes` | 200/201 status, JSON shape, ownership from JWT subject, parent/child state. |
| Item mutations | Rename, move, reorder, favorite, generic content, note content, delete | 200 status, `MutationResult`, version/state update, and follow-up API read/list where appropriate. |
| Note shares | List, create, update, delete | 200/201/204 statuses, JSON shape, updated access role, and post-delete empty list. |
| Block comments | List and create | 200/201 statuses, JSON shape, authenticated author identity, and post-create list result. |

## Runtime environment

- Test files retain `//go:build integration`.
- `make test-integration` starts only `build/docker-compose.test.yml`'s
  disposable PostgreSQL service and supplies its database URL.
- Tests start local `httptest` API/JWKS servers and never contact Auth0 or an
  email provider.

## Verification commands

```bash
gofmt -w internal/http/*_integration_test.go
make check
make test-integration
```

Success requires each command to exit 0 and `make test-integration` to execute
all non-empty integration test groups.
