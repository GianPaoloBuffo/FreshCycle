# FreshCycle API

This API scaffold now includes the first backend slices for the garment capture flow. It includes:

- a `chi` router
- environment-based config loading
- Supabase Postgres connection bootstrap
- a small `internal` package layout
- a `/health` endpoint for local and deployment checks
- a `POST /garments/parse-label` endpoint with provider-abstracted label parsing

## Layout

- `cmd/api`: application entrypoint
- `internal/config`: environment-driven bootstrap config
- `internal/postgres`: Postgres connection setup and validation
- `internal/app`: server wiring and lifecycle startup
- `internal/httpapi`: router setup and HTTP handlers
- `internal/labelparser`: provider abstraction plus parser implementations

## Commands

```bash
go run ./cmd/api
go test ./...
```

The API listens on `API_PORT`, defaulting to `8080`.

## Required environment

- `SUPABASE_DB_URL`: Postgres connection string for your local or hosted Supabase database
- `API_PORT`: optional HTTP port override
- `LABEL_PARSER_PROVIDER`: optional, defaults to `stub`; set to `openai` for real parsing
- `OPENAI_API_KEY`: required when `LABEL_PARSER_PROVIDER=openai`
- `OPENAI_MODEL`: optional, defaults to `gpt-5-mini`
- `OPENAI_BASE_URL`: optional override for the Responses API URL
- `API_ALLOWED_ORIGINS`: optional comma-separated browser origins allowed to call the API; defaults to local Expo web origins plus `https://*.vercel.app`

## Parse-label endpoint

`POST /garments/parse-label` expects `multipart/form-data` with an `image` field. The response shape preserves the garment parsing contract used for the app review step:

```json
{
  "name_suggestion": "Linen Shirt",
  "fabric_notes": ["55% linen", "45% cotton"],
  "wash_temp_max": 30,
  "machine_washable": true,
  "tumble_dry": false,
  "dry_clean_only": false,
  "iron_allowed": true,
  "iron_temp": "low",
  "bleach_allowed": false,
  "raw_label_text": "Machine wash cold. Do not bleach. Cool iron."
}
```

For local development, the API entrypoint loads `api/.env.local` and `api/.env` automatically if present.
