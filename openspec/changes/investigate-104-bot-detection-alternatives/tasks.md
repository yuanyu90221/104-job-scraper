## 1. Evidence gathering

- [x] 1.1 Capture live Cloudflare response headers/body for a plain HTTP request to `/jobs/search/api/jobs` (no JS execution) and record findings in `design.md`
- [x] 1.2 Check whether the block is API-path-specific or domain-wide (tested `robots.txt`, the search HTML page)
- [x] 1.3 Review git history of prior anti-bot fix attempts (`internal/client`, `daily-scrape.yml`) to avoid re-proposing something already tried and rejected
- [x] 1.4 Pull recent `daily-scrape.yml` CI run outcomes (`gh run list`) to establish the current real-world success/failure rate of the status quo
- [x] 1.5 TDD: add `isCloudflareChallenge`/`ErrCloudflareChallenge` to `internal/client` (red→green, fixtures from the real captured challenge response) so blocked runs fail fast with a clear error instead of a 60s timeout; then run the real CLI once against production to confirm the diagnostic and the current failure mode end-to-end (see `design.md` "Test-driven detection")

## 2. Comparative analysis

- [x] 2.1 Enumerate candidate alternatives (browser-fingerprint fixes, Go-native stealth libraries, cookie/session reuse, self-hosted runner, paid unlocker APIs, community challenge-solver proxies, alternate data sources)
- [x] 2.2 Score each alternative on reliability, maintenance cost, monetary cost, and legal/ToS risk in `design.md`'s comparison table
- [x] 2.3 Write an ordered recommendation (what to try first, what to hold in reserve, what to rule out and why)

## 3. Validation spike

- [x] 3.1 Spike option B locally (TDD): added an opt-in `SCRAPER_BROWSER_CHANNEL` env var + `chromiumLaunchOptions` helper (red→green, unit-tested without a real browser), then ran the real CLI against production 6 times (3× real Chrome channel, 3× bundled Chromium, interleaved). Result: **6/6 succeeded, both channels** — inconclusive on incremental benefit, because this network's block from 2026-07-04 had cleared by 2026-07-05 even for the unmodified status quo. Stronger evidence for IP-reputation decay/recovery than for a fingerprint-only explanation (see `design.md` "Spike: option B"). **Follow-up still open**: run the same A/B via a few real `daily-scrape.yml` `workflow_dispatch` runs on GitHub-hosted runners (where the two historical unexplained failures actually happened) — deferred pending explicit go-ahead, since it means pushing a workflow change and spending shared CI minutes.
- [ ] 3.2 De-prioritized: today's evidence points at IP reputation (D/E) over browser fingerprint (C) as the more likely lever — consider a D/E spike before time-boxing `go-rod` + `go-rod/stealth` (option C)

## 4. Decision handoff

- [ ] 4.1 Confirm with the user which alternative to implement based on spike results (or, if no spike is run, based on the analysis alone)
- [ ] 4.2 Open a follow-up OpenSpec change scoped to the actual `internal/client.go` (and `daily-scrape.yml`, if needed) rewrite for the chosen alternative
