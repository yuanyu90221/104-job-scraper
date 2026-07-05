## ADDED Requirements

### Requirement: PR-scoped Pants cache isolation
The workflow SHALL restore/save the Pants LMDB store using a cache key scoped to the pull request
number, distinct from the shared cache key used by `01-pants-list.yml`, `03-incremental.yml`, and
`pants-ci.yml`, so that a PR's first run is not influenced by cache state warmed by other workflows.

#### Scenario: First run on a brand-new PR is a genuine cache miss
- **WHEN** a pull request is opened for the first time and no prior cache entry exists under the key
  prefix `pants-lmdb-tutorial-pr<PR number>-`
- **THEN** the cache restore step reports `cache-hit` as not `true`, and the workflow proceeds to run
  a full `pants test ::` build

#### Scenario: Subsequent push in the same PR reuses that PR's own cache
- **WHEN** a second commit is pushed to a pull request that already produced a cache entry under
  `pants-lmdb-tutorial-pr<PR number>-`
- **THEN** the cache restore step matches that PR-scoped entry via `restore-keys` and populates the
  LMDB store from the prior push, measurably speeding up that push's `pants test` run (even though
  `cache-hit` itself reports `false`, since the key's `github.sha` component differs from the prior
  push's commit)

### Requirement: Stage classification from real Pants signals
The workflow SHALL classify each run into exactly one stage — `stage_1` (cold full build), `stage_2`
(leaf package changed), `stage_3` (common package changed), or `other` — using only the real
`github.event.action` (`opened` vs `synchronize`) and the real output of
`pants --changed-since=<base_sha> --changed-dependents=transitive list`. The workflow SHALL NOT use a
simulated/hardcoded build log or call an external LLM to perform this classification. The workflow
SHALL NOT use `actions/cache`'s `cache-hit` output as the cold/warm signal, since that output only
reports `true` on an exact primary-key match and the cache key embeds `github.sha`, making it `false`
on every push after the first regardless of whether the LMDB store was actually warm.

#### Scenario: PR-opened event always yields Stage 1 regardless of changed files
- **WHEN** the triggering event's `action` is `opened`
- **THEN** the workflow selects `stage_1`, independent of which files changed in the PR

#### Scenario: Leaf package change yields Stage 2
- **WHEN** the triggering event's `action` is `synchronize` and the changed-target list includes
  `//internal/notifier` or `//internal/formatter` but not `//internal/models`
- **THEN** the workflow selects `stage_2`

#### Scenario: Common package change yields Stage 3
- **WHEN** the triggering event's `action` is `synchronize` and the changed-target list includes
  `//internal/models`
- **THEN** the workflow selects `stage_3` (even if leaf packages also appear in the same list)

#### Scenario: Unrecognized change yields the fallback stage
- **WHEN** the triggering event's `action` is `synchronize` and the changed-target list contains
  neither `//internal/models` nor `//internal/notifier`/`//internal/formatter` (e.g., only `//cmd` or
  `//internal/client` changed)
- **THEN** the workflow selects `other` and the resulting comment names `internal/notifier` and
  `internal/models` as the suggested files to edit next

### Requirement: Real build execution backs the reported data
The workflow SHALL derive the elapsed-time figure and the list of rebuilt/affected targets shown in
the PR comment from an actual `pants test` invocation for that run (`pants test ::` for `stage_1`;
`pants --changed-since=<base_sha> --changed-dependents=transitive test` otherwise), not from
hardcoded or simulated values.

#### Scenario: Reported duration matches the actual measured run
- **WHEN** the mentor comment is generated for any stage
- **THEN** the elapsed-seconds value in the comment equals the measured wall-clock time of that run's
  `pants test` step (start/end timestamps captured around the real invocation)

### Requirement: Single sticky comment per pull request
The workflow SHALL maintain at most one mentor comment per pull request, identified by a hidden
marker (`<!-- pant-mentor -->`), updating it in place on subsequent runs rather than creating
additional comments.

#### Scenario: First run creates the comment
- **WHEN** the workflow runs on a pull request that has no existing comment containing the
  `<!-- pant-mentor -->` marker
- **THEN** it creates exactly one new comment containing that marker

#### Scenario: Later runs update the same comment
- **WHEN** the workflow runs again on a pull request that already has a comment containing the
  `<!-- pant-mentor -->` marker
- **THEN** it updates that existing comment's body and does not create a second comment

### Requirement: No modification of existing teaching workflows
Adding this capability SHALL NOT change the trigger, steps, or behavior of
`01-pants-list.yml`, `02-full-build.yml`, `03-incremental.yml`, or `pants-ci.yml`.

#### Scenario: Existing workflows remain byte-for-byte unchanged
- **WHEN** this change is applied
- **THEN** a diff of `01-pants-list.yml`, `02-full-build.yml`, `03-incremental.yml`, and
  `pants-ci.yml` against their pre-change versions shows no differences
