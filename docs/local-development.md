# Local Development

FreshCycle uses the Supabase CLI local stack for day-to-day development.

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

This recreates the local database and reapplies all checked-in migrations.

When you are ready to apply new migrations to the linked hosted project:

```bash
supabase db push
```

## Local env defaults

`app/.env.example` points the Expo app at the local Supabase API and local Go API.

`api/.env.example` points the Go API at the local Supabase Postgres instance.

If you intentionally want to use the hosted Supabase project instead, replace the local URLs and keys in your `.env.local` files with the hosted project values.
