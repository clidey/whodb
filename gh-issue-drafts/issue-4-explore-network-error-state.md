## Summary

Network failures on the explore/storage-unit path do not consistently produce actionable UI state. Some request paths show toast/error UI, but the main row-loading path can fail without a clear inline state.

## Current evidence

- The main explore row load issues a lazy query and only updates state on success:
  - `frontend/src/pages/storage-unit/explore-storage-unit.tsx:344`
  - `frontend/src/pages/storage-unit/explore-storage-unit.tsx:377`
- Login does have explicit network handling that updates health state:
  - `frontend/src/pages/auth/login.tsx:281`
- The app already has server/database overlays driven by health state:
  - `frontend/src/app.tsx:132`
  - `frontend/src/components/health/health-overlays.tsx:56`

## Why this should be tracked

This is still a real UX gap in the current checkout, but it should be scoped narrowly to storage/explore network failures rather than as a generic Apollo-wide ticket.

## Scope

- Decide how explore-page row-fetch failures should surface:
  - inline error state
  - toast plus retry affordance
  - health overlay handoff for backend-unreachable cases
- Ensure the main row-loading path handles rejected requests explicitly.
- Keep behavior consistent with existing login and health-check patterns.

## Acceptance criteria

- If the backend is unreachable during storage-unit row loading, the user sees a clear actionable state.
- Failed row loads do not silently leave the page in an ambiguous stale/empty state.
- The error treatment is consistent across the main explore load and page-change/refetch flows.
