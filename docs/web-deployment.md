# Web Deployment

FreshCycle can now be exported as a static web app from [`/app`](/Users/gp-macbook/Projects/FreshCycle/app).

## Recommended hosted setup

For early public testing, use:

- Hosted Supabase for auth, database, and storage
- Vercel for the Expo web frontend

This gives FreshCycle a public browser URL without depending on a laptop-bound local stack.

## Why this split

- Supabase gives us a managed backend with a free tier that is suitable for early testing
- Vercel is a simple free-tier option for static web hosting
- The Expo app already exports a static browser build with `expo export --platform web`

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
