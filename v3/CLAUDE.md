# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Coding guidelines (read first)

Before writing or modifying any Go code in this repo, read [`CODING_GUIDELINES.md`](./CODING_GUIDELINES.md) and follow its conventions. Highlights worth keeping in working memory even before reading the full doc:

- **Never `panic` in production code.** Type assertions must use the two-value `val, ok := x.(T)` form. Regex must use `MustCompile`. `os.Exit` only in `main`.
- **New configurable features need a `Config<Feature>...` option** in `newrelic/config_options.go` AND a matching `NEW_RELIC_<FEATURE>_...` env var parsed in `configFromEnvironment`. Use the existing parse helpers in that file.
- **Use `any`, not `interface{}`.** Use functional options instead of long parameter lists or option structs. Initialize maps with `make`, not `var`. Pointer receivers if the method could mutate.
- **Goroutines need an explicit termination path** (prefer `context`) on a channel independent of data channels. Don't capture loop variables into goroutine literals — pass them as parameters.
- **Don't `defer x.Close()` when `Close` returns an error** — defer a literal that handles the error.
- **Errors:** wrap with `fmt.Errorf("...: %w", err)` when adding context; never silently swallow; never overload a non-error return for error signaling. Lowercase, no trailing punctuation.
- **Slices:** return `nil` for empty; check emptiness via `len(s) == 0`.
- **Copy data at trust boundaries** (slices/maps share backing arrays — callers can mutate).
- **PR title** follows Angular conventions: `<type>(<scope>): <imperative subject>` (e.g., `feat: add nrpgx5 support`). PRs target `develop`, merged via **Squash and Merge** only.

When touching legacy code that doesn't conform, fix it as part of the change.

## Repository layout

This repo holds the New Relic Go Agent. The Go module root is `v3/` (cwd), but the **Makefile, `docker-compose.yml`, `core-tests.mk`, `integration-tests.mk`, and `run-tests.sh` all live one directory up** (`../`). Most commands below must be run from the repo root, not from `v3/`.

- `v3/newrelic/` — public API surface. `Application`, `Transaction`, `Config`, segments, harvest cycle, collector, distributed tracing, attributes, errors. `application.go` and `internal_app.go` are the entry points; `harvest.go` drives the periodic data send loop.
- `v3/internal/` — agent internals not exported to users (utilization detection, CAT, jsonx, sysinfo, logger, gRPC trace observer protobuf in `com_newrelic_trace_v1`, crossagent test fixtures).
- `v3/integrations/` — **each subdirectory is its own Go module** with its own `go.mod`. They depend on `github.com/newrelic/go-agent/v3` via a `replace` directive pointing at `../..` for local development. Adding/touching one of these never affects the others.
- `v3/newrelic/integrationsupport/` and `v3/newrelic/sqlparse/` — small helper packages used by integration packages.
- `v3/examples/` — runnable sample apps (`server/main.go` is the canonical one referenced in the README).

## Common commands

Run `make` targets **from the repo root** (`..` from cwd), not from `v3/`:

```sh
make core-test TEST=internal/utilization      # single core test (path relative to v3/)
make integration-test TEST=nrgin              # single integration test (path relative to v3/integrations/)
make core-suite                               # all core tests
make integration-suite                        # all integration tests
make tidy                                     # add local replace directive + go mod tidy
make dev-shell                                # docker dev env with all service containers; dev-stop to tear down
```

Append `COVERAGE=1` to a test target to write `coverage.txt`. Integration tests need service containers — `dev-shell` brings them all up; otherwise use `make test-services-start PROFILE=<name>` (profile names match integration dirs, e.g. `nrpgx5`, `nrmongo-v2`). Override Go version with `GO_VERSION=1.23`.

**Don't `go test ./...` from `v3/`** — it won't traverse the integration modules and the local replace directive won't be in place. Always go through the make targets.

## Architecture notes

- **`Application` is nil-safe.** Every method checks `app == nil` so a nil `*Application` works as a mock. Preserve this when adding methods. Same convention for `*Transaction`.
- **Harvest cycle.** `app.process` (in `internal_app.go`) runs a single goroutine that owns the connect/harvest state. Data is delivered via `dataChan`; harvests are scheduled via `harvestTimer` (`harvest.go`) with separate periods per data type (metrics/traces, span events, custom events, log events, txn events, error events). Shutdown coordinates through two channels (`shutdownStarted`, `shutdownComplete`) — read the comments at the top of `app` struct in `internal_app.go` before changing channel logic; the two-channel design exists specifically to prevent a deadlock during shutdown harvest failures.
- **`appRun` is immutable** once set. Mutating it after connection is a bug. Access `run`/`err` only via `getState`/`setState`.
- **Integration packages** are independently versioned. Each has its own `go.mod` with a `replace` pointing to `../..`. When adding a new integration, follow the wiki "Writing a New Integration Package" guide and the structure of an existing simple integration (`nrgin` is a good template).

## Conventions to preserve

- **Wide compatibility window.** Instrumentation must work with a broad range of versions of the third-party module it wraps. Code that looks overcomplicated is often that way for compatibility — read comments and tests before "simplifying."
- **Test manifests must stay in sync with the filesystem.** Any new or moved test directory under `v3/newrelic/` or `v3/internal/` must be added to `core-tests.mk`; new directories under `v3/integrations/` go in `integration-tests.mk`. CI iterates these lists, not the filesystem.
