export type AppEnv = {
  apiBaseUrl: string | null;
  supabaseUrl: string | null;
  supabaseKey: string | null;
  authRedirectUrl: string | null;
  nativeAuthRedirectUrl: string | null;
  webAuthRedirectUrl: string | null;
};

export function getAppEnv(): AppEnv {
  return {
    apiBaseUrl: process.env.EXPO_PUBLIC_API_BASE_URL ?? null,
    supabaseUrl: process.env.EXPO_PUBLIC_SUPABASE_URL ?? null,
    supabaseKey: process.env.EXPO_PUBLIC_SUPABASE_KEY ?? null,
    authRedirectUrl: process.env.EXPO_PUBLIC_AUTH_REDIRECT_URL ?? null,
    nativeAuthRedirectUrl: process.env.EXPO_PUBLIC_NATIVE_AUTH_REDIRECT_URL ?? null,
    webAuthRedirectUrl: process.env.EXPO_PUBLIC_WEB_AUTH_REDIRECT_URL ?? null,
  };
}

export function getAuthRedirectUrl(platform: 'native' | 'web') {
  const env = getAppEnv();

  if (platform === 'web') {
    if (env.webAuthRedirectUrl) {
      return env.webAuthRedirectUrl;
    }

    if (env.authRedirectUrl?.startsWith('http')) {
      return env.authRedirectUrl;
    }

    if (typeof window !== 'undefined' && window.location.origin) {
      return `${window.location.origin}/auth`;
    }

    return null;
  }

  return env.nativeAuthRedirectUrl ?? env.authRedirectUrl;
}
