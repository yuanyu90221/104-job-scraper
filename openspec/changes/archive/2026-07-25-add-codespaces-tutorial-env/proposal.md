## Why

The repo already has three "教學 workflow" (`01-pants-list.yml`, `02-full-build.yml`, `03-incremental.yml`) that demonstrate Pants targets, full builds, and incremental builds — but they only run inside GitHub Actions. A learner who wants to poke at a Pants command, tweak a package, and immediately see the effect on the dependency graph has to push a commit and wait for a workflow run each time. There's no fast, interactive environment that mirrors what CI does. A GitHub Codespaces devcontainer that pre-installs Go + Pants (with the Go 1.24+ `GOEXPERIMENT` patch already applied) gives learners the same tooling as the CI workflows, but with an interactive shell — so they can iterate on `pants list` / `pants dependencies` / `pants --changed-since` in seconds instead of minutes.

## What Changes

- Add `.devcontainer/devcontainer.json` that provisions Go 1.25 and, via `postCreateCommand`, installs Pants and runs `scripts/patch-pants-go.sh` automatically — so a freshly opened Codespace is immediately ready to run `pants` commands.
- Add a short "在 GitHub Codespaces 練習" section to `README.md` describing how to open the repo in a Codespace and which `pants` commands to try first (mirrors what Step 1–3 workflows already run).
- No changes to Go source, no changes to existing workflow YAML files, no breaking changes.

## Capabilities

### New Capabilities
- `devcontainer-tutorial-env`: A GitHub Codespaces devcontainer configuration that provisions Go + Pants (including the Go 1.24+ `GOEXPERIMENT` workaround) so learners get an interactive shell equivalent to what the teaching workflows run in CI.

### Modified Capabilities
(none — no existing specs in `openspec/specs/` are affected)

## Impact

- New file: `.devcontainer/devcontainer.json` (and, if needed, a small `postCreateCommand` script reusing the existing `scripts/patch-pants-go.sh`).
- Updated file: `README.md` — adds Codespaces usage instructions, replacing the manual copy-paste snippet already there.
- No impact on `cmd/`, `internal/`, `go.mod`, or the existing `.github/workflows/*.yml` files.
