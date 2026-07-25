## Why

`internal/client` drives a headless Playwright/Chromium browser to bypass 104's Cloudflare bot protection and capture the search API's JSON response. Live checks against `www.104.com.tw` (2026-07-04) confirm the entire domain — including `robots.txt` — sits behind a Cloudflare **managed challenge** (`Cf-Mitigated: challenge`, CSP referencing `challenges.cloudflare.com`) that requires real JS execution to pass. The current stealth-Playwright approach mostly works in CI today (recent `daily-scrape.yml` runs are mostly green), but it has already gone through five prior fix attempts (plain `net/http` + cookies → headless Playwright → stealth flags → self-hosted runner + real display, reverted → jitter/URL tweaks) and remains fragile: any change to Cloudflare's bot-management heuristics or the CI runner's IP reputation can silently break job search / the daily LINE notification. Before making another reactive patch to `client.go`, we should evaluate and validate which architectural approach is actually resilient, so the next fix isn't another guess.

## What Changes

- Document the current failure mode with live evidence (Cloudflare challenge headers/response captured directly against the production endpoints).
- Enumerate candidate alternative strategies for acquiring 104 job data (browser-fingerprint improvements, Go-native stealth libraries, session/cookie reuse, self-hosted runner with persistent identity, third-party unlocker APIs, community challenge-solver proxies, alternate endpoints), with validated pros/cons, cost, legal/ToS risk, and maintenance burden for each.
- Recommend which alternative(s) are worth a follow-up implementation spike, and which are ruled out with evidence.
- **No architectural rewrite of `internal/client` as part of this change** — the goal is still investigation/decision-making, not adopting a new strategy. It does add small, TDD'd pieces needed to gather real evidence, with default behavior otherwise unchanged: (1) `isCloudflareChallenge`/`ErrCloudflareChallenge` so a block is an explicit error instead of a 60s timeout, (2) a `SCRAPER_BROWSER_CHANNEL` env var (default: unset, unchanged bundled-Chromium behavior) so option B (real Chrome channel) could actually be spiked against production rather than assessed on paper, and (3) a genuine bug fix found *by* validating (2) on the real CI runner — the 403/"blocked" detection wasn't scoped to the search API, so an unrelated subresource 403 could falsely abort a search that would have otherwise succeeded; now scoped to the page URL / `/search/api/jobs`. Also fixes a `daily-scrape.yml` monitoring blind spot (`continue-on-error` was masking real scrape failures behind a green run conclusion) found while cross-checking CI history for this same investigation. The actual chosen-alternative rewrite (adopting D and/or E) is still a separate follow-up change.

## Capabilities

### New Capabilities
- `bot-detection-strategy-assessment`: a documented, evidence-backed comparison of alternative approaches for retrieving 104 job data past Cloudflare bot detection, ending in a concrete recommendation.

### Modified Capabilities
- (none — `internal/client`'s actual scraping behavior is unchanged by this investigation change)

## Impact

- Affected docs: new `design.md` comparison table and recommendation in this change; no source files touched.
- Informs a future change against `internal/client/client.go` and possibly `.github/workflows/daily-scrape.yml` (e.g. self-hosted runner, added secrets for a third-party API key, or a new Go dependency like `go-rod`).
- No user-facing behavior changes yet; CLI and daily scrape continue running on the current Playwright implementation until a follow-up change lands.
