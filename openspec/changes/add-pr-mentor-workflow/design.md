## Context

The repo already has three `workflow_dispatch` teaching steps and a real `pants-ci.yml` gate. Their
shared Pants LMDB cache key is `pants-lmdb-${{ runner.os }}-${{ hashFiles('pants.toml', 'go.sum') }}`
— this key is warmed on every push to `main` by `pants-ci.yml`, so any new workflow that reused it
would almost always see a cache hit, defeating a "cold start" lesson. The real dependency graph
(confirmed via Go imports + BUILD files) is:

- `internal/models` — fan-in 4 (imported by `client`, `formatter`, `notifier`, `search`), no internal
  deps of its own → the "shared/common" package.
- `internal/notifier` and `internal/formatter` — fan-in 0, depend only on `models` → "leaf" packages.
- `cmd` depends on all four leaves/mid-tier packages and is the final build target (`//cmd:bin`).

Full detail and rationale live in `/home/yuanyu/.claude/plans/github-action-hazy-ocean.md` (the
approved plan this change implements).

## Goals / Non-Goals

**Goals:**
- Classify each PR push into a tutorial stage using only information Pants and GitHub Actions
  already produce (cache-hit flag, `pants --changed-since` output, `pants test` elapsed time) — no
  LLM call, no simulated/echoed build output.
- Reproduce a genuine cold-start experience for every new PR, independent of what `pants-ci.yml` has
  already cached on `main`.
- Keep the PR thread readable: one comment per PR that updates in place across pushes.
- Ship as a pure addition — zero modifications to `01`–`03` or `pants-ci.yml`.

**Non-Goals:**
- Not building a general-purpose bot framework or supporting arbitrary package additions beyond the
  two documented example paths (`internal/notifier`/`internal/formatter` as leaf, `internal/models` as
  common). Extending to more packages is future work.
- Not handling `pull_request` events from forks specially (see Risks) — out of scope for this change.
- Not replacing or deprecating the existing manual Step 1–3 workflows; Step 4 is additive, self-paced
  practice.

## Decisions

1. **PR-scoped cache key instead of the shared key.**
   Key: `pants-lmdb-tutorial-pr${{ github.event.pull_request.number }}-${{ github.sha }}`, with
   restore-keys prefix `pants-lmdb-tutorial-pr${{ github.event.pull_request.number }}-`.
   Alternative considered: reuse the shared `pants-lmdb-${{ runner.os }}-...` key used elsewhere.
   Rejected because it is kept warm by `pants-ci.yml` running on every `main` push, so a learner's
   first PR would almost never observe a genuine cache miss, breaking the Stage 1 narrative.

2. **Stage classification driven by two real signals, not a fixed embedded rulebook.**
   `steps.cache.outputs.cache-hit` decides cold-vs-warm; when warm, the *actual* target list from
   `pants --changed-since=<base_sha> --changed-dependents=transitive list` decides leaf-vs-common
   vs-other. Alternative considered: classify purely from `git diff --name-only` path substrings
   (closer to the user's original draft). Rejected in favor of using Pants' own target resolution,
   since it is the same signal the build itself uses and stays correct if BUILD structure changes.

3. **Sticky comment via hidden marker, not `createComment` per push.**
   Use `<!-- pant-mentor -->` as a hidden HTML comment marker; on each run, list existing comments via
   `actions/github-script`, find one containing the marker, and `updateComment` if found, else
   `createComment`. Alternative considered: always create a new comment (matches the user's draft).
   Rejected because `synchronize` fires on every push and would otherwise spam the PR timeline.

4. **Real `pants test` run scoped by stage, not simulated echo output.**
   Stage 1 runs `pants test ::` (full); Stage 2/3/other run
   `pants --changed-since=<base_sha> --changed-dependents=transitive test`. The comment's timing and
   "what rebuilt" data come directly from these real runs, matching the precedent already set by
   `02-full-build.yml`/`03-incremental.yml` (both use real Pants output, not fixtures).

5. **New workflow file (`04-pr-mentor.yml`) instead of extending `03-incremental.yml`.**
   `03` is deliberately a manual `workflow_dispatch` lesson with a `changed_package` **input** the
   learner picks. Converting it to a `pull_request` trigger would remove the manual-selection lesson
   entirely rather than add a new one, and would touch a workflow this change's proposal commits to
   leaving untouched.

## Risks / Trade-offs

- **[Risk]** PRs from forks run with `pull_request` (not `pull_request_target`), so the workflow has
  no access to secrets and runs the fork's own workflow file — safe by default, but also means a
  fork's *modified* `04-pr-mentor.yml` governs its own run. → **Mitigation**: acceptable for a
  learning repo where forks are expected to run their own copy of the tutorial; no secrets are used by
  this workflow (`pull-requests: write`, `contents: read` only), so there is nothing sensitive to leak.
- **[Risk]** PR-scoped cache entries accumulate (one per PR number) and are never explicitly cleaned
  up. → **Mitigation**: GitHub Actions cache has its own repo-wide 10 GB eviction/LRU policy; no
  action needed for a teaching repo's PR volume.
- **[Risk]** A learner edits neither `internal/notifier`/`internal/formatter` nor `internal/models`
  (e.g., only touches `cmd/` or docs) → classified as `other`. → **Mitigation**: the `other` stage
  comment explicitly nudges the learner toward the two known example files instead of failing silently.
- **[Trade-off]** Classifying via `pants --changed-since ... list` output (Decision 2) requires Pants
  to already be installed and runnable before classification, making the detect step slightly heavier
  than a plain `git diff` grep — accepted because correctness (matching real build behavior) matters
  more than shaving a few seconds off a teaching workflow.

## Migration Plan

Pure addition — no migration. Rollback is deleting `.github/workflows/04-pr-mentor.yml` and reverting
the README section; no other workflow, secret, or spec is touched.

## Open Questions

- None blocking implementation. Naming of the workflow file (`04-pr-mentor.yml`) follows the existing
  `NN-*` convention from `01`–`03`; confirm this is still desired once a learner has run through
  Steps 1–3, since Step 4 is automatic rather than a manual dispatch like its numbered siblings.
