# Feature: Export readable repository security settings

## Rule: Acquire security families independently

Ordinary `export` attempts all seven supported Security families independently:
Secret Protection, Dependabot, CodeQL configuration, CodeQL default setup, AI Scan,
private vulnerability reporting and immutable releases. Product-wide Code Security
and Advanced Security switches remain excluded. All other supported families now follow the [automatic settings contract](readable-settings-export.feature.md).

Passing `--security FAMILY` fails before remote work with guidance about automatic
observation. Plan and apply require complete observations of declared settings.

### Scenario: Retain one unread security family while refreshing another

- Given the current file declares private vulnerability reporting and immutable releases
- When export cannot read private vulnerability reporting but reads immutable releases as enabled
- Then it reports the unavailable family and keeps its prior declaration
- And it writes the observed enabled value for immutable releases

## Rule: Distinguish an unavailable CodeQL read from confirmed disablement

- A family is adopted only after a complete supported observation and conversion.
  Missing/denied/inapplicable/malformed/unsupported reads emit an `info: export`
  diagnostic to stderr, identify the family and a bounded reason, and continue
  independent Security reads. Upstream response bodies/messages/URLs, tokens and
  variable/secret values are never rendered in these info messages.
- Initial export omits unavailable families. A successful false value is emitted
  as false; failure is never an observed false, empty collection or default.
- Existing valid declarations retain unavailable Security families. Retained
  values are prior user intent, not verified remote state. Acquired families
  replace their complete owned values. CodeQL state and details use
  a single response. Confirmed `not-configured` exports false and removes prior
  details with info. Unknown state preserves the prior pair. Configured state
  exports true and acquired details; unavailable details retain prior intent
  with info. No prerequisite is enabled by export.
- Existing endpoint-specific 404 and explicit normalization policies remain in
  force; this feature does not reinterpret every 404 or change apply behavior.

### Scenario: Clear old CodeQL details after confirmed disablement

- Given the existing declaration enables CodeQL default setup and specifies languages
- When the single default-setup response reports `not-configured`
- Then export writes the disabled value, removes the old details and emits an informational diagnostic

## Local safety and fatal errors

Target resolution, core repository/export prerequisites, invalid or unreadable
existing files, diagnostic IO errors and destination write failures remain fatal.
A bad existing file is preserved, not silently overwritten. Prepare a validated
snapshot before remote reads; refuse replacement if the file changes, disappears,
becomes a symlink, or appears concurrently during an initial export. Existing-file
replacement is guarded and atomic, not a concurrency lock. Encoding and read
failures never create a plausible partial declaration from an incomplete family.

## Acceptance

Use local HTTP/Git fixtures and the actual CLI to verify automatic no-option
observation, both successful booleans, denied/missing/malformed families followed
by successful independent reads, first-file omission versus re-export retention,
CodeQL confirmed disabled/configured/unknown states with and without prior details,
single-response consistency, secret-free info, retired-option rejection and fundamental
file/target failures. Preserve plan/apply strict-read and side-effect cases.

The [CodeQL export reader tests](../../../internal/github/codeql_export_test.go) own
single-GET state/detail decoding and HTTP/network/malformed failure semantics.
The [application matrix](../../../internal/app/export_security_test.go) checks prior
intent and later-family continuation. The [built-CLI fixture](../../../tests/acceptance/readable_security_export_test.go)
checks valid first/re-export output, detail removal and bounded diagnostics.
