## Context

`cmd/main.go` builds every search with `models.SearchParams{Order: 2, Asc: 0}` (`Order: 1=relevance, 2=date, 13=salary`, per `internal/models/job.go`). This has been the default since before this investigation; nothing regressed it — it was simply never validated against actual keyword relevance.

## Goals / Non-Goals

**Goals:**
- Confirm, with live evidence, whether `order`/`asc` is the cause of keyword-irrelevant results in the daily LINE notification.
- Fix the default so results match the search keyword.
- Make the exact request URL visible (logs + LINE message) so this and future relevance issues are easy to verify without a special debug tool.

**Non-Goals:**
- Client-side keyword-relevance filtering/scoring (deferred unless live evidence shows 104's own relevance ordering is still insufficient).
- Changing `asc` (currently `0`/descending for all three variants tested).

## Evidence: live comparison (2026-07-25)

`cmd/debugprobe`, keyword `golang 後端工程師`, page 1, run sequentially against production `www.104.com.tw` with a 3s gap between variants:

| variant | URL `order` param | jobs returned | relevance |
|---|---|---|---|
| `baseline` | `order=2` (date) | 22 | Almost entirely irrelevant: 前端工程師, PHP 工程師, AI系統架構師, 產品經理, Data Engineer, etc. Only 2-3 loosely relate to "後端". |
| `order1` | `order=1` (relevance) | 21 | Overwhelmingly relevant: `Golang 後端工程師`, `Golang 資深後端工程師`, `後端工程師（Golang）`, `遊戲後端工程師（GoLang）`, plus general Backend Engineer roles. |
| `ro0` | `order` omitted | 21 | **Byte-for-byte identical result set to `order1`** — confirms 104's own default (when `order` is absent) is relevance, not date. |

Conclusion: `order=2` (date) is the root cause. 104's search appears to match keywords loosely (tokenized/fuzzy across job fields) and only its relevance ranking actually surfaces genuinely matching postings within page 1; sorting by raw date instead defeats that ranking and surfaces whatever is newest regardless of match quality.

## Decisions

- **Change the hardcoded default from `Order: 2` to `Order: 1`.** Alternative considered: drop the `order` param entirely (relying on 104's own default). Rejected — behaviorally identical per the evidence above, but an explicit `1` documents the intent and doesn't depend on 104 never changing its unstated default.
- **Keep recency filtering as-is** (`--days` server-side + `--months` client-side date cutoff), since that's an orthogonal, already-working mechanism — relevance ordering and recency filtering are two different concerns and don't need to collapse into one.
- **Log the request URL from `internal/client.Search` (stderr), not a separate telemetry layer.** It's already the natural place the URL is constructed, requires no new dependency, and stderr is captured by both a local terminal and the `daily-scrape.yml` Action's step log with no workflow changes needed.
- **Add `RequestURL` to `models.SearchResponse` (`json:"-"`)** rather than a second return value from `Search`, to avoid changing every call site's error-handling shape; `Searcher.Run` and `cmd/main.go` were already the two calling layers that needed the value, and both already destructure the response.
- **Debug harness (`SetBaseURLForDebug`, `cmd/debugprobe`) was a fixed 3-variant switch**, not a generic query-param injector — only these three combinations were ever in question. Removed entirely once the fix was confirmed live, per repo convention of not leaving one-off diagnostic code behind; this evidence table is the durable record instead.

## Risks / Trade-offs

- [Risk] 104 could change what "relevance" means server-side in the future, silently regressing this fix. → Mitigation: the request URL is now logged (stderr) and included in every LINE message, so a future regression is visible immediately without needing to rebuild a debug tool from scratch.
- [Risk] Relevance ordering might still admit some loosely-related jobs 104 itself ranks as "relevant". → Mitigation: out of scope per Non-Goals; revisit with client-side filtering only if evidence shows this in practice.

## Migration Plan

Single-commit change, no data migration. Deploys via the next `daily-scrape.yml` run; no manual steps.

## Open Questions

- None outstanding — evidence is conclusive and the fix is a one-line default change.
