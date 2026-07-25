## MODIFIED Requirements

### Requirement: Push job summary via LINE Messaging API
The system SHALL deliver the daily job summary by calling the LINE Messaging API's push message endpoint (`POST /v2/bot/message/push`) with a configured channel access token and a fixed target ID (a LINE `userId` or `groupId`), rather than the retired LINE Notify service. The message SHALL include the exact search request URL used to fetch the jobs, so the recipient can verify what was queried.

#### Scenario: Successful push
- **WHEN** a caller invokes `LineNotifier.Send` with a non-empty job list, a keyword, `topN` greater than zero, and the search request URL
- **THEN** the notifier sends one text message containing up to `topN` formatted jobs and the search request URL to the configured target ID via the LINE Messaging API, and `Send` returns a nil error

#### Scenario: API failure surfaces as an error
- **WHEN** the LINE Messaging API responds to the push request with a non-2xx status
- **THEN** `Send` returns a non-nil error describing the failure, and does not panic or silently succeed

#### Scenario: Message includes the search request URL
- **WHEN** `Send` is called with a search request URL
- **THEN** the outgoing message text contains that URL
