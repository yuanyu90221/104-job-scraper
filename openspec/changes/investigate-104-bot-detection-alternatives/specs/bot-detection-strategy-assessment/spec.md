## ADDED Requirements

### Requirement: Documented evidence of the current blocking behavior
The investigation SHALL capture live, reproducible evidence of how Cloudflare blocks non-browser requests to `www.104.com.tw`, rather than relying on assumption.

#### Scenario: Live request against the production search API
- **WHEN** a plain HTTP request (no JS execution) is sent to `https://www.104.com.tw/jobs/search/api/jobs` with realistic browser headers
- **THEN** the response and its Cloudflare-specific headers (e.g. `Cf-Mitigated`, `Server: cloudflare`, challenge-related CSP directives) are recorded in `design.md` as evidence

### Requirement: Comparative assessment of alternative strategies
The investigation SHALL enumerate candidate alternatives to the current stealth-Playwright approach and score each on reliability, maintenance cost, monetary cost, and legal/ToS risk.

#### Scenario: Reviewing the comparison table
- **WHEN** a reader opens `design.md`'s alternative comparison table
- **THEN** they find every candidate alternative identified during the investigation, each marked as either validated (with evidence) or explicitly flagged as needing a follow-up spike, with no alternative silently omitted from scoring

### Requirement: Actionable recommendation
The investigation SHALL end with a concrete, ordered recommendation for what to try next, distinct from merely listing options.

#### Scenario: Deciding what to implement next
- **WHEN** a reader wants to know what the next `internal/client` change should attempt
- **THEN** `design.md`'s Decisions section states which alternative(s) to try first, which to hold in reserve, and which to rule out, each with a stated reason
