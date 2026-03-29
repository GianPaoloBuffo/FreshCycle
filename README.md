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

1. Install the base tools:
   - Node.js
   - Go
   - Docker Desktop
   - Supabase CLI
2. Copy the example env files:
   - `cp app/.env.example app/.env.local`
   - `cp api/.env.example api/.env.local`
3. Start the local Supabase stack from the repo root:
   - `supabase start`
4. Start the API:
   - `cd api && go run ./cmd/api`
5. Start the Expo app:
   - `cd app && npm install && npm run start`

The local Supabase stack provides:

- API URL: `http://127.0.0.1:54321`
- Postgres URL: `postgresql://postgres:postgres@127.0.0.1:54322/postgres`
- Studio: `http://127.0.0.1:54323`

For a fuller walkthrough, see [docs/local-development.md](/Users/gp-macbook/Projects/FreshCycle/docs/local-development.md#L1).

## Getting started

1. Copy the app and API example env files to `.env.local`.
2. Run `supabase start` from the repo root.
3. In [`app`](/Users/gp-macbook/Projects/FreshCycle/app), run `npm install` and `npm run start`.
4. In [`api`](/Users/gp-macbook/Projects/FreshCycle/api), run `go run ./cmd/api`.
5. Use `npm run typecheck` in [`app`](/Users/gp-macbook/Projects/FreshCycle/app) and `go test ./...` in [`api`](/Users/gp-macbook/Projects/FreshCycle/api) to verify the scaffold.
6. Review the initial schema in [`supabase/migrations`](/Users/gp-macbook/Projects/FreshCycle/supabase/migrations) and use `supabase db reset` or `supabase db push` when evolving it.

## Environment files

- [`app/.env.example`](/Users/gp-macbook/Projects/FreshCycle/app/.env.example#L1): local Expo app defaults for the Supabase CLI stack
- [`api/.env.example`](/Users/gp-macbook/Projects/FreshCycle/api/.env.example#L1): local Go API defaults for the Supabase CLI stack
- [.env.example](/Users/gp-macbook/Projects/FreshCycle/.env.example#L1): combined reference for app and API variables

Hosted Supabase values can still be used when needed, but local development should prefer the CLI-managed local stack.

## Next Phase 1 steps

- add root-level bootstrap commands for new-machine setup
