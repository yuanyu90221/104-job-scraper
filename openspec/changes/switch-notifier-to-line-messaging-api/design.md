## Context

`internal/notifier.LineNotifier` posts to LINE Notify (`https://notify-api.line.me/api/notify`), a service LINE officially retired 2025-03-31. The daily-scrape CI workflow's notification step now fails on every run with a DNS resolution error, unrelated to the scraper itself (which was separately validated at parity under `rewrite-client-go-rod`). The sibling project `~/egg-village-platform/linebot` already integrates with LINE's still-supported Messaging API via `github.com/line/line-bot-sdk-go/v7/linebot`, giving a proven, in-house reference for the client construction and message-sending pattern to follow.

Unlike `egg-village-platform/linebot` (a webhook-driven chat bot that replies to inbound events), this project is a one-shot batch CLI with no inbound webhook — it only ever needs to *push* a message to one fixed destination once per run. The user has decided:
- The push target (a LINE `userId` or `groupId`) will be injected via config/flag rather than hardcoded.
- The Messaging API channel secret/access token will be reused from the existing `egg-village-platform` LINE Bot channel rather than provisioning a new LINE channel.

## Goals / Non-Goals

**Goals:**
- Replace the retired LINE Notify HTTP call with a `PushMessage` call via the official `linebot` SDK, mirroring `egg-village-platform/linebot`'s client-construction style.
- Keep `LineNotifier.Send(jobs, keyword, topN) error` as the boundary the CLI (`cmd/main.go`) calls, so the rest of the pipeline (search → format → notify) doesn't change shape.
- Make the new `LineNotifier` unit-testable without hitting LINE's real API, using `linebot.WithEndpointBase` pointed at an `httptest.Server`, following the same testing shape LINE's own SDK test suite (`send_message_test.go`) uses.

**Non-Goals:**
- Building any webhook/reply handling — this project has no inbound LINE events, only outbound push.
- Supporting multiple simultaneous push targets (multicast/broadcast) — out of scope; a single fixed target is what the user asked for.
- Provisioning the actual LINE channel or GitHub secret values — those are operational steps for the user to perform after this change merges, not something this change can do.

## Decisions

- **Use `client.PushMessage(targetID, linebot.NewTextMessage(text)).Do()`** rather than hand-rolling the HTTP POST (as the old `LineNotifier.post` did for LINE Notify). The SDK handles endpoint construction, JSON encoding, auth header, and error decoding (`APIError`), removing hand-maintained duplicate logic and matching the pattern already proven in `egg-village-platform/linebot`.
- **`linebot.New(channelSecret, channelToken, options...)` requires a non-empty `channelSecret`** even though push-only usage never verifies webhook signatures with it (confirmed by reading the pinned SDK source, `client.go`'s `New`). `NewLine` therefore takes and requires a channel secret as a parameter, matching `egg-village-platform`'s own `config.Config` fields (`LineChannelSecret`, `LineChannelAccessToken`) rather than inventing a placeholder value — this keeps the two projects' credential shapes consistent if they ever reuse the same channel's secret.
- **`NewLine` signature: `NewLine(channelSecret, channelToken, targetID string, options ...linebot.ClientOption) (*LineNotifier, error)`.** The variadic `linebot.ClientOption` parameter exists solely so tests can pass `linebot.WithEndpointBase(testServer.URL)`; production callers (`cmd/main.go`) pass none, defaulting to the real `https://api.line.me`.
- **Raise the truncation limit from ~1000 to 5000 characters**, matching the LINE Messaging API's text message length limit (LINE Notify's limit was ~1000; Messaging API's is 5000) — `buildMessage`'s output was previously being cut short well below what the new channel actually allows.
- **CLI flags**: replace the single `--line-token` with `--line-channel-secret`, `--line-channel-token`, `--line-target-id` (three separate LINE Messaging API credentials, since the Messaging API separates "prove you're a valid channel" (secret) from "prove you're authorized to call the API" (access token) from "who to send to" (target ID) — LINE Notify collapsed all of this into one per-user token). `--line-top` is unchanged. Sending is gated on `--line-channel-token != ""` (matching the old gate on `--line-token != ""`); if that's set but `--line-target-id` is empty, `run()` returns a config error immediately rather than letting the SDK fail with a confusing "invalid target" API error later.
- **GitHub Actions secrets**: rename `LINE_NOTIFY_TOKEN` to `LINE_CHANNEL_SECRET` / `LINE_CHANNEL_ACCESS_TOKEN` / `LINE_TARGET_ID` in `daily-scrape.yml`. The old secret becomes unused (left in place; removing repo secrets is an operational action outside this change's scope).
- **`--line-channel-secret`/`--line-channel-token`/`--line-target-id` default to `os.Getenv("LINE_CHANNEL_SECRET"/"LINE_CHANNEL_ACCESS_TOKEN"/"LINE_TARGET_ID")`** rather than a literal `""` default, so credentials can be supplied via environment (a sourced `.env` locally, or an `env:` block in CI) without appearing as literal flag values in shell history or workflow YAML. An explicitly-passed flag still wins over the environment variable (`pflag`'s normal precedence — the flag's value is only read from the env-derived default when the flag isn't set on the command line). `daily-scrape.yml` moves the three secrets from explicit `--line-*` flags into the run step's `env:` block accordingly.

## Risks / Trade-offs

- [Risk] The reused `egg-village-platform` LINE Bot channel is a chat-bot channel with its own followers/groups; pushing unrelated job-scraper messages to it depends entirely on the operator picking a `targetID` (a group or the operator's own user ID) that makes sense for that channel's actual audience. → Mitigation: this is an operational/config decision the user already made explicitly (reuse the channel, decide the target ID later via secrets) — not something this change can validate at the code level. No code-level mitigation needed.
- [Risk] `linebot.New` requires a non-empty channel secret merely to satisfy a constructor check that's actually irrelevant to push-only usage; a future reader might assume the secret is used for request signing on pushes when it isn't. → Mitigation: a doc comment on `NewLine` notes explicitly that the secret is unused for push, kept only because the SDK constructor requires it.
- [Trade-off] This is a **BREAKING** CLI/config change (flag names, secret names) with no backward-compatible fallback (e.g., no dual-support for the old `--line-token`). Given LINE Notify is already permanently dead, there is no working old behavior left to preserve, so a compatibility shim would add complexity for no benefit.

## Migration Plan

1. Implement and land the new `LineNotifier`/CLI flags/workflow changes (this change).
2. Operator (user) provisions/reuses the LINE Messaging API channel and sets the three new GitHub Actions secrets (`LINE_CHANNEL_SECRET`, `LINE_CHANNEL_ACCESS_TOKEN`, `LINE_TARGET_ID`).
3. Dispatch `daily-scrape.yml` once manually to confirm the push succeeds end-to-end against the real LINE Messaging API.
4. Once confirmed, the old `LINE_NOTIFY_TOKEN` secret can be deleted from the repo (operational cleanup, not part of this change's tasks).

## Open Questions

- Should the old `LINE_NOTIFY_TOKEN` GitHub secret be deleted as part of this change's tasks, or left for the user to clean up manually later? (Currently: left for later, since deleting repo secrets is a repo-admin action outside a code change's normal scope.)
