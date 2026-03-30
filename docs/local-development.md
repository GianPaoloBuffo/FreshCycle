# Local Development

FreshCycle uses the Supabase CLI local stack for day-to-day development.
At the repo root, use the [Makefile](/Users/gp-macbook/Projects/FreshCycle/Makefile#L1) as the default command surface.

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
make bootstrap
```

Then start the local Supabase services:

```bash
make supabase-start
```

This exposes the default local endpoints:

- Project URL: `http://127.0.0.1:54321`
- REST API: `http://127.0.0.1:54321/rest/v1`
- Studio: `http://127.0.0.1:54323`
- Postgres: `postgresql://postgres:postgres@127.0.0.1:54322/postgres`

To confirm the stack is healthy:

```bash
make supabase-status
```

## Run the app

```bash
make app
```

## Run on an Android device with a dev build

Expo Go is not the long-term path for this app. FreshCycle now includes `expo-dev-client`, which lets you install a project-specific development build on Android.

From the repository root:

```bash
make app-android-dev
```

This runs `expo run:android`, generates the native Android project if needed, builds a debug APK, and installs it on a connected Android device or running emulator.

Suggested workflow:

1. Enable Developer Options and USB debugging on the Android device.
2. Connect the device over USB and confirm it appears in `adb devices`.
3. Run `make app-android-dev` once to install the dev client.
4. After the app is installed, use `make app` for the Metro bundler and open the FreshCycle dev build on the device.

Important network note:

- [`app/.env.local`](/Users/gp-macbook/Projects/FreshCycle/app/.env.local) cannot use `localhost` or `127.0.0.1` for services you want to reach from a physical Android device.
- Replace `EXPO_PUBLIC_API_BASE_URL` and, if needed, `EXPO_PUBLIC_SUPABASE_URL` with your laptop's LAN IP, for example `http://192.168.1.25:8080`.
- Keep `EXPO_PUBLIC_AUTH_REDIRECT_URL=freshcycle://auth/callback` so auth redirects continue to open the app.

Useful verification commands:

```bash
make test
cd app && CI=1 npx expo export --platform web
```

## Run the API

```bash
make api
```

Useful verification command:

```bash
make test
```

The API loads `api/.env` and `api/.env.local` automatically if present.

## Database workflow

Use local Supabase while developing schema changes:

```bash
make supabase-reset
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
make supabase-start
make supabase-status
make supabase-stop
make supabase-reset
```

What each one means:

- `make supabase-stop`: stops local services but keeps local state
- `make supabase-start`: starts or resumes the local stack
- `make supabase-status`: shows local URLs, keys, and service health
- `make supabase-reset`: destroys and recreates the local database from migrations and seeds

Use `make supabase-reset` when:

- you changed migrations
- you want a known-clean database state
- you want to verify the checked-in schema can be rebuilt from scratch

Avoid using `make supabase-reset` if you want to preserve local test data.

## Local env defaults

`app/.env.example` points the Expo app at the local Supabase API and local Go API.

`api/.env.example` points the Go API at the local Supabase Postgres instance.

If you intentionally want to use the hosted Supabase project instead, replace the local URLs and keys in your `.env.local` files with the hosted project values.

## Seed data

The local reset workflow expects [`supabase/seed.sql`](/Users/gp-macbook/Projects/FreshCycle/supabase/seed.sql#L1). It currently contains a placeholder script so resets stay predictable even before we add real seed data.
