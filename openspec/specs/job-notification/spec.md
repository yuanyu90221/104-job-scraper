# job-notification Specification

## Purpose
TBD - created by archiving change switch-notifier-to-line-messaging-api. Update Purpose after archive.
## Requirements
### Requirement: Push job summary via LINE Messaging API
The system SHALL deliver the daily job summary by calling the LINE Messaging API's push message endpoint (`POST /v2/bot/message/push`) with a configured channel access token and a fixed target ID (a LINE `userId` or `groupId`), rather than the retired LINE Notify service.

#### Scenario: Successful push
- **WHEN** a caller invokes `LineNotifier.Send` with a non-empty job list, a keyword, and `topN` greater than zero
- **THEN** the notifier sends one text message containing up to `topN` formatted jobs to the configured target ID via the LINE Messaging API, and `Send` returns a nil error

#### Scenario: API failure surfaces as an error
- **WHEN** the LINE Messaging API responds to the push request with a non-2xx status
- **THEN** `Send` returns a non-nil error describing the failure, and does not panic or silently succeed

### Requirement: Messaging API credentials are explicit and validated
The system SHALL require a channel secret, a channel access token, and a target ID to construct a notifier, and SHALL fail fast with a descriptive error if any required credential is missing, rather than deferring to a confusing API-level failure.

#### Scenario: Missing channel token
- **WHEN** `NewLine` is called with an empty channel access token
- **THEN** `NewLine` returns a non-nil error and no notifier

#### Scenario: Missing channel secret
- **WHEN** `NewLine` is called with an empty channel secret
- **THEN** `NewLine` returns a non-nil error and no notifier

#### Scenario: Channel token set but target ID missing
- **WHEN** the CLI is invoked with a non-empty `--line-channel-token` but an empty `--line-target-id`
- **THEN** the CLI returns a configuration error before attempting any network call

### Requirement: CLI credential flags fall back to environment variables
The system SHALL default `--line-channel-secret`, `--line-channel-token`, and `--line-target-id` to the environment variables `LINE_CHANNEL_SECRET`, `LINE_CHANNEL_ACCESS_TOKEN`, and `LINE_TARGET_ID` respectively when the corresponding flag is not explicitly provided, so credentials can be supplied via environment (a sourced `.env` file locally, or a CI `env:` block) without appearing as literal values in shell history or workflow flag lists.

#### Scenario: Flag omitted, environment variable set
- **WHEN** the CLI is invoked without `--line-channel-secret` and the environment variable `LINE_CHANNEL_SECRET` is set
- **THEN** the CLI uses the environment variable's value as the channel secret

#### Scenario: Flag explicitly provided overrides environment variable
- **WHEN** the CLI is invoked with `--line-channel-secret` explicitly set and the environment variable `LINE_CHANNEL_SECRET` is also set to a different value
- **THEN** the CLI uses the flag's value, not the environment variable

### Requirement: Outgoing message respects the Messaging API length limit
The system SHALL truncate the outgoing text message to at most 5000 characters (the LINE Messaging API's text message limit), rather than the LINE Notify service's ~1000 character limit.

#### Scenario: Long job list is truncated, not rejected
- **WHEN** the formatted job summary exceeds 5000 characters
- **THEN** the notifier truncates the message to fit within the limit before sending, rather than sending an oversized message or erroring out

