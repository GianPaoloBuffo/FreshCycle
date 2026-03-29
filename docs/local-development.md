# Local Development

FreshCycle uses the Supabase CLI local stack for day-to-day development.

## Orchestration choice

FreshCycle uses the Supabase CLI as the source of truth for local infrastructure orchestration.

We are intentionally not maintaining a separate `docker-compose.yml` for Supabase services right now because the CLI already manages:

- Postgres
- Auth
- Storage
- Studio
- the local API gateway
- migration and reset commands tied to the checked-in `supabase` directory

That keeps local development aligned with Supabase's default workflow and avoids maintaining two competing local stack definitions.

## Prerequisites

- Node.js
- Go
- Docker Desktop
- Supabase CLI

## First-time setup

From the repository root:

```bash
cp app/.env.example app/.env.local
cp api/.env.example api/.env.local
```

Then start the local Supabase services:

```bash
supabase start
```

This exposes the default local endpoints:

- Project URL: `http://127.0.0.1:54321`
- REST API: `http://127.0.0.1:54321/rest/v1`
- Studio: `http://127.0.0.1:54323`
- Postgres: `postgresql://postgres:postgres@127.0.0.1:54322/postgres`

To confirm the stack is healthy:

```bash
supabase status
```

## Run the app

```bash
cd app
npm install
npm run start
```

Useful verification commands:

```bash
npm run typecheck
CI=1 npx expo export --platform web
```

## Run the API

```bash
cd api
go run ./cmd/api
```

Useful verification command:

```bash
go test ./...
```

The API loads `api/.env` and `api/.env.local` automatically if present.

## Database workflow

Use local Supabase while developing schema changes:

```bash
supabase db reset
```

This recreates the local database and reapplies all checked-in migrations and seeds.

When you are ready to apply new migrations to the linked hosted project:

```bash
supabase db push
```

## Persistence strategy

The Supabase CLI stores local service state in Docker-managed volumes, so normal `supabase stop` and `supabase start` cycles keep your local database data.

Use these commands intentionally:

```bash
supabase stop
supabase start
supabase status
supabase db reset
```

What each one means:

- `supabase stop`: stops local services but keeps local state
- `supabase start`: starts or resumes the local stack
- `supabase status`: shows local URLs, keys, and service health
- `supabase db reset`: destroys and recreates the local database from migrations and seeds

Use `supabase db reset` when:

- you changed migrations
- you want a known-clean database state
- you want to verify the checked-in schema can be rebuilt from scratch

Avoid using `supabase db reset` if you want to preserve local test data.

## Local env defaults

`app/.env.example` points the Expo app at the local Supabase API and local Go API.

`api/.env.example` points the Go API at the local Supabase Postgres instance.

If you intentionally want to use the hosted Supabase project instead, replace the local URLs and keys in your `.env.local` files with the hosted project values.

## Seed data

The local reset workflow expects [`supabase/seed.sql`](/Users/gp-macbook/Projects/FreshCycle/supabase/seed.sql#L1). It currently contains a placeholder script so resets stay predictable even before we add real seed data.
