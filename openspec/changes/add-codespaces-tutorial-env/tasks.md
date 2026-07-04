## 1. Devcontainer

- [x] 1.1 Create `.devcontainer/devcontainer.json` using the `mcr.microsoft.com/devcontainers/go:1-1.25` image
- [x] 1.2 Add `postCreateCommand` that installs Pants (`get-pants.sh`) and runs `scripts/patch-pants-go.sh`
- [ ] 1.3 Open a Codespace from this branch and verify `pants list ::`, `pants dependencies internal/search::`, and `pants --changed-since=HEAD~1 --changed-dependents=transitive list` all run without manual setup

## 2. Documentation

- [x] 2.1 Replace the manual copy-paste snippet in README.md's "用 GitHub Codespaces 建立互動式練習環境" section with instructions to just open a Codespace (setup is now automatic)
- [x] 2.2 Note in README that the same `pants` commands mirror what the teaching workflows (`01-pants-list.yml`, `03-incremental.yml`) run in CI
