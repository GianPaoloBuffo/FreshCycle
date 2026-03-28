# FreshCycle

FreshCycle is a mobile-first laundry companion for capturing garment care information, organizing a wardrobe, and planning compatible wash loads. This repository starts with the Phase 1 foundation only: project scaffolding, app structure, and the seams needed for API, database, and auth work.

## Phase 1 scope

The current implementation establishes the Phase 1 app, API, and schema foundation: the Expo app in [`/app`](/Users/gp-macbook/Projects/FreshCycle/app), the Go API scaffold in [`/api`](/Users/gp-macbook/Projects/FreshCycle/api), and the first Supabase migration set in [`/supabase`](/Users/gp-macbook/Projects/FreshCycle/supabase).

This is intentionally not implemented yet:

- monorepo bootstrap commands and orchestration

## Repository layout

- [`app`](/Users/gp-macbook/Projects/FreshCycle/app): Expo mobile app scaffold with Supabase email/password auth screens
- [`api`](/Users/gp-macbook/Projects/FreshCycle/api): Go API scaffold with `chi`, config wiring, and `/health`
- [`supabase`](/Users/gp-macbook/Projects/FreshCycle/supabase): foundational schema and migrations for garments and laundry schedules
- [`docs`](/Users/gp-macbook/Projects/FreshCycle/docs): optional product and setup docs as the project grows

## Getting started

1. Copy `.env.example` to `.env` or platform-specific local env files as needed.
2. Move into [`app`](/Users/gp-macbook/Projects/FreshCycle/app).
3. Install dependencies with `npm install` if they are not already present.
4. Start the app with `npm run start`.
5. Use `npm run typecheck` to validate the TypeScript scaffold.
6. Move into [`api`](/Users/gp-macbook/Projects/FreshCycle/api) and run `go test ./...` to verify the API scaffold.
7. Review the initial schema in [`supabase/migrations`](/Users/gp-macbook/Projects/FreshCycle/supabase/migrations) before wiring local Supabase.
8. Add real Supabase project values to `.env` so the auth screen can sign users in and up.

## Next Phase 1 steps

- add root-level bootstrap commands for new-machine setup
