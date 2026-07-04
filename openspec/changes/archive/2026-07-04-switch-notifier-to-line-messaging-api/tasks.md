## 1. Preparation

- [x] 1.1 Add `github.com/line/line-bot-sdk-go/v7` to `go.mod`/`go.sum` (already fetched into the local module cache at v7.21.0 during design spike; pin that version) — done as a side effect of the design spike (`go get github.com/line/line-bot-sdk-go/v7@v7.21.0`)
- [x] 1.2 Confirm `linebot.New`, `Client.PushMessage`, `NewTextMessage`, `WithEndpointBase`, `WithHTTPClient` signatures against the pinned version's source (already read during design; re-verify no drift before coding) — confirmed via `go doc` and reading `client.go`/`send_message.go`/`send_message_test.go` in the pinned module cache; no drift, `linebot.New` requires non-empty channelSecret/channelToken

## 2. TDD: LineNotifier rewrite (red -> green)

- [x] 2.1 Replace `line_test.go`'s `TestSend_Success` (which reaches into `LineNotifier`'s private fields against the old LINE Notify shape) and add new failing tests against an `httptest.Server` wired via `linebot.WithEndpointBase`: assert the push request hits `POST /v2/bot/message/push`, carries `Authorization: Bearer <channelToken>`, and its JSON body's `to` field matches the configured target ID and `messages[0].text` contains the expected job/keyword content
- [x] 2.2 Add a failing test asserting `Send` returns a non-nil error when the test server responds with a non-2xx status
- [x] 2.3 Add failing tests asserting `NewLine` returns an error (not a notifier) when `channelSecret` or `channelToken` is empty
- [x] 2.4 Implement the new `LineNotifier` struct (`*linebot.Client` + `targetID`) and `NewLine(channelSecret, channelToken, targetID string, options ...linebot.ClientOption) (*LineNotifier, error)` to turn the above tests green, keeping `buildMessage`/`salaryDesc`/`formatDate` unchanged
- [x] 2.5 Update the message-truncation constant from ~1000 to 5000 characters (LINE Messaging API's text message limit) in the code path that truncates before sending — added `maxMessageLen = 5000` const
- [x] 2.6 Run `go test ./internal/notifier/...` until green, including the pre-existing `TestBuildMessage_ContainsJobName`/`TestBuildMessage_TruncatesLongMessages` (unaffected by the rewrite, but must still pass) — all 6 tests pass

## 3. CLI wiring

- [x] 3.1 Replace `cmd/main.go`'s `--line-token` flag with `--line-channel-secret`, `--line-channel-token`, `--line-target-id` (keep `--line-top` as-is)
- [x] 3.2 Update `run()`'s gating logic: send only when `--line-channel-token` is non-empty; if it's set but `--line-target-id` is empty, return a config error immediately instead of calling `NewLine`/`Send`
- [x] 3.3 Update the command's help text/examples referencing the old `--line-token=<TOKEN>` usage

## 3a. CLI env-var fallback for credentials (TDD, red -> green)

- [x] 3a.1 Add failing tests in `cmd/main_test.go` asserting `newRootCmd()`'s `--line-channel-secret`/`--line-channel-token`/`--line-target-id` flags default to `os.Getenv("LINE_CHANNEL_SECRET"/"LINE_CHANNEL_ACCESS_TOKEN"/"LINE_TARGET_ID")` when set, default to `""` when unset, and that an explicitly-set flag still overrides the environment variable
- [x] 3a.2 Implement the change in `cmd/main.go`: replace the three `""` flag defaults with `os.Getenv(...)` calls
- [x] 3a.3 Run `go test ./cmd/...` until green; re-run `go build ./...`, `go vet ./...`, `go test ./...` repo-wide to confirm no regressions
- [x] 3a.4 Update `.github/workflows/daily-scrape.yml` to pass `LINE_CHANNEL_SECRET`/`LINE_CHANNEL_ACCESS_TOKEN`/`LINE_TARGET_ID` via the run step's `env:` block instead of explicit `--line-*` flags
- [x] 3a.5 Update `README.md`'s flag table and add a `set -a && source .env && set +a` example for local manual runs

## 4. CI cutover

- [x] 4.1 Update `.github/workflows/daily-scrape.yml` to pass `--line-channel-secret`, `--line-channel-token`, `--line-target-id` sourced from new secrets `LINE_CHANNEL_SECRET`, `LINE_CHANNEL_ACCESS_TOKEN`, `LINE_TARGET_ID`, removing the `--line-token="${{ secrets.LINE_NOTIFY_TOKEN }}"` line
- [x] 4.2 Update `README.md`'s LINE Notify references (flag table, feature description, workflow description) to describe the new flags/Messaging API flow

## 5. Validation

- [x] 5.1 Confirm `go build ./...`, `go vet ./...`, and `go test ./...` succeed repo-wide after the rewrite — all pass (client 47.154s, search 9.839s, notifier/formatter instant)
- [x] 5.2 Manually run the CLI once with real Messaging API credentials (channel secret/token reused from the `egg-village-platform` LINE Bot channel per the user's decision) and a real target ID, confirming a job summary message actually arrives in LINE — ran `./bin/104-job-scraper --months=1 --pages=1 --line-top=5` with `.env` credentials sourced into the environment; found 22 jobs, pushed top 5, user confirmed receipt in LINE
- [x] 5.3 Once GitHub secrets `LINE_CHANNEL_SECRET`/`LINE_CHANNEL_ACCESS_TOKEN`/`LINE_TARGET_ID` are set by the user, dispatch `daily-scrape.yml` once to confirm the full pipeline (scrape → notify) succeeds end-to-end in CI — first dispatch (run 28717872689) failed because the commit hadn't been pushed to `origin/main` yet, so CI ran the old LINE-Notify code; after `git push origin main`, re-dispatched (run 28717936439) and it completed with all steps `success`

## 6. Follow-up

- [x] 6.1 Archive this change via OpenSpec once merged and validated, promoting `job-notification` into `openspec/specs/`
- [x] 6.2 Note the now-unused `LINE_NOTIFY_TOKEN` GitHub secret for the user to delete manually (repo-admin action, out of scope for this change) — flagged to the user; deletion is a manual repo-admin action left for them
