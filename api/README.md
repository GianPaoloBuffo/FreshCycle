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
- `LABEL_PARSER_PROVIDER`: optional, defaults to `stub`; supported values are `stub`, `ocr`, `hybrid`, and `openai`
- `OCR_TESSERACT_PATH`: optional, defaults to `tesseract`; used by `ocr` and `hybrid`
- `OCR_LANGUAGES`: optional, defaults to `eng+spa`; install matching Tesseract language packs locally and in deployment
- `LABEL_PARSER_FALLBACK_PROVIDER`: optional, defaults to `gemini`; used when `LABEL_PARSER_PROVIDER=hybrid`
- `GEMINI_API_KEY`: required when `LABEL_PARSER_PROVIDER=hybrid`
- `GEMINI_MODEL`: optional, defaults to `gemini-3.1-flash-lite`
- `GEMINI_BASE_URL`: optional, defaults to `https://generativelanguage.googleapis.com/v1beta`
- `SYMBOL_DETECTOR_ENABLED`: optional, defaults to `true`; enables the local taxonomy/rules detector that normalizes client symbol hints into backend detections
- `SCAN_CACHE_ENABLED`: optional, defaults to `true`; caches structured scan interpretations by SHA-256 image hash in API memory
- `SCAN_CACHE_TTL`: optional Go duration, defaults to `24h`; controls in-memory scan cache retention
- `SCAN_CACHE_MAX_ENTRIES`: optional, defaults to `512`; caps in-memory scan cache entries
- `OPENAI_API_KEY`: required when `LABEL_PARSER_PROVIDER=openai`
- `OPENAI_MODEL`: optional, defaults to `gpt-4.1-mini`
- `OPENAI_BASE_URL`: optional override for the Responses API URL
- `API_ALLOWED_ORIGINS`: optional comma-separated browser origins allowed to call the API; defaults to local Expo web origins plus `https://*.vercel.app`
- `SUPABASE_URL`: required for validating Supabase access tokens on protected API routes
- `SUPABASE_SECRET_KEY`: required for validating Supabase access tokens on protected API routes

## Parse-label endpoint

`POST /garments/parse-label` expects:

- `Authorization: Bearer <supabase-access-token>`
- `multipart/form-data` with an `image` field

The response shape preserves the garment parsing contract used for the app review step:

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

## OCR-first parser

`LABEL_PARSER_PROVIDER=ocr` runs Tesseract and deterministic text rules only. `LABEL_PARSER_PROVIDER=hybrid` runs the same OCR pass first, then calls Gemini only when OCR text is missing, low confidence, contradictory, or does not include recognizable care-label signals.

Install Tesseract locally before using either OCR-backed mode. On macOS:

```bash
brew install tesseract tesseract-lang
```

The Docker deployment installs Tesseract plus English and Spanish language packs.

## Scan-label Phase 2 path

`POST /scan-label` now runs a Phase 2 pipeline before invoking paid fallback providers:

1. Hash the cropped image bytes with SHA-256 and check the in-memory scan cache.
2. Normalize client detector hints through the laundry-symbol taxonomy.
3. Merge OCR text and symbol detections through deterministic care rules.
4. Route high-confidence local/rules or OCR results without a multimodal fallback.
5. Use Gemini/OpenAI-style multimodal fallback only for ambiguous or conflicting scans.

Responses include field-level `confidence`, `explanation`, and `needs_confirmation`, plus `symbol_detections`, `provider`, `route`, `cache_hit`, `image_hash`, `paid_fallback_used`, `fallback_calls_avoided`, and `routing_reasons`.

The cache stores only the image hash and structured interpretation metadata; it does not retain the uploaded image bytes.

See:

- `docs/laundry-symbol-taxonomy.md`
- `docs/laundry-symbol-dataset.md`
- `models/laundry-symbol-detector/v0.1.0/manifest.json`
