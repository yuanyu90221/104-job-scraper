## 1. Evidence gathering

- [x] 1.1 Add `SetBaseURLForDebug` to `internal/client.Client` (three fixed variants: `baseline`, `order1`, `ro0`) and a `cmd/debugprobe` CLI that runs all three sequentially against live 104 search
- [x] 1.2 Run `cmd/debugprobe` for `keyword=golang 後端工程師` against production and record results in `design.md`
- [x] 1.3 Confirm root cause: `order=2` (date) surfaces mostly keyword-irrelevant results; `order=1` and `order` omitted are identical and overwhelmingly relevant

## 2. Request-URL visibility (log + Action + LINE message)

- [x] 2.1 TDD: add `RequestURL` to `models.SearchResponse`; failing test in `internal/client/client_test.go` asserting `Search()`'s response contains the built URL, then implement in `internal/client.Search`
- [x] 2.2 Log the request URL to stderr on every page fetch in `internal/client.Search` (visible in local runs and in the `daily-scrape.yml` Action step log with no workflow changes)
- [x] 2.3 Thread the URL through `internal/search.Searcher.Run` (returns page 1's URL alongside jobs) and `cmd/main.go` (logs it, passes it to the notifier)
- [x] 2.4 TDD: failing tests in `internal/notifier/line_test.go` asserting `buildMessage`/`Send` include the request URL, then implement in `internal/notifier/line.go`

## 3. Fix the default ordering

- [x] 3.1 TDD: add a failing test pinning the production default to `Order: 1` (relevance) via an extracted `buildSearchParams` helper, then change `cmd/main.go`'s hardcoded `Order: 2` to `Order: 1`
- [x] 3.2 Run the full test suite (`go test ./...`) and confirm all green
- [x] 3.3 Re-run the real CLI live post-fix (`keyword=golang 後端工程師`) as a sanity check — 20-21 of 21 results are genuinely backend/Golang roles, vs. the pre-fix baseline where the majority were irrelevant

## 4. Close out

- [x] 4.1 Remove the throwaway `cmd/debugprobe` CLI and `SetBaseURLForDebug`/`applyDebugVariant` debug-only code from `internal/client/client.go` now that the fix is confirmed — no longer needed
- [x] 4.2 Write spec deltas: `job-search-scraping` (default order = relevance) and `job-notification` (message includes request URL)
- [x] 4.3 Verify with the Pants build tool: `pants tailor ::` (BUILD files in sync), `pants package ::`, `pants check ::`, `pants test ::` — all succeeded
- [x] 4.4 Archive the change
