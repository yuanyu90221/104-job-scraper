## Why

Production (`cmd/main.go`) hardcodes `models.SearchParams{Order: 2, Asc: 0}` — 104's "sort by posting date, newest first". A live `cmd/debugprobe` comparison (2026-07-25) against `keyword=golang 後端工程師` shows this default returns mostly keyword-irrelevant jobs (frontend, PHP, AI, product manager, etc. — 22 results, almost none actually "golang 後端"), while `order=1` (relevance) or omitting `order` entirely (104's own default, confirmed identical to `order=1`) returns 21 results that are overwhelmingly genuine Golang/backend postings. Sorting strictly by date, rather than relevance, is the root cause of the reported anomaly: the daily LINE notification has been surfacing jobs unrelated to the search keyword.

## What Changes

- Change the production default order from `2` (date) to `1` (relevance) in `cmd/main.go`, so search results actually match the keyword instead of being dominated by whatever was posted most recently. Recency is still respected via the existing `--days`/`--months` client-side date filtering, which is independent of `order`.
- Gathered the evidence above with a throwaway `cmd/debugprobe` CLI (plus a debug-only `SetBaseURLForDebug` on `internal/client.Client`) that reproduced three order/asc combinations (`baseline` order=2, `order1` order=1, `ro0` order omitted) against live 104 search. Both were removed once the fix was confirmed — the evidence is preserved in `design.md`, and there's no ongoing need for a live query-param injector in the shipped codebase.
- Surface the exact search request URL used for diagnosis/transparency: `models.SearchResponse` now carries `RequestURL`; `internal/client.Search` logs it to stderr on every page fetch (visible in local runs and in the `daily-scrape.yml` Action log); `internal/search.Searcher.Run` returns page 1's URL; `cmd/main.go` logs it and passes it to `notifier.LineNotifier.Send`, which now includes it in the LINE message so recipients can see exactly what was queried.

## Capabilities

### New Capabilities
- (none)

### Modified Capabilities
- `job-search-scraping`: default result ordering changes from posting-date to relevance.
- `job-notification`: the outgoing LINE message now includes the search request URL.

## Impact

- Affected code: `cmd/main.go`, `internal/client/client.go`, `internal/search/search.go`, `internal/notifier/line.go`, `internal/models/job.go`. (`cmd/debugprobe` was added and removed within this same change — evidence-gathering only, not part of the shipped diff.)
- Behavior change: daily LINE notification and CLI output should contain far fewer keyword-irrelevant jobs going forward; message body gains one extra line (search URL).
- No new dependencies, no breaking API changes outside this repo (all changed functions are internal to this module).
