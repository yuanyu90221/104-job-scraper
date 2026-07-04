## Context

The repo's three teaching workflows (`01-pants-list.yml`, `02-full-build.yml`, `03-incremental.yml`) each install Pants fresh via `get-pants.sh` and then run `scripts/patch-pants-go.sh` to work around Pants' Go backend setting `GOEXPERIMENT=coverageredesign`, which Go 1.24+ no longer supports. That's currently the only place this setup is automated — a learner working locally or in Codespaces has to run the same two commands by hand (already documented as a copy-paste snippet in `README.md`). This change turns that snippet into an automatic Codespace provisioning step.

## Goals / Non-Goals

**Goals:**
- A learner can open this repo in a GitHub Codespace and immediately run `pants list ::`, `pants dependencies`, `pants --changed-since=...` without manual setup steps.
- The devcontainer setup reuses the existing `scripts/patch-pants-go.sh`, so there is exactly one place that encodes the Go 1.24+ workaround.
- README documents how to open the Codespace and what to try first.

**Non-Goals:**
- No changes to the existing CI workflow YAML files.
- No custom web-based tutorial UI — the "interactive environment" is a standard Codespaces terminal.
- No automatic simulation of "which package changed" (Step 3 already covers that manually via `workflow_dispatch` input); the devcontainer just provides the toolchain.

## Decisions

1. **Base image**: use the official `mcr.microsoft.com/devcontainers/go:1.25-bookworm` devcontainer image instead of a custom Dockerfile.
   Alternative considered: hand-rolled Dockerfile installing Go manually — rejected, more to maintain and the official image already matches `go.mod`'s `go 1.25.0`.

2. **Pants installation**: install Pants and apply the patch via `postCreateCommand` (shell one-liners calling `get-pants.sh` then `scripts/patch-pants-go.sh`), rather than baking Pants into a custom image layer.
   Alternative considered: pre-baked custom image with Pants pinned in — rejected for now; it would drift from `pants.toml`'s `pants_version` and require a separate image-build/publish pipeline, which is disproportionate for a teaching aid.

3. **No new shared install script**: `postCreateCommand` calls the exact same two commands already in the CI workflows and in README's Codespaces snippet, instead of extracting a new `scripts/setup-pants.sh`.
   Alternative considered: refactor CI + devcontainer + README to a single shared script — deferred; it touches the existing (working) CI workflows, which is out of scope for this change.

## Risks / Trade-offs

- [Risk] Codespace's Go version could drift from `go.mod`'s `1.25.0` if the base image tag updates. → Mitigation: `pants.toml`'s `minimum_expected_version = "1.21"` already tolerates a range, and the image tag is pinned to major.minor `1-1.25`.
- [Risk] `postCreateCommand` network installs (Pants + patch script) could fail or leave a half-configured container. → Mitigation: `scripts/patch-pants-go.sh` is already idempotent; the same two commands can be re-run manually, which is what README will document.
- [Risk] Running Codespaces has a cost/quota impact for the user's GitHub account. → Mitigation: out of scope to manage; documented as a note in README, no prebuild/auto-start configured.

## Migration Plan

Purely additive — no existing behavior changes or rollback needed:
1. Add `.devcontainer/devcontainer.json`.
2. Update `README.md`'s Codespaces section to reference the automated setup instead of manual copy-paste.
3. Manually verify by opening a Codespace and running `pants list ::`.

## Open Questions

- Should a later iteration mount `~/.cache/pants/lmdb_store` as a Codespaces volume to persist the Pants cache across rebuilds (mirroring the `actions/cache` step in CI)? Deferred — not needed for the first pass.
