## 1. Workflow scaffold

- [x] 1.1 Create `.github/workflows/04-pr-mentor.yml` with `pull_request` trigger
      (`branches: [main]`, `types: [opened, synchronize]`) and
      `permissions: { pull-requests: write, contents: read }`.
- [x] 1.2 Add checkout (`fetch-depth: 0`), `actions/setup-go@v5` (`go-version: '1.25'`), Pants
      install (`get-pants.sh`), and the `scripts/patch-pants-go.sh` step, matching the pattern in
      `03-incremental.yml`.

## 2. PR-scoped cache

- [x] 2.1 Add the `actions/cache@v4` step with `id: cache`, path `~/.cache/pants/lmdb_store`, key
      `pants-lmdb-tutorial-pr${{ github.event.pull_request.number }}-${{ github.sha }}`, restore-keys
      `pants-lmdb-tutorial-pr${{ github.event.pull_request.number }}-`.
- [ ] 2.2 Verify (via a scratch PR) that a brand-new PR number produces `cache-hit != 'true'` on its
      first run, satisfying the "PR-scoped Pants cache isolation" spec's first scenario.

## 3. Stage detection

- [x] 3.1 Add a `detect` step that captures `pants --changed-since=<base_sha>
      --changed-dependents=transitive list` output via `$GITHUB_OUTPUT` (multi-line-safe heredoc).
- [x] 3.2 Implement the classification rule in shell: `cache-hit != 'true'` → `stage_1`; else
      contains `//internal/models` → `stage_3`; else contains `//internal/notifier` or
      `//internal/formatter` → `stage_2`; else → `other`. Expose as `steps.detect.outputs.stage`.
- [x] 3.3 Add unit-style verification: for each of the 4 branches, manually run the shell classifier
      logic locally against fixture `CHANGED`/`cache-hit` values and confirm the expected stage.

## 4. Real build execution

- [x] 4.1 Record `START=$(date +%s)` before the build step.
- [x] 4.2 Run `pants test ::` when `stage == stage_1`, else run
      `pants --changed-since=<base_sha> --changed-dependents=transitive test`.
- [x] 4.3 Compute `ELAPSED=$(( $(date +%s) - START ))` and expose both `ELAPSED` and the `detect`
      step's changed-target list to the comment-generation step via `$GITHUB_ENV`/step outputs.

## 5. Sticky mentor comment

- [x] 5.1 Add an `actions/github-script@v7` step that lists existing comments on the PR via
      `github.rest.issues.listComments`, searching for one containing the `<!-- pant-mentor -->`
      marker.
- [x] 5.2 Build the comment body per stage (`stage_1`/`stage_2`/`stage_3`/`other`), reusing the
      emoji-header + 📊 build-report + 💡 mentor-insight + next-challenge structure from the approved
      plan, with real `ELAPSED` seconds and real changed-target list substituted in; embed the
      `<!-- pant-mentor -->` marker in the body.
- [x] 5.3 Call `updateComment` if a marked comment was found, else `createComment`.
- [ ] 5.4 Verify via a scratch PR with two pushes that only one comment exists after both runs and its
      content reflects the second run's stage.

## 6. Documentation

- [x] 6.1 Update `README.md`'s "用這個專案學習 Pants Build 的 GitHub Action" section to describe Step
      4 (open a PR, edit `internal/notifier/line.go` or `internal/models/job.go`, read the bot
      comment) alongside the existing manual Step 1–3 description.
- [x] 6.2 Update the "自己做實驗" bullet (~README.md line 83) to mention Step 4 as an alternative to
      manually re-running Step 3.

## 7. End-to-end verification

- [ ] 7.1 Open a scratch PR touching an unrelated/no file (or trivial doc change) → confirm the
      mentor comment shows Stage 1 (cold, full build).
- [ ] 7.2 Push a commit to that PR changing only `internal/notifier/line.go` (add the debug
      `fmt.Fprintf` line from the plan) → confirm the comment updates to Stage 2 and lists
      `models`/`client`/`formatter`/`search` as cache-hit.
- [ ] 7.3 Push a further commit changing only `internal/models/job.go` (add the `String()` method from
      the plan) → confirm the comment updates to Stage 3 and lists all dependent packages as rebuilt.
- [ ] 7.4 Confirm exactly one bot comment existed throughout steps 7.1–7.3 (updated in place, not
      recreated).
- [x] 7.5 Diff `01-pants-list.yml`, `02-full-build.yml`, `03-incremental.yml`, `pants-ci.yml` against
      their pre-change versions and confirm zero changes.
- [ ] 7.6 Close/clean up the scratch PR and branch used for 7.1–7.5.
