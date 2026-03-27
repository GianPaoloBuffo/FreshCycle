# FreshCycle

FreshCycle is a mobile-first laundry companion for capturing garment care information, organizing a wardrobe, and planning compatible wash loads. This repository starts with the Phase 1 foundation only: project scaffolding, app structure, and the seams needed for API, database, and auth work.

## Phase 1 scope

The current implementation addresses `GP-9` by establishing the Expo app in [`/app`](/Users/gp-macbook/Projects/FreshCycle/app) with TypeScript, Expo Router, shared screen primitives, and environment accessors.

This is intentionally not implemented yet:

- Go API scaffold in `/api`
- Supabase schema and migrations in `/supabase`
- email/password auth flow
- monorepo bootstrap commands and orchestration

## Repository layout

- [`app`](/Users/gp-macbook/Projects/FreshCycle/app): Expo mobile app scaffold
- [`api`](/Users/gp-macbook/Projects/FreshCycle/api): reserved for the Go API
- [`supabase`](/Users/gp-macbook/Projects/FreshCycle/supabase): reserved for migrations, seed data, and local config
- [`docs`](/Users/gp-macbook/Projects/FreshCycle/docs): optional product and setup docs as the project grows

## Getting started

This workspace currently uses a local Node bootstrap under `.local-tools` because Node was not available on the machine PATH during setup. If you already have Node installed globally, you can ignore that directory and use your system toolchain instead.

1. Copy `.env.example` to `.env` or platform-specific local env files as needed.
2. Move into [`app`](/Users/gp-macbook/Projects/FreshCycle/app).
3. Install dependencies with `npm install` if they are not already present.
4. Start the app with `npm run start`.
5. Use `npm run typecheck` to validate the TypeScript scaffold.

## Next Phase 1 steps

- scaffold the Go API in `/api`
- define the first Supabase schema and migration flow
- wire a Supabase client into the Expo app
- implement the initial auth state and route protection
- add root-level bootstrap commands for new-machine setup
