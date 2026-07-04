## Why

LINE officially retired the LINE Notify service on 2025-03-31. `internal/notifier`'s `LineNotifier` posts to `https://notify-api.line.me/api/notify`, which no longer resolves (confirmed via a live `daily-scrape.yml` CI failure and independent local DNS lookups) — every daily run's notification step now fails permanently, regardless of the scraper's own success. This must be replaced with LINE's supported channel, the Messaging API's push message endpoint, following the same SDK (`github.com/line/line-bot-sdk-go/v7/linebot`) already used in the sibling `egg-village-platform/linebot` project.

## What Changes

- Replace `internal/notifier.LineNotifier`'s hand-rolled HTTP POST to the retired LINE Notify endpoint with a `*linebot.Client`-backed push message to a fixed target (a LINE `userId` or `groupId`).
- **BREAKING**: `notifier.NewLine`'s signature changes from a single LINE Notify token to a channel secret, channel access token, and target ID (three LINE Messaging API credentials instead of one LINE Notify token).
- **BREAKING**: `cmd/main.go`'s `--line-token` CLI flag is replaced by `--line-channel-secret`, `--line-channel-token`, and `--line-target-id`. `--line-top` is unchanged.
- Update `.github/workflows/daily-scrape.yml` to pass the new three secrets (`LINE_CHANNEL_SECRET`, `LINE_CHANNEL_ACCESS_TOKEN`, `LINE_TARGET_ID`) instead of `LINE_NOTIFY_TOKEN`.
- Raise the outgoing message truncation limit from LINE Notify's ~1000 chars to the Messaging API text message limit (5000 chars).
- Update `README.md`'s LINE Notify references to describe the Messaging API push flow instead.

## Capabilities

### New Capabilities
- `job-notification`: delivering the daily job summary to LINE via the Messaging API's push message. (No `openspec/specs/job-notification/` exists yet — the prior LINE Notify behavior was never formally spec'd — so this is captured as a new capability rather than a delta.)

### Modified Capabilities
(none)

## Impact

- `internal/notifier/line.go`, `internal/notifier/line_test.go` — full rewrite of the LINE delivery mechanism.
- `cmd/main.go` — CLI flag surface changes (breaking for any existing script/automation passing `--line-token`).
- `go.mod`/`go.sum` — adds `github.com/line/line-bot-sdk-go/v7` as a new direct dependency.
- `.github/workflows/daily-scrape.yml` — new required secrets (`LINE_CHANNEL_SECRET`, `LINE_CHANNEL_ACCESS_TOKEN`, `LINE_TARGET_ID`); the existing `LINE_NOTIFY_TOKEN` secret becomes unused and can eventually be removed from the repo's secrets.
- `README.md` — usage docs for the new flags/env.
