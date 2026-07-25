## ADDED Requirements

### Requirement: Codespace provisions the Pants toolchain automatically
The devcontainer SHALL install Go, install Pants, and apply the Go 1.24+ `GOEXPERIMENT` patch (`scripts/patch-pants-go.sh`) automatically when a Codespace is created, so that Pants commands work without any manual setup step.

#### Scenario: Opening a fresh Codespace
- **WHEN** a learner opens this repository in a new GitHub Codespace
- **THEN** the devcontainer's `postCreateCommand` installs Pants and runs `scripts/patch-pants-go.sh` before the terminal is handed to the learner
- **AND** running `pants list ::` in that terminal succeeds without any additional installation step

#### Scenario: Re-running setup manually
- **WHEN** a learner re-runs the same install + patch commands by hand inside an already-provisioned Codespace
- **THEN** the commands complete without error, because `scripts/patch-pants-go.sh` is idempotent

### Requirement: README documents the Codespaces workflow
The project README SHALL describe how to open the repository in a GitHub Codespace and which `pants` commands to try first, so a learner does not need to read the workflow YAML files to get started interactively.

#### Scenario: Learner follows the README Codespaces section
- **WHEN** a learner reads the "在 GitHub Codespaces 練習" section of `README.md`
- **THEN** they find the steps to open a Codespace and a short list of `pants` commands (`pants list ::`, `pants dependencies`, `pants --changed-since=...`) matching what the teaching workflows run in CI
