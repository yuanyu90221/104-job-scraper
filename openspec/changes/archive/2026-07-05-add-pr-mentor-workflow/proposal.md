## Why

The repo's existing Pants Build teaching material (`01-pants-list.yml`, `02-full-build.yml`,
`03-incremental.yml`) requires a learner to manually trigger `workflow_dispatch` steps and read Job
Summaries — it teaches by demonstration, not by doing. The pedagogical goal is to make learners
experience incremental compilation as a natural side effect of their own workflow: edit code, open a
PR, and get real-time feedback grounded in the actual dependency graph and actual `pants` build
output, without needing an LLM in the loop.

## What Changes

- Add a new, additive GitHub Actions workflow, `.github/workflows/04-pr-mentor.yml`, triggered on
  `pull_request` (`opened`, `synchronize`) against `main`.
- Detect which "stage" of the tutorial a PR is in using rule-based logic (no LLM call):
  - Cold cache (no PR-scoped Pants cache hit yet) → Stage 1 (full build).
  - Cache warm and the real `pants --changed-since=<base_sha> --changed-dependents=transitive list`
    output includes `//internal/models` → Stage 3 (shared/common package changed, full fan-out
    rebuild).
  - Cache warm and that output includes `//internal/notifier` or `//internal/formatter` → Stage 2
    (leaf package changed, precise incremental rebuild).
  - Anything else → a generic "other" stage nudging the learner toward the two known packages.
- Run real `pants test` (full or `--changed-since`-scoped, depending on stage) and use its actual
  elapsed time and the real changed-target list as the data behind the mentor comment — no simulated
  `echo` output.
- Use a **PR-scoped** Pants LMDB cache key (`pants-lmdb-tutorial-pr<PR#>-...`), distinct from the
  shared key used by `01`/`03`/`pants-ci.yml`, so every new PR reliably starts cold and Stage 1 is
  reproducible regardless of what `pants-ci.yml` already warmed on `main`.
- Post a single **sticky** PR comment (find-and-update via a hidden `<!-- pant-mentor -->` marker)
  rather than one new comment per push.
- Update `README.md`'s existing Pants teaching section to mention this new automatic Step 4 and point
  learners at the two example edits (`internal/notifier/line.go`, `internal/models/job.go`).
- Do **not** modify `01-pants-list.yml`, `02-full-build.yml`, `03-incremental.yml`, or `pants-ci.yml`.

## Capabilities

### New Capabilities
- `pr-mentor`: Automatic, PR-triggered feedback that classifies a pull request into a tutorial stage
  (cold full build / leaf incremental / common-package fan-out rebuild) using real Pants build output,
  and posts/updates a single explanatory PR comment.

### Modified Capabilities
- none (no existing spec's requirements change; `job-notification` is unrelated to this workflow)

## Impact

- New file: `.github/workflows/04-pr-mentor.yml`.
- Modified: `README.md` (teaching section only, additive).
- No changes to application code (`cmd/`, `internal/**`) as part of this change — `internal/notifier`
  and `internal/models` are referenced only as example targets the *learner* edits after this
  workflow ships.
- No changes to `pants.toml`, `pants.ci.toml`, or existing workflow files.
- New GitHub Actions permission usage: `pull-requests: write` (comment create/update), `contents:
  read` (checkout only) — least-privilege, no new secrets required.
