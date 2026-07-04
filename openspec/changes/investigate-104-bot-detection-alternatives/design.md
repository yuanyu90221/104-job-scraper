## Context

`internal/client/client.go` launches a stealth-configured headless Chromium via `playwright-go`, navigates to `https://www.104.com.tw/jobs/search/`, and intercepts the page's own `/search/api/jobs` XHR response. Git history shows this is the *fifth* iteration of the anti-bot strategy:

1. `592ef42`/`4f7ea7e` — plain `net/http` + browser-like headers + cookie jar + warm-up request. **Failed.**
2. `5d87eb4` — replaced `net/http` with a Playwright headless browser. Basic pass.
3. `ba08d8c`/`ddb38a7` — increased timeouts, added stealth launch args (`--disable-blink-features=AutomationControlled`, spoofed `navigator.webdriver`/`plugins`/`languages`) and switched to `/search/api/jobs`.
4. `12b56f9`/`34f505d` — tried a self-hosted runner with a real `DISPLAY` (headful-capable), reverted (`3d2dc2d`) because the self-hosted runner didn't reliably pick up `workflow_dispatch` triggers — **not** because it failed at Cloudflare.
5. `957c277`/`cfc638f` — inter-page jitter, URL encoding fix, `jobsource=joblist_search` param.

Live verification (2026-07-04) against the production site, using plain `curl` with a realistic desktop `User-Agent` and `Referer`, confirms the scope of the problem:

```
$ curl -sI ".../jobs/search/api/jobs?..." -A "Mozilla/5.0 ... Chrome/125 ..." -H "Referer: https://www.104.com.tw/jobs/search/"
HTTP/1.1 403 Forbidden
Cf-Mitigated: challenge
Server: cloudflare
Content-Security-Policy: ...script-src 'nonce-...' 'unsafe-eval' https://challenges.cloudflare.com...
```

Even `GET /robots.txt` returns the same Cloudflare "Just a moment..." **managed challenge** page (`cType: 'managed'`) with a Turnstile-backed JS challenge. This means:

- The block is **domain-wide**, not just on the API path.
- It is **JS-execution-gated**, not purely IP-reputation-gated — any client that doesn't execute the challenge script gets a 403, regardless of how browser-like its headers are. This is consistent with attempt #1 failing even with full header/cookie spoofing.
- A real browser capable of running the challenge script is the minimum bar. The open question is *which kind* of real browser, run *how*, is durable.
- CI evidence (`gh run list --workflow=daily-scrape.yml`, last 10 runs) shows the **current headless-Playwright approach mostly succeeds** (8/10 recent runs green); the 2 failures were a `playwright install` step failure and one unexplained scrape-step failure (logs expired, could not confirm root cause). So this isn't "completely broken" — it's "works today, fragile by construction," which is exactly the kind of problem worth a deliberate comparison instead of another one-off patch.

### Test-driven detection: making "blocked" observable

Before this investigation, `Client.Search` had no way to tell "Cloudflare blocked us" apart from "no results" or "slow network" — both silently fall through to a generic 60s-timeout error. That ambiguity is exactly what made attempts #3–#5 hard to evaluate (did the jitter/URL fix actually help, or did the run just get lucky?). To make the *investigation itself* falsifiable, this change adds a small, test-first diagnostic to `internal/client`:

1. **Red** — wrote `TestIsCloudflareChallenge`, `TestSearch_DetectsCloudflareChallenge`, and `TestSearch_ReturnsJobsOnSuccess` against a not-yet-existing `isCloudflareChallenge` classifier and `ErrCloudflareChallenge`, using the *real* captured Cloudflare challenge response (`internal/client/testdata/cloudflare_challenge.html`, `Cf-Mitigated: challenge`) as a fixture. Confirmed via `go vet` that these failed to compile — the gap was real, not assumed.
2. **Green** — implemented `isCloudflareChallenge(status, headers, body)` (checks the `Cf-Mitigated: challenge` header, falling back to a body signature match) and wired it into `Search`'s response handler so a challenge response returns `ErrCloudflareChallenge` immediately instead of waiting out the timeout. Made `Client.baseURL` injectable so tests can point a *real* headless browser at a local `httptest.Server` replaying the fixture — no production traffic involved, deterministic, and it still exercises the actual browser/CDP code path. All three new tests pass, including the two full-browser integration tests (`go test ./internal/client/... -v`, ~26s).
3. **Validate against production** — with the detector in place, ran the actual CLI once against the real site (`./bin/104-job-scraper --keyword="golang 後端工程師" --pages=1`) from this investigation's network. Result: `搜尋失敗: page 1: https://www.104.com.tw/jobs/search/api/jobs?...: blocked by Cloudflare challenge` — a clean, unambiguous failure, not a 60s hang. This is real evidence, not a guess: **from this specific network, the current stealth-Playwright approach fails the challenge**, even though the same code mostly passes on GitHub Actions runners (see CI evidence above). That gap — same code, same browser, different network/environment, different outcome — is itself useful signal: it means IP/network reputation is *also* a factor on top of the JS-execution gate, not an either/or. This directly supports trying option E (stable-IP self-hosted runner) and D (cookie reuse tied to that stable IP) if B/C don't fully solve it, and argues against assuming any fix is "done" from a single passing environment.

### Spike: option B (real Chrome channel), executed live against production

Option B was originally listed as "not validated live." It has since been spiked for real:

1. **Red** — `TestChromiumLaunchOptions` asserted a not-yet-existing `chromiumLaunchOptions(channel string)` helper: empty channel → bundled Chromium (no `Channel` field), `"chrome"` → `Channel: "chrome"`, stealth args applied either way. Confirmed via `go vet` that `chromiumLaunchOptions` didn't exist yet.
2. **Green** — extracted the existing launch-options literal out of `New()` into `chromiumLaunchOptions`, added the `Channel` field when a channel string is non-empty, and wired `New()` to read the channel from a new `SCRAPER_BROWSER_CHANNEL` env var (default unset → identical behavior to before). All existing tests plus the 3 new sub-tests pass; no browser needed to test the option-building logic itself.
3. **Validate against production** — this sandbox already had a real Chrome binary installed (`playwright install chrome` reported "already installed"), so the spike could run for real, not just in theory:
   - `SCRAPER_BROWSER_CHANNEL=chrome ./bin/104-job-scraper --keyword="golang 後端工程師" --pages=1` → **success, 22 jobs**, no challenge.
   - Immediately re-ran with the env var unset (bundled Chromium, i.e. today's status quo) from the same network → **also success, 22 jobs**.
   - Ran 2 more interleaved rounds of each (chrome, chromium, chrome, chromium) → **6/6 total runs succeeded**, zero `ErrCloudflareChallenge` from either channel.

**This is a different — and more important — result than expected.** On 2026-07-04, this same network was blocked immediately (`blocked by Cloudflare challenge`) using the unmodified status-quo bundled Chromium. Today, the *same* status-quo bundled Chromium — with no code change at all — passes cleanly every time. That rules out the simplest version of the option-B hypothesis ("bundled Chromium's fingerprint is why we get blocked; a real Chrome binary fixes it") as the *primary* explanation, because the control (bundled Chromium) now passes just as reliably as the treatment (real Chrome) with nothing else changed. The variable that actually flipped between the two days is time — most consistent with Cloudflare's bot-management score being **IP/network-reputation-driven and it recovers/decays over time**, not a stable, deterministic per-request fingerprint check that a better browser binary permanently defeats.

Practical consequence: this spike could not produce a blocked baseline today to compare against, so it cannot show whether B provides an incremental improvement *when actually challenged* — only that B causes no regression and both pass under today's (clean) reputation state. The original recommendation ("adopt B immediately, it directly targets the most likely cause") is downgraded from *the* fix to a *cheap, no-regret hardening* that should ship alongside, not instead of, work on IP/session durability (options D and E), since reputation — not just fingerprint — now looks like the dominant lever.

## Goals / Non-Goals

**Goals:**
- Lay out every realistic alternative for getting 104 search-result data past this Cloudflare managed challenge.
- Back each option with either live evidence gathered during this investigation, or a clearly labeled "needs a timeboxed spike" if it couldn't be validated without writing production code.
- Score each option on reliability, ongoing maintenance cost, monetary cost, and legal/ToS risk.
- End with a concrete recommendation for the next change.

**Non-Goals:**
- Implementing the chosen alternative (follow-up change).
- Re-litigating whether scraping 104 at all is acceptable — out of scope, this project already does it and the `daily-scrape.yml` workflow is an existing, working feature.
- Solving Cloudflare's challenge algorithm itself (e.g., writing a Turnstile solver) — treated as a black box; we only choose *whether* to run a real browser, reuse its output, or pay someone else to run one.

## Decisions

### Alternative comparison

| # | Alternative | How it works | Validated? | Reliability | Maintenance | Cost | Risk |
|---|---|---|---|---|---|---|---|
| A | **Status quo**: stealth Playwright headless Chromium per request | Current `client.go` | Live: 8/10 recent CI runs pass | Medium — works today, historically brittle (5 iterations already) | High — every Cloudflare bot-management update is a fire drill | Free (GitHub-hosted runner minutes) | Low legal risk, medium ops risk |
| B | **Real Chrome channel** instead of bundled Chromium (`playwright.BrowserTypeLaunchOptions{Channel: "chrome"}`) + `playwright install chrome` | Headless Chromium has telltale fingerprints (GPU string `SwiftShader`, missing codecs, CDP-only signals) that Cloudflare's bot management scores against; real Google Chrome binary reduces those tells | **Spiked live** (see "Spike: option B" above): 6/6 real production runs succeeded with both real Chrome and bundled Chromium today — the network's reputation had recovered since the 2026-07-04 block, so both channels passed and no incremental improvement from B could be measured against a blocked baseline | No regression, but improvement over A is unproven — today's evidence points to IP reputation as the dominant variable, not browser fingerprint alone | Low — implemented as an opt-in env var (`SCRAPER_BROWSER_CHANNEL=chrome`), one-line change + CI step tweak | Free | Low |
| C | **Go-native stealth browser automation**: replace `playwright-go` with `go-rod` + `go-rod/stealth` | `go-rod`'s stealth package patches many of the same CDP-detectable leaks (navigator.webdriver, permissions, iframe contentWindow leaks, etc.) that dedicated anti-detection forks (e.g. Patchright) patch for Node/Python — but natively in Go, so no cross-language binding gap | Not validated live — well-known community pattern, but untested against 104 specifically | Unknown until spiked — plausibly better than A, unproven | Medium — swaps a core dependency, needs re-testing all of `internal/search`'s consumers | Free | Low |
| D | **Session/cookie reuse**: solve the challenge once with a real browser, persist `cf_clearance` + related cookies, replay with `net/http` for subsequent requests | Cloudflare issues a `cf_clearance` cookie after a passed challenge, tied to IP + TLS/JA3 + UA fingerprint, typically valid ~30 min–a few hours | Partially validated: confirmed Cloudflare *does* set `__cf_bm`/challenge cookies on our 403 response, consistent with this model | Low on GitHub-hosted runners (fresh IP every run invalidates the binding) — only viable on a runner with a **stable IP**, i.e. self-hosted | Medium | Free | Low |
| E | **Self-hosted runner, headful Chromium under Xvfb, persistent identity** | Already attempted in `12b56f9`/`34f505d`, reverted for an unrelated CI-trigger reliability problem, not because it failed the challenge. Headful (real display) avoids headless-only fingerprint signals; a persistent machine gets a stable IP reputation over time (helps D and general trust score) | Partially validated: the revert was about `workflow_dispatch` not firing on the self-hosted runner, not about Cloudflare — worth re-attempting with the trigger issue fixed separately | Potentially the most durable option, unproven | High — requires maintaining a self-hosted runner (uptime, security patching, exposed to the internet) | Requires owning/renting a machine | Low legal risk, but self-hosted runners carry their own security exposure (must not be used on a public repo without care) |
| F | **Managed "web unlocker" API** (e.g. Bright Data Web Unlocker, ScrapingBee, ZenRows) | Third-party service runs its own browser farm + proxy rotation + challenge solving, returns final HTML/JSON over a simple HTTP call | Not validated (would require a paid account) | High, per vendor SLAs | Low — outsourced | **Paid**, usage-based (typically $ per 1k requests) | Medium — sends 104 traffic through a third party; check vendor ToS and 104's ToS |
| G | **Community challenge-solver proxy** (e.g. FlareSolverr-style sidecar) | Self-hosted sidecar that runs a browser internally, exposes a simple API returning cleared cookies/HTML for reuse by a plain HTTP client | Not validated. Note: the best-known project in this space (FlareSolverr) archived itself in 2024 after a Cloudflare legal complaint — real precedent risk | Depends on fork/maintenance status | High if self-maintained; the ecosystem is actively discouraged by Cloudflare | Free (self-hosted) | **High** — documented legal pushback from Cloudflare against this category of tool |
| H | **Alternate data source** (104 "open" partner API, RSS/sitemap, mobile-app API, other job boards) | Look for a lower-friction channel for the same data | Checked: `robots.txt` is challenge-gated too (no sitemap discoverable without solving the challenge). 104's public "open API" is an employer-facing job-*posting* API, not a jobseeker search API — not usable for this project's purpose. Mobile-app API is unverified and would need its own reverse-engineering spike | Unknown | N/A | High if pursued (reverse engineering) | N/A | Unclear ToS for an undocumented mobile endpoint |

### Recommendation

Layered, cheapest-first:

1. **Ship B (real Chrome channel) as a no-regret hardening, not as "the fix"** — it's already implemented behind an opt-in `SCRAPER_BROWSER_CHANNEL` env var, spiked live with zero failures, and costs nothing to keep. But the live spike could not reproduce a blocked baseline to prove it fixes anything — today both the real-Chrome and bundled-Chromium channels passed equally, which points at IP/network reputation, not fingerprint, as the more likely lever for the intermittent CI failures. Recommend flipping `SCRAPER_BROWSER_CHANNEL=chrome` on for real in `daily-scrape.yml` (cheap, safe) but not treating it as sufficient on its own.
2. **Prioritize D/E's IP-reputation angle over C** — today's finding (same network, same code, blocked on 07-04 and clean on 07-05 with no changes) is stronger evidence for a reputation-decay/recovery model than for "the wrong browser binary." That argues for re-validating cookie reuse (D) and a stable-IP runner (E) *before* spending migration effort on C, since C only changes fingerprint quality, the same axis B already tested inconclusively.
3. **Spike C (go-rod + stealth) only if D/E don't pan out** — it removes the Playwright dependency entirely and has a stronger community track record against Cloudflare specifically, but is a bigger change (new dependency, rewrite of `Client`/`Search`), and today's evidence doesn't clearly justify it over B.
4. **Do not pursue G (FlareSolverr-style sidecars)** — the legal precedent (Cloudflare's takedown campaign against this exact tool category) makes it a poor fit for a project that should stay low-maintenance and low-risk.
5. **F (paid unlocker API) is the pragmatic fallback** if reliability becomes business-critical (e.g. the daily LINE notification must never miss a day) and engineering time is scarcer than a monthly API bill.
6. **H (alternate source) stays parked** — no viable jobseeker-facing alternative was found; not worth a spike unless B/C/D/E/F all fail.

## Risks / Trade-offs

- [Risk] Cloudflare's bot-management scoring model changes over time (as it already has across 5 prior iterations). → Mitigation: none of these options is "permanent"; prefer B/C because they're cheap to re-try, and keep this design doc as a living reference for the next iteration instead of starting from zero.
- [Risk] Options B and C were not validated live in this investigation (no network-heavy install in this environment). → Mitigation: tasks.md schedules them as explicit spikes with a pass/fail criterion before any client.go change is proposed.
- [Risk] Recommending against G (FlareSolverr-style tools) is based on documented legal precedent, not a technical failure — worth re-checking if circumstances change. → Mitigation: noted explicitly rather than silently dropped.
- [Risk] This investigation's live checks were run from this sandbox's network, not from a GitHub Actions runner or the user's own machine — Cloudflare's response could differ by IP reputation. → Mitigation: recommendation explicitly says "re-validate B/C spikes in the actual CI environment," not just here.

## Migration Plan

Not applicable — this change produces no code migration, only a decision record. The follow-up change (once B or C is chosen) will carry its own migration plan for `client.go` and `daily-scrape.yml`.

## Open Questions

- ~~Does switching to `Channel: "chrome"` (option B) measurably reduce the two unexplained failure types seen in recent `daily-scrape.yml` runs?~~ Spiked live from this sandbox (see "Spike: option B" above): inconclusive, because both channels passed 6/6 today. Still open: does `SCRAPER_BROWSER_CHANNEL=chrome` change the CI success rate on GitHub-hosted runners specifically, where the two historical unexplained failures actually happened? That needs a few real `workflow_dispatch` runs with the env var set — not yet done, since it requires pushing a workflow change and consuming shared CI minutes (holding off for explicit go-ahead).
- If B is insufficient, is a `go-rod`/`stealth` rewrite (option C) worth the migration cost given `internal/client.Client`'s interface is only consumed by `internal/search`? (Small blast radius — favors trying it.) De-prioritized below D/E per the reputation-decay finding above.
- Is there appetite for owning a self-hosted runner (option E) long-term, given it reopens the exact `workflow_dispatch`-trigger problem that caused the prior revert (`3d2dc2d`)? This is a decision for the user, not something to resolve in this investigation.
