# FreshCycle Supabase schema

This directory holds the foundational Postgres schema for FreshCycle.

The initial migration focuses on the Phase 1 data model only:

- `garments`
- `laundry_schedules`
- `schedule_garments`

The schema keeps ownership aligned with `auth.users` and enables row-level security so later authenticated API work can remain user-scoped by default.

## Current model

### garments

Core garment records for the authenticated user. The table includes the attributes needed for the early garment save flow and later wardrobe grouping:

- display name
- category
- primary color
- wash temperature
- care instructions
- optional label image path

### laundry_schedules

User-owned schedules for recurring laundry reminders and due-today views. The recurrence string is stored directly for now so Phase 4 can use simple values like `daily`, `weekly:monday`, or `fortnightly` without introducing a more complex recurrence engine in the database.

### schedule_garments

Join table connecting schedules to one or more garments.

## Migrations

- `migrations/20260328194000_initial_schema.sql`: creates the initial tables, indexes, `updated_at` trigger, and row-level security policies
