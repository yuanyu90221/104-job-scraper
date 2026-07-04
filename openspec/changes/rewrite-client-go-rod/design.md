## Context

`internal/client/client.go` currently drives a headless browser via `github.com/mxschmitt/playwright-go` to get past Cloudflare's managed challenge on `www.104.com.tw` and capture the search page's own JSON API call. `investigate-104-bot-detection-alternatives` established (on the real GitHub-hosted CI runner, not just locally) that browser channel/fingerprint (bundled Chromium vs. real Chrome) does not change pass/fail — both are blocked identically on the same shared IP. The user has chosen to spend a time-boxed spike ruling out `go-rod` + `go-rod/stealth` (a Go-native alternative to playwright-go) as the automation engine, independent of that IP-reputation finding, before deciding what (if anything) to do about IP reputation itself.

The existing implementation has three moving parts that this design must re-home onto go-rod:
1. Browser/context lifecycle (`New`, `Close`) — launch, stealth init, and a `SCRAPER_BROWSER_CHANNEL` env var to force bundled-Chromium vs. real-Chrome for A/B testing.
2. Response interception (`Search`'s `p.On("response", ...)` handler) — inspect every response on the page, extract the search API's JSON body, and separately flag a Cloudflare-challenge response as `ErrCloudflareChallenge`, while ignoring unrelated subresource 403s (a real bug fixed in the current code — must not regress).
3. Pure helpers with no browser dependency (`isCloudflareChallenge`, `buildURL`) — untouched by this rewrite.

`internal/client/client_test.go` is the existing regression suite: three browser-integration tests (`TestSearch_DetectsCloudflareChallenge`, `TestSearch_IgnoresUnrelatedCloudflare403`, `TestSearch_ReturnsJobsOnSuccess`) built on `httptest.NewServer` + `New()`/`Search()`/`c.baseURL`, plus pure-function tests (`TestChromiumLaunchOptions`, `TestBuildURL`, `TestIsCloudflareChallenge`). The three integration tests are black-box against the public API and are the main parity guardrail for this rewrite; they should port with no behavioral change to their assertions, only to what's exercised underneath.

## Goals / Non-Goals

**Goals:**
- Replace playwright-go with go-rod + go-rod/stealth as the only automation engine in `internal/client`, with `New`, `Client.Search`, `Client.Close` unchanged in signature and observable behavior.
- Preserve exactly: JSON API response capture, `ErrCloudflareChallenge` detection, the unrelated-403 false-positive guard, and the 60s search timeout.
- Preserve the `SCRAPER_BROWSER_CHANNEL` A/B knob's intent, translated to go-rod's launcher model.
- Follow TDD: the three black-box integration tests and the pure-function tests are the acceptance bar; new/adjusted tests are written red before implementation.

**Non-Goals:**
- Solving IP reputation / Cloudflare blocking itself. This spike is judged on *parity with the current playwright-go implementation*, not on whether it clears Cloudflare more often — `investigate-104-bot-detection-alternatives` already found evidence that the blocker is IP-reputation, not engine/fingerprint. A go-rod version that is blocked exactly as often as playwright-go is still a successful, landable rewrite.
- Changing `cmd/main.go`, CLI flags, output format, or the LINE notification path.
- Introducing a persistent multi-page "browser context" abstraction if go-rod's model doesn't need one — see Decisions.

## Decisions

### API mapping: playwright-go → go-rod/go-rod-stealth

| Concern | playwright-go today | go-rod replacement |
|---|---|---|
| Engine bootstrap | `playwright.Run()` | `launcher.New()...Launch()` → control URL string |
| Browser connect | `pw.Chromium.Launch(opts)` | `rod.New().ControlURL(controlURL).Connect()` |
| Launch flags | `BrowserTypeLaunchOptions.Args` | `launcher.Set(flagName, values...)` per flag |
| Channel selection | `BrowserTypeLaunchOptions.Channel` | `launcher.Bin(path)`, path resolved via `launcher.LookPath()` when `SCRAPER_BROWSER_CHANNEL=chrome`; unset uses go-rod's own resolution/auto-download |
| Isolated session | `browser.NewContext(opts)` | No direct equivalent is required — see "No BrowserContext equivalent" below |
| Stealth init script | `ctx.AddInitScript(...)` (4 manual JS patches) | `stealth.Page(browser)` from `go-rod/stealth`, which injects a broader, maintained evasion script (webdriver, plugins, languages, chrome runtime, and more) in place of the hand-rolled patch |
| Per-search page | `context.NewPage()` | `stealth.Page(browser)` (returns a `*rod.Page` with stealth already applied) |
| Response listener | `page.On("response", func(r playwright.Response))` | `page.EachEvent(func(e *proto.NetworkResponseReceived) bool { ... })` |
| Response status/headers/URL | `r.Status()` / `r.Headers()` / `r.URL()` | `e.Response.Status` / `e.Response.Headers` / `e.Response.URL` |
| Response body | `r.Body()` | `proto.NetworkGetResponseBody{RequestID: e.RequestID}.Call(page)` |
| Navigate | `page.Goto(url, PageGotoOptions{WaitUntil, Timeout})` | `page.Navigate(url)`, with the 60s deadline enforced by the same `select`/timeout pattern already used around the response channel, not by a per-call Playwright timeout option |
| Teardown | `context.Close()` / `browser.Close()` / `pw.Stop()` | `page.Close()` / `browser.Close()` / `launcher.Cleanup()` (kills the process go-rod launched) |

### No BrowserContext equivalent

Playwright's `BrowserContext` is an isolated session (cookies, UA, locale, viewport) that can spawn multiple pages. go-rod has no first-class equivalent in daily use — a `*rod.Browser` is normally used directly, with `browser.MustIncognito()` available if isolation between calls is ever needed (not currently required, since each `Search` call already gets exactly one fresh page and closes it). Decision: drop the context layer; `Client` holds the `*rod.Browser` (plus the `*launcher.Launcher` for cleanup) directly, and per-page setup (user agent, locale, timezone, viewport) that used to be set once on the context is instead applied to each page right after `stealth.Page(browser)` creates it. This is a few extra CDP calls per search, not a correctness risk — `Search` already re-creates a page every call.

### Browser channel A/B knob

go-rod's launcher resolves a browser binary automatically (system lookup, then auto-download if none found) — there's no built-in "bundled vs. named channel" concept like Playwright's `Channel` option. To preserve the `SCRAPER_BROWSER_CHANNEL` knob's intent for a fair A/B against the current implementation:
- Unset (default): let the launcher resolve normally (`launcher.New()` with no `.Bin()` override) — whatever it finds or downloads.
- `"chrome"`: require `launcher.LookPath()` to find a real installed Chrome and call `.Bin()` with that path explicitly, returning an error from `New()` if none is found — mirroring Playwright's `Channel: "chrome"` behavior, which fails rather than silently falling back.

### Testable launch configuration without a real browser

`TestChromiumLaunchOptions` currently asserts on a `playwright.BrowserTypeLaunchOptions` value's `Channel`/`Args` fields without launching anything. `*launcher.Launcher`'s internal flag storage may not be as directly inspectable. Decision: keep the "pure function, no browser" testability property by extracting a small unexported, plain-data return type (e.g. a struct listing the flags to set and the resolved binary path/"required" bit) from a function that takes the channel string — call it once to build test assertions, and a second, thin function turns that plain data into an actual `*launcher.Launcher`. The second function is not unit-tested in isolation (it just calls `launcher.Set`/`.Bin()`), but the first one preserves today's red/green cycle without touching a browser.

### Network response capture keeps the same relevance/timeout logic

`isRelevant(url)`, the `respCh`/`errCh` channel pattern, and the 60-second `select` timeout are engine-agnostic and are preserved as-is — only the event source changes (`page.On("response", ...)` → `page.EachEvent(...)`). This is deliberate: the unrelated-403 false-positive fix and the Cloudflare-challenge fail-fast behavior are the two hard-won bug fixes from the current implementation, and keeping their logic untouched (just re-wired to a different event type) is the safest way to avoid regressing either.

## Risks / Trade-offs

- **[Risk]** go-rod/stealth's bundled evasion script differs from the current 4-line manual patch and may behave differently against Cloudflare. → **Mitigation**: the acceptance bar is parity with today's pass/fail rate, not improvement; `investigate-104-bot-detection-alternatives` already established the real blocker is IP reputation, not fingerprint, so an at-parity result is expected and still worth landing (maintenance/consistency), not a regression to chase further.
- **[Risk]** go-rod's own browser auto-download (if `SCRAPER_BROWSER_CHANNEL` is unset and no system browser is found) is a different code path than `playwright install`, with different first-run latency/reliability in CI. → **Mitigation**: measure on the first real CI dispatch after the rewrite; if flaky, either cache the downloaded revision (`actions/cache`) or install system Chrome via apt and force the `"chrome"` channel instead of relying on auto-download.
- **[Risk]** `proto.NetworkGetResponseBody` can fail for certain already-consumed/cached responses, a known CDP quirk. → **Mitigation**: same class of failure already exists with Playwright's `r.Body()`; keep the current pattern of silently skipping a response whose body can't be read rather than erroring the whole search.
- **[Risk]** Dropping the BrowserContext layer means per-page setup (UA/locale/timezone/viewport) is repeated on every `Search` call instead of set once. → **Mitigation**: this is a minor duplication cost, not a correctness risk, since `Search` already creates and tears down one page per call today.

## Migration Plan

1. Land go-rod/go-rod/stealth behind the existing public API with tests green locally (browser auto-downloaded or system-installed).
2. Manually smoke-test the CLI once against production, same as prior spikes, before touching CI.
3. Update `.github/workflows/daily-scrape.yml`'s install step and retire the now-irrelevant `browser_channel` A/B input.
4. Remove `github.com/mxschmitt/playwright-go` from `go.mod`/`go.sum` and retire/replace `cmd/debug_playwright/main.go`.
5. Rollback is a plain `git revert` of the rewrite commit(s) — no data migration, no persisted state, so there is no partial-rollback hazard.

## Open Questions

- Exact go-rod/stealth function names/signatures (e.g. `stealth.Page`) should be confirmed against the actual pinned version's source/godoc once it's added to `go.mod` — this design was written without the dependency added yet, per the module-addition step being explicitly deferred until after design/specs/tasks are in place.
- Whether to keep an explicit CI browser-install/warm-up step or rely fully on go-rod's auto-download: left open until the first real CI dispatch after the rewrite lands, per the Risks section above.
