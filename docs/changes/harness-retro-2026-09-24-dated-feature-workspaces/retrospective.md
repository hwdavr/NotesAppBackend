# Harness retrospective — dated feature workspaces

## Incident

The backend harness directed feature and bug artifacts to `docs/current/`, and
`check-stage-artifacts.sh` silently defaulted to that directory when no
workspace argument was supplied. This made dated feature workspaces optional
rather than enforced.

## Classification and root cause

`WORKFLOW_GAP` — the workflow instructions, initializer, README, and stage
validator did not share an authoritative requirement for a dated feature
workspace.

## Invariant

Every stage-artifact check must receive an explicit
`docs/product/YYYY-MM-DD-feature` workspace; `docs/current` must never be used
as an implicit artifact location.

## Harness change

- Replaced `docs/current` guidance in `AGENTS.md`, the README, all delivery,
  bug-fixing, contract-update, and review workflows, the initializer, and the
  retrospective skill with `docs/product/YYYY-MM-DD-feature/` workspaces.
- `harness/scripts/check-stage-artifacts.sh` now requires a third workspace
  argument and rejects every path outside the dated workspace format.
- Added `harness/scripts/tests/check-stage-artifacts-date-feature.sh` and a
  fixture containing both the old and dated layouts. The former implicit call
  is rejected; the explicit dated path is accepted.
- Moved the active API coverage planning artifacts to
  `docs/product/2026-09-24-api-endpoint-integration-coverage/`.

## Verification

| Command | Result |
| --- | --- |
| `bash -n .harness/harness/scripts/check-stage-artifacts.sh` | Passed |
| `bash -n .harness/harness/scripts/init-harness.sh` | Passed |
| `bash .harness/harness/scripts/tests/check-stage-artifacts-date-feature.sh` | Passed; the no-workspace call exited nonzero and the dated fixture passed. |
| `bash harness/scripts/check-stage-artifacts.sh feature-delivery requirement-analysis docs/product/2026-09-24-api-endpoint-integration-coverage` | Passed |
| `bash harness/scripts/check-stage-artifacts.sh feature-delivery implementation-plan docs/product/2026-09-24-api-endpoint-integration-coverage` | Passed |
| `bash harness/scripts/check-feature-lifecycle.sh` | Passed |
| `make check` | Passed |
| `git diff --check` | Passed |

## Routed items

No application source, OpenAPI contract, database schema, migration, or
feature lifecycle status was changed.

## Remaining risk

Existing developer-authored references outside the harness may still need a
separate migration if discovered later.
