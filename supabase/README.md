# FreshCycle Supabase schema

This directory holds the foundational Postgres schema for FreshCycle.

## Local workflow

FreshCycle uses the Supabase CLI for local development orchestration rather than maintaining a separate Docker Compose setup.

Common commands:

```bash
supabase start
supabase status
supabase stop
supabase db reset
supabase db push
```

The checked-in `config.toml` defines the local service ports and behavior. The checked-in `seed.sql` is used during `supabase db reset`.

The initial migration focuses on the Phase 1 data model only:

- `garments`
- `laundry_schedules`
- `schedule_garments`
- private `garment-labels` storage bucket for label uploads

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

### garment-labels storage bucket

Private Supabase Storage bucket for care-label images. Objects live under user-scoped paths such as
`{user_id}/labels/{garment_id}.jpg`, so the authenticated user session can upload and later request a
signed URL without widening access to other users' files.

### laundry_schedules

User-owned schedules for recurring laundry reminders and due-today views. The recurrence string is stored directly for now so Phase 4 can use simple values like `daily`, `weekly:monday`, or `fortnightly` without introducing a more complex recurrence engine in the database.

### schedule_garments

Join table connecting schedules to one or more garments.

## Migrations

- `migrations/20260328194000_initial_schema.sql`: creates the initial tables, indexes, `updated_at` trigger, and row-level security policies
- `migrations/20260412131500_add_private_label_storage.sql`: creates the private `garment-labels` bucket and user-scoped storage policies
- `seed.sql`: placeholder local seed file used by the CLI reset workflow
