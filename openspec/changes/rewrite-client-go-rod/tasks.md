## 1. Preparation

- [x] 1.1 Add `github.com/go-rod/rod` and `github.com/go-rod/stealth` to `go.mod`/`go.sum` (pin latest tagged versions)
- [x] 1.2 Confirm the exact go-rod/go-rod-stealth API surface referenced in `design.md`'s mapping table (`launcher.New`, `launcher.LookPath`, `rod.New().ControlURL(...).Connect()`, `page.EachEvent`, `proto.NetworkResponseReceived`, `proto.NetworkGetResponseBody`, `stealth.Page`) against the pinned version's godoc/source, and note any signature differences before writing code

## 2. TDD: pure launch-config function (red -> green)

- [x] 2.1 Write a failing test replacing `TestChromiumLaunchOptions`, asserting on the new pure, browser-free launch-config function's output (default channel vs. `"chrome"` channel, stealth flags present) per `design.md`'s "Testable launch configuration without a real browser"
- [x] 2.2 Implement the pure launch-config function to green
- [x] 2.3 Implement the thin function that turns that plain config into a real `*launcher.Launcher` (flags via `.Set`, binary via `.Bin`) — not unit-tested in isolation, only exercised via the integration tests in section 4

## 3. TDD: Client lifecycle (New/Close) rewrite

- [x] 3.1 Change `Client`'s fields from `playwright.*` types to go-rod equivalents (`*launcher.Launcher`, `*rod.Browser`) — this intentionally breaks compilation of `New`, `Close`, and the three integration tests (red)
- [x] 3.2 Implement `New()`: resolve the launch config (section 2), launch via the launcher, connect via `rod.New().ControlURL(...).Connect()`, and open one `stealth.Page(browser)` to smoke-test the pipeline; get the package compiling again (integration tests may still be red/skipped until section 4 rewires `Search`)
- [x] 3.3 Implement `Close()`: close page/browser and clean up the launcher process, returning the package to a buildable state

## 4. TDD: Search response capture rewrite

- [x] 4.1 Confirm `TestSearch_DetectsCloudflareChallenge`, `TestSearch_IgnoresUnrelatedCloudflare403`, `TestSearch_ReturnsJobsOnSuccess` are left assertion-unchanged (per `design.md`, they're black-box against the public API) — run them now and confirm red (compile or behavior failure) before touching `Search`
- [x] 4.2 Rewire `Search`'s response listener from `page.On("response", ...)` to `page.EachEvent(func(e *proto.NetworkResponseReceived) bool { ... })`, preserving the existing `isRelevant`/`respCh`/`errCh`/60s-timeout logic unchanged. Also required a second `*proto.NetworkLoadingFinished` callback registered on the same `EachEvent` call: `Network.getResponseBody` fails with "No data found for resource with given identifier" if fetched before loading finishes, so relevant responses are now tracked by request ID and their body is only fetched once `NetworkLoadingFinished` fires for that ID — this was the actual cause of the 60s timeouts seen during development, not a dispatcher deadlock as first suspected.
- [x] 4.3 Rewire status/headers/URL/body access to `e.Response.Status`/`e.Response.Headers`/`e.Response.URL`/`proto.NetworkGetResponseBody{RequestID: e.RequestID}.Call(page)` per `design.md`'s mapping table. Also required `page.EnableDomain(&proto.NetworkEnable{})` before events would fire at all (go-rod, unlike Playwright, doesn't auto-enable CDP domains), and moving the body-fetch `.Call(page)` into a goroutine off the `EachEvent` callback (a blocking CDP call made synchronously inside the callback stalls the single-threaded event dispatcher).
- [x] 4.4 Apply per-page user-agent/locale/timezone/viewport setup on every `stealth.Page(browser)` call (replacing the old once-per-context setup), per `design.md`'s "No BrowserContext equivalent" decision
- [x] 4.5 Run `go test ./internal/client/...` until all pure-function and integration tests are green — all pass locally with a real (auto-downloaded) browser, no skips

## 5. Validation

- [x] 5.1 Manually run the CLI once against production `104.com.tw` with the go-rod build (same spike pattern as the prior Playwright channel A/B) and record success/Cloudflare-blocked for comparison against the current playwright-go baseline — ran three times against the live site: default (auto-resolved) channel with 1 page (22 jobs), `SCRAPER_BROWSER_CHANNEL=chrome` with 1 page (22 jobs), and default channel with 3 pages (63 jobs). All three succeeded with no Cloudflare block, at parity with (or better than) the playwright-go baseline.
- [x] 5.2 Confirm `go build ./...` and `go vet ./...` succeed repo-wide, including `cmd/main.go` and `cmd/debug_playwright/main.go` (if still present at this point)

## 6. Cutover

- [ ] 6.1 Update `.github/workflows/daily-scrape.yml`'s browser-install step for go-rod's requirements (rely on auto-download, or install/point at system Chrome), per `design.md`'s Migration Plan and Open Questions
- [ ] 6.2 Retire the `browser_channel` `workflow_dispatch` input (superseded by go-rod's channel handling; the Playwright-fingerprint A/B question it existed for is already answered)
- [ ] 6.3 Replace `cmd/debug_playwright/main.go` with a go-rod equivalent, or delete it if no longer needed
- [ ] 6.4 Remove `github.com/mxschmitt/playwright-go` from `go.mod`/`go.sum` (`go mod tidy`)
- [ ] 6.5 Dispatch `daily-scrape.yml` once on the real GitHub-hosted runner to confirm the go-rod build behaves at parity with the playwright-go baseline (pass/fail parity is success — this spike is not expected to fix IP-reputation blocking)

## 7. Follow-up

- [ ] 7.1 Record this spike's outcome in `investigate-104-bot-detection-alternatives` (tasks 3.2/4.1/4.2) so its decision handoff reflects go-rod's actual result
- [ ] 7.2 Archive this change via OpenSpec once merged and validated, promoting `job-search-scraping` into `openspec/specs/`
