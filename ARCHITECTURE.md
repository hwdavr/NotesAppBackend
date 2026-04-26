# Notes App Backend Architecture

## High-Level Flow

```
HTTP server (cmd/server/main.go)
        |
        v
Router + middleware (internal/http)
        |
        v
Handlers (internal/http/handlers)
        |
        v
Domain service (internal/domain/service.go)
        |
        v
Repository (internal/domain/repository.go)
        |
        v
PostgreSQL
```

## Modules

- `cmd/server`: process startup and dependency wiring
- `internal/config`: environment-backed configuration
- `internal/db`: PostgreSQL connection setup
- `internal/domain`: business entities, service logic, repository access
- `internal/http`: router and Auth0 JWT middleware
- `internal/http/handlers`: item and tree mutation endpoints
- `migrations`: schema setup for tree items and sync metadata

## Main Tree Flow

1. Mobile app obtains an Auth0 access token.
2. Client calls authenticated tree endpoints with `Authorization: Bearer <token>`.
3. Middleware validates `iss`, `aud`, `alg=RS256`, JWKS signature, and `exp`.
4. Middleware extracts the Auth0 `sub` claim and injects it into request context.
5. Handlers delegate to the domain service.
6. Service applies version-based conflict rules per field instead of per whole tree.
7. `name` and note `content` use conflict detection against `lastSyncedVersion`.
8. `parentId` and `sortKey` mutate independently through move and reorder endpoints.
9. Repository increments item `version`, stores `deviceId`, and preserves delete tombstones via `deletedAt`.
