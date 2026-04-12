# Web Deployment

FreshCycle can now be exported as a static web app from [`/app`](/Users/gp-macbook/Projects/FreshCycle/app).

## Recommended hosted setup

For early public testing, use:

- Hosted Supabase for auth, database, and storage
- Vercel for the Expo web frontend
- Render for the Go API

This gives FreshCycle a public browser URL without depending on a laptop-bound local stack, while keeping the standalone Go API easy to deploy and inspect.

## Why this split

- Supabase gives us a managed backend with a free tier that is suitable for early testing
- Vercel is a simple free-tier option for static web hosting
- Render offers a straightforward public web service for a small Go server
- The Expo app already exports a static browser build with `expo export --platform web`
- The Go API already expects a long-running HTTP server, which maps neatly to a Render web service

## Commands

From the repo root:

```bash
make app-web
make app-web-export
```

From [`/app`](/Users/gp-macbook/Projects/FreshCycle/app):

```bash
npm run web
npm run export:web
```

## Required app environment variables

Set these for browser builds:

- `EXPO_PUBLIC_API_BASE_URL`
- `EXPO_PUBLIC_SUPABASE_URL`
- `EXPO_PUBLIC_SUPABASE_KEY`
- `EXPO_PUBLIC_WEB_AUTH_REDIRECT_URL`

Keep this value too if you also build native apps from the same repo:

- `EXPO_PUBLIC_NATIVE_AUTH_REDIRECT_URL`

Example hosted values:

```bash
EXPO_PUBLIC_API_BASE_URL=https://api.example.com
EXPO_PUBLIC_SUPABASE_URL=https://your-project-ref.supabase.co
EXPO_PUBLIC_SUPABASE_KEY=your-supabase-publishable-key
EXPO_PUBLIC_NATIVE_AUTH_REDIRECT_URL=freshcycle://auth/callback
EXPO_PUBLIC_WEB_AUTH_REDIRECT_URL=https://freshcycle-web.vercel.app/auth
```

## Vercel setup

FreshCycle includes [`app/vercel.json`](/Users/gp-macbook/Projects/FreshCycle/app/vercel.json#L1) with the expected build command and output directory.

Recommended Vercel project settings:

1. Import the repository into Vercel.
2. Set the root directory to `app`.
3. Use `npm install` as the install command.
4. Use `npm run export:web` as the build command.
5. Use `dist` as the output directory if Vercel does not detect it automatically.
6. Add the `EXPO_PUBLIC_*` environment variables to Preview and Production.

## Recommended API hosting

For FreshCycle's current architecture, Render is the simplest fit for the Go API. Render's web services are designed for public HTTP apps, which aligns with the API's current long-running `chi` server model. Source: [Render service types](https://render.com/docs/service-types).

Vercel is still the right home for the Expo web frontend, but the current project is configured only as a static frontend build. There is not yet a separate hosted API project in Vercel.

FreshCycle now includes [render.yaml](/Users/gp-macbook/Projects/FreshCycle/render.yaml), so you can create the API service from the repo without hand-entering every setting.

Recommended Render steps:

1. Create a new Blueprint or Web Service from this repository.
2. Use the `freshcycle-api` service defined in [render.yaml](/Users/gp-macbook/Projects/FreshCycle/render.yaml).
3. Set `SUPABASE_DB_URL`, `SUPABASE_URL`, and `SUPABASE_SECRET_KEY` in Render.
4. Keep `LABEL_PARSER_PROVIDER=stub` for the first smoke test.
5. Once the service is live, copy its URL into Vercel as `EXPO_PUBLIC_API_BASE_URL`.
6. Redeploy the Vercel app.
7. Later, switch to `LABEL_PARSER_PROVIDER=openai` and add `OPENAI_API_KEY` when you're ready for real parsing.

The API now includes CORS support for local Expo web and `*.vercel.app` origins, so browser calls from Vercel previews and production can reach the hosted parser endpoint.

## Supabase hosted project setup

When moving from the local CLI stack to a publicly reachable browser environment:

1. Create or choose a hosted Supabase project.
2. Apply the checked-in schema from [`/supabase`](/Users/gp-macbook/Projects/FreshCycle/supabase).
3. Add your web app URL to the Supabase Auth redirect allow-list.
4. Use the hosted project URL and publishable key in the web app environment.

## Current browser limitations

- The current browser flow supports file upload for garment label selection
- Native camera capture still belongs to the mobile app path
- A public web deployment still needs public backend services behind it
