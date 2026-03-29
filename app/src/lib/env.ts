export type AppEnv = {
  apiBaseUrl: string | null;
  supabaseUrl: string | null;
  supabaseKey: string | null;
  authRedirectUrl: string | null;
};

export function getAppEnv(): AppEnv {
  return {
    apiBaseUrl: process.env.EXPO_PUBLIC_API_BASE_URL ?? null,
    supabaseUrl: process.env.EXPO_PUBLIC_SUPABASE_URL ?? null,
    supabaseKey: process.env.EXPO_PUBLIC_SUPABASE_KEY ?? null,
    authRedirectUrl: process.env.EXPO_PUBLIC_AUTH_REDIRECT_URL ?? null,
  };
}
