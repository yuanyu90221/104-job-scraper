# job-search-scraping Specification

## Purpose
TBD - created by archiving change fix-search-order-relevance-mismatch. Update Purpose after archive.
## Requirements
### Requirement: Default Search Ordering Favors Relevance
The system SHALL request search results ordered by relevance (`order=1`) by default, rather than by posting date (`order=2`), so returned jobs actually match the search keyword instead of being dominated by unrelated but recently-posted listings. Recency filtering (the `isnew`/days window and client-side month cutoff) is independent of this ordering and remains unaffected.

#### Scenario: Default search request omits date-based ordering
- **WHEN** the CLI builds a search request without an explicit order override
- **THEN** the request is sent with `order=1` (relevance), not `order=2` (date)

#### Scenario: Relevance ordering surfaces keyword-matching jobs
- **WHEN** a search for a multi-term keyword (e.g. "golang 後端工程師") is run with relevance ordering
- **THEN** the returned jobs are predominantly genuine matches for the keyword, not jobs that merely happen to be recently posted

