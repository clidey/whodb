## Summary

Expected request aborts are still likely being captured as frontend exceptions. That is noisy telemetry and makes it harder to distinguish real failures from normal cancellation behavior.

## Current evidence

- Global window error and unhandled rejection handlers forward exceptions to PostHog:
  - `frontend/src/config/posthog.tsx:143`
  - `frontend/src/config/posthog.tsx:150`
- Apollo/global network handling does not currently filter expected aborts:
  - `frontend/src/config/graphql-client.ts:101`
- Some request paths fire promise chains without a local catch for cancellation:
  - `frontend/src/pages/storage-unit/explore-storage-unit.tsx:369`

## Why this should be tracked

The original draft treated `AbortError` as a product failure. The better framing is telemetry hygiene: expected aborts from navigation, re-query, or teardown should not hit the global exception pipeline as if they were real faults.

## Scope

- Identify which frontend request paths can produce expected aborts.
- Filter known benign abort/cancel cases before they reach global exception capture.
- Preserve reporting for real network failures and unexpected promise rejections.

## Acceptance criteria

- Expected aborts no longer appear in frontend exception tracking.
- Real request failures still surface in telemetry and UI error states.
- The filtering logic is centralized rather than duplicated across many pages.
