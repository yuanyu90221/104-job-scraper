## Why

`investigate-104-bot-detection-alternatives` produced direct evidence, gathered on the real GitHub-hosted CI runner, that browser fingerprint/channel (bundled Chromium vs. real Chrome, option B) does **not** explain today's Cloudflare blocks — both channels were blocked identically on the same shared-IP runner. That investigation is explicitly scoped to analysis only and does not touch `internal/client`'s implementation. Separately, the user has chosen to spend a small, time-boxed effort ruling out option C (`go-rod` + `go-rod/stealth`, a Go-native browser-automation stack) as a candidate replacement for `playwright-go`, since D/E (cookie/session reuse, self-hosted stable-IP runner) are blocked on infrastructure the user doesn't currently have. This change is that spike, done properly: TDD, with the same public behavior preserved, so it can be judged fairly against the current `playwright-go` implementation on the real CI runner.

## What Changes

- Replace `playwright-go` with `go-rod` + `go-rod/stealth` as the browser-automation engine inside `internal/client`, preserving the existing public API (`New`, `Client.Search`, `Client.Close`) so `cmd/main.go` and other callers require no changes.
- Preserve existing externally-observable behavior: `Search` still navigates to the 104 search URL, waits for the `/search/api/jobs` response, and returns `ErrCloudflareChallenge` when a Cloudflare managed challenge is detected instead of timing out silently.
- Preserve the `SCRAPER_BROWSER_CHANNEL` env var's intent (selectable browser binary) translated to `go-rod`'s launcher equivalent, and preserve the stealth init-script/fingerprint mitigations (`navigator.webdriver`, plugins, languages, `window.chrome`) either via `go-rod/stealth` defaults or an equivalent explicit patch if stealth's defaults don't cover them.
- Update `go.mod`/`go.sum` to add `go-rod`/`go-rod/stealth` and drop `playwright-go` once the rewrite is complete and validated.
- Update `.github/workflows/daily-scrape.yml`'s browser-install step (currently `playwright install chromium chrome`) to whatever `go-rod` requires (it can drive system Chrome/Chromium directly, or download its own via `rod`'s launcher `Launcher.Get`), and retire the now-irrelevant `browser_channel` A/B input once go-rod's approach is decided.
- Retire or repoint `cmd/debug_playwright/main.go` — either replace it with an equivalent `cmd/debug_gorod/main.go` diagnostic, or delete it once the rewrite lands, so the repo doesn't carry two parallel debug tools referencing a removed dependency.
- TDD throughout: write/port `internal/client_test.go`'s existing test cases (Cloudflare-challenge detection, unrelated-403 false-positive guard, successful search) against the new go-rod-backed implementation first (red), then implement to green, before doing any live validation against production or CI.
- **BREAKING (internal only)**: `chromiumLaunchOptions`/`browserChannelEnv` and other `playwright-go`-specific internals are removed/replaced; no impact on the package's public API or CLI flags.

## Capabilities

### New Capabilities
- `job-search-scraping`: the browser-automation-backed contract for retrieving 104 job search results — navigate to a search URL, capture the JSON API response, and explicitly detect/report a Cloudflare managed challenge as a distinct error — independent of which underlying browser-automation library implements it. This formalizes behavior that already exists in `internal/client` but has never been captured as a spec; writing it down now is what makes the go-rod rewrite testable against a fixed contract instead of "whatever the old code happened to do."

### Modified Capabilities
- (none — no existing `openspec/specs/` capability covers this yet; see New Capabilities)

## Impact

- Affected code: `internal/client/client.go`, `internal/client/client_test.go`, `go.mod`/`go.sum`, `.github/workflows/daily-scrape.yml`, `cmd/debug_playwright/main.go` (replaced or removed).
- Affected dependency: removes `github.com/mxschmitt/playwright-go`, adds `github.com/go-rod/rod` and `github.com/go-rod/stealth`.
- No change to `cmd/main.go`, CLI flags, output format, or the LINE notification path — `internal/client`'s public API is unchanged.
- Informs (and is informed by) `investigate-104-bot-detection-alternatives` tasks 3.2/4.1/4.2 — this change is the concrete follow-up spike/rewrite that change deferred.
