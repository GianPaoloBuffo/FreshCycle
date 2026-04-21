# Schedule recurrence

FreshCycle Phase 4 keeps recurrence intentionally small and string-based:

- `daily`: due every local calendar day.
- `weekly:<weekday>`: due on one local weekday, where weekday is `sunday` through `saturday`.
- `fortnightly`: due every 14 days from the schedule's explicit `starts_on` date.

The client stores `starts_on` as `YYYY-MM-DD`. Fortnightly reminders use that value as the deterministic anchor for both local notification scheduling and the home-screen Today section. If a schedule is missing `starts_on`, older fallback paths may use `created_at`, but new schedules should always send `starts_on`.

Current edge-case handling:

- Invalid recurrence strings are ignored for Today and rejected during schedule creation.
- Invalid start dates are rejected during schedule creation.
- Local notification scheduling creates a small window of concrete upcoming notification dates so fortnightly schedules can be cancelled and replaced by schedule id.
