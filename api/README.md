# FreshCycle API

This API scaffold provides the first backend slice for Phase 1. It includes:

- a `chi` router
- environment-based config loading
- Supabase Postgres connection bootstrap
- a small `internal` package layout
- a `/health` endpoint for local and deployment checks

## Layout

- `cmd/api`: application entrypoint
- `internal/config`: environment-driven bootstrap config
- `internal/postgres`: Postgres connection setup and validation
- `internal/app`: server wiring and lifecycle startup
- `internal/httpapi`: router setup and HTTP handlers

## Commands

```bash
go run ./cmd/api
go test ./...
```

The API listens on `API_PORT`, defaulting to `8080`.

## Required environment

- `SUPABASE_DB_URL`: Postgres connection string for your local or hosted Supabase database
- `API_PORT`: optional HTTP port override

For local development, the API entrypoint loads `api/.env.local` and `api/.env` automatically if present.
