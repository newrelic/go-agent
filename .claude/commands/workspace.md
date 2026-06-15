# Workspace

Manage the active development workspace for the go-agent repo.

The workspace state is stored in `.dev-workspace.json` at the repo root. This file is gitignored and local to your machine.

## Schema

```json
{
  "active_integration": "nrgin",        // integration path relative to v3/integrations/, e.g. "nrgin" or "logcontext-v2/nrlogrus"
  "environment": "local",               // future: "local" | "staging" | "prod" — for injecting env var profiles
  "env_vars": {}                        // future: map of env var name -> value to inject when running/testing
}
```

## Commands

### `/workspace` — show current workspace

Read `.dev-workspace.json` and report:
- Active integration (or "none" if empty)
- Environment (or "not set" if empty)
- Any env vars configured (or "none")

### `/workspace <integration>` — switch active integration

`<integration>` is the integration path as it appears in `integration-tests.mk`, e.g.:
- `nrgin`
- `nrpgx5`
- `logcontext-v2/nrlogrus`

Steps:
1. Read the current `.dev-workspace.json`.
2. Validate that `v3/integrations/<integration>/go.mod` exists. If it doesn't, report an error and list valid integrations by reading `integration-tests.mk`.
3. If there is a currently active integration, restore it first (see **Restore a go.mod** below) before proceeding.
4. Check if the new integration's go.mod already has a `replace github.com/newrelic/go-agent/v3` directive. If it does not, add it:
   ```bash
   cd v3/integrations/<new_integration>
   go mod edit -replace github.com/newrelic/go-agent/v3="<repo_root>/v3"
   ```
   Where `<repo_root>` is the absolute path to the repo root (parent of `v3/`). Then run `go mod tidy`.
5. Do **not** change the `go` version line in the go.mod — if `go mod tidy` updates it, revert it to what it was before.
6. Update `.dev-workspace.json` with the new `active_integration`.
7. Report what changed.

### `/workspace core` — switch to core module

Sets active integration to the special value `"core"`, which refers to the root `v3/` module rather than any integration.

Steps:
1. If there is a currently active integration (not `"core"`), restore it first (see **Restore a go.mod** below).
2. Run tidy on the core module:
   ```bash
   cd v3
   go mod tidy
   ```
3. Update `.dev-workspace.json` with `active_integration: "core"`.

### `/workspace clear` — clear active integration

If there is a currently active integration, restore it (see **Restore a go.mod** below). Then set `active_integration` to `""` in `.dev-workspace.json`.

## Restore a go.mod

To restore a go.mod to its pre-workspace state, edit the file directly:
1. Remove the `replace` directive block (the line starting with `replace github.com/newrelic/go-agent/v3`).
2. Remove all indirect dependencies — every line in the `require` block that ends with `// indirect`, and remove the entire second `require (...)` block if it only contained indirect deps.
3. Do **not** change the `go` version line.
4. Do **not** run `go mod tidy`.

## Notes

- The replace directive format is always: `github.com/newrelic/go-agent/v3` → absolute path to `v3/`
- Always use absolute paths in the replace directive, not relative paths
- The `environment` and `env_vars` fields are reserved for future use — do not modify them unless explicitly asked
- Never change the `go` version line in any go.mod — if `go mod tidy` modifies it, revert it
