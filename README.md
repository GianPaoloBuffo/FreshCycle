# FreshCycle

FreshCycle is a mobile-first laundry companion for capturing garment care information, organizing a wardrobe, and planning compatible wash loads. This repository starts with the Phase 1 foundation only: project scaffolding, app structure, and the seams needed for API, database, and auth work.

## Phase 1 scope

The current implementation establishes the Phase 1 app, API, and schema foundation: the Expo app in [`/app`](/Users/gp-macbook/Projects/FreshCycle/app), the Go API scaffold in [`/api`](/Users/gp-macbook/Projects/FreshCycle/api), and the first Supabase migration set in [`/supabase`](/Users/gp-macbook/Projects/FreshCycle/supabase).

This is intentionally not implemented yet:

- monorepo bootstrap commands and orchestration

## Repository layout

- [`app`](/Users/gp-macbook/Projects/FreshCycle/app): Expo mobile app scaffold with Supabase email/password auth screens
- [`api`](/Users/gp-macbook/Projects/FreshCycle/api): Go API scaffold with `chi`, config wiring, `/health`, and Supabase Postgres startup wiring
- [`supabase`](/Users/gp-macbook/Projects/FreshCycle/supabase): foundational schema and migrations for garments and laundry schedules
- [`docs`](/Users/gp-macbook/Projects/FreshCycle/docs): setup and product docs as the project grows

## Local development

FreshCycle now uses the Supabase CLI local stack for development by default.

This repository treats the Supabase CLI as the local orchestration layer for database and supporting services. We are not maintaining a parallel Docker Compose setup for Supabase right now.

The default root command surface is now the project [Makefile](/Users/gp-macbook/Projects/FreshCycle/Makefile#L1).

1. Install the base tools:
   - Node.js
   - Go
   - Docker Desktop
   - Supabase CLI
2. Bootstrap the repo:
   - `make bootstrap`
3. Start the local Supabase stack from the repo root:
   - `make supabase-start`
4. Start the API:
   - `make api`
5. Start the Expo app:
   - `make app`

The local Supabase stack provides:

- API URL: `http://127.0.0.1:54321`
- Postgres URL: `postgresql://postgres:postgres@127.0.0.1:54322/postgres`
- Studio: `http://127.0.0.1:54323`

For a fuller walkthrough, see [docs/local-development.md](/Users/gp-macbook/Projects/FreshCycle/docs/local-development.md#L1).

## Getting started

1. Run `make bootstrap`.
2. Run `make supabase-start`.
3. Run `make api`.
4. Run `make app`.
5. Run `make test` to verify the scaffold.
6. Review the initial schema in [`supabase/migrations`](/Users/gp-macbook/Projects/FreshCycle/supabase/migrations) and use `make supabase-reset` or `supabase db push` when evolving it.

Useful local Supabase lifecycle commands:

- `make supabase-start`
- `make supabase-status`
- `make supabase-stop`
- `make supabase-reset`
- `supabase db push`

## Environment files

- [`app/.env.example`](/Users/gp-macbook/Projects/FreshCycle/app/.env.example#L1): local Expo app defaults for the Supabase CLI stack
- [`api/.env.example`](/Users/gp-macbook/Projects/FreshCycle/api/.env.example#L1): local Go API defaults for the Supabase CLI stack
- [.env.example](/Users/gp-macbook/Projects/FreshCycle/.env.example#L1): combined reference for app and API variables

Hosted Supabase values can still be used when needed, but local development should prefer the CLI-managed local stack.

## Next Phase 1 steps

- continue refining the monorepo command surface as more slices land
