## ADDED Requirements

### Requirement: Job Search Result Retrieval
The system SHALL retrieve 104 job search results by driving a headless browser to the search URL and capturing the JSON response from the page's own search API call, independent of which underlying browser-automation library performs the navigation.

#### Scenario: Successful search returns job data
- **WHEN** the search page loads and its own JavaScript fetches the search API and receives a normal JSON response with at least one job
- **THEN** the system returns a parsed result containing that job data, with no error

#### Scenario: No relevant response received within the timeout
- **WHEN** neither the search API response nor a Cloudflare challenge response is observed within 60 seconds of navigating
- **THEN** the system returns a timeout error rather than blocking indefinitely

### Requirement: Cloudflare Managed Challenge Detection
The system SHALL detect when a relevant response (the search page itself or the search API call) is a Cloudflare managed challenge and report it as a distinct, identifiable error instead of treating it as "no results" or letting it silently time out.

#### Scenario: Cloudflare challenge on a relevant request is detected
- **WHEN** the search page or the search API response comes back with HTTP 403, a `Cf-Mitigated: challenge` header (or a body containing both `challenges.cloudflare.com` and "Just a moment"), instead of job data
- **THEN** the system returns an error that can be identified as the Cloudflare-challenge error, distinct from other error types

#### Scenario: Unrelated subresource 403 does not abort the search
- **WHEN** a request unrelated to the search page or the search API (e.g. a keyword-suggest autocomplete call) receives an HTTP 403 or a Cloudflare challenge response, while the search API call itself succeeds
- **THEN** the system still returns the successful job data and does not report the Cloudflare-challenge error

### Requirement: Browser Channel Selection
The system SHALL allow the browser binary used for automation to be selected via an environment variable, to support comparing different browser channels without code changes.

#### Scenario: Default channel uses the automatically resolved browser
- **WHEN** the browser-channel environment variable is unset
- **THEN** the system launches using whatever browser binary it resolves by default, with no explicit channel requirement

#### Scenario: Explicit chrome channel requires a real installed Chrome
- **WHEN** the browser-channel environment variable is set to request a real installed Chrome and no such binary can be found on the system
- **THEN** the system returns an error rather than silently falling back to a different browser binary
