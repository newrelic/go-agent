# Run Integration

Run an example from an integration. If no integration is passed as an argument, use the current workspace integration.

ARGUMENTS: $ARGUMENTS

## Steps

1. Determine the integration to run:
   - If an argument was passed, use it.
   - If no argument was passed, read `.dev-workspace.json` and use `active_integration`. If that is also empty, prompt the user to select one.

2. Find examples: look for an `example/` or `examples/` directory under `v3/integrations/<integration>/`. If there are multiple example programs, list them and ask the user to pick one.

3. Prepare the integration's go.mod if needed:
   - If the integration differs from the current workspace, add the replace directive and run tidy:
     ```bash
     cd v3/integrations/<integration>
     go mod edit -replace github.com/newrelic/go-agent/v3="<repo_root>/v3"
     go mod tidy
     ```
   - Do **not** change the `go` version line — if `go mod tidy` updates it, revert it.
   - Do **not** update `.dev-workspace.json` — this is a temporary setup for running only.

4. Read the comments at the top of the chosen example's `main.go` for setup instructions (local databases, Docker containers, etc.) and follow them before running.

5. Determine env vars for running:
   - Check `env_vars` in `.dev-workspace.json` — any keys set there override the environment.
   - Otherwise, use the user's existing shell environment as-is. Do not prompt for env vars that are already likely set (e.g. `NEW_RELIC_LICENSE_KEY`) — just run with the current environment.

6. Run the example:
   - Read the `main.go` to determine if it starts an HTTP server (look for `http.ListenAndServe`, `router.Run`, `srv.ListenAndServe`, etc.).
   - If it is a server, run it in the **background** using the `run_in_background` parameter on the Bash tool.
   - After starting a background server, wait a moment for it to be ready, then read the routes and their handler functions from `main.go` and provide example `curl` commands the user can run to exercise each route.
   - If it is not a server (i.e. it runs and exits), run it in the foreground.

7. After the example exits (or after the user asks you to stop a background server), if you temporarily modified the go.mod in step 3, restore it:
   - Remove the `replace` directive line.
   - Remove all indirect dependencies (lines ending in `// indirect`) and the second `require (...)` block if it only contained indirect deps.
   - Do **not** run `go mod tidy`.
   - Do **not** modify `.dev-workspace.json`.

## Notes

- Always use absolute paths in replace directives, not relative paths
- Never change the `go` version line in any go.mod
