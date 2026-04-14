import { getAppEnv } from '@/lib/env';

import { WardrobeGarment } from './types';

type FetchGarmentsDeps = {
  accessToken?: string | null;
  apiBaseUrl?: string | null;
  fetchImpl?: typeof fetch;
};

export async function fetchGarments(deps: FetchGarmentsDeps = {}) {
  const accessToken = deps.accessToken ?? null;
  const apiBaseUrl = deps.apiBaseUrl ?? getAppEnv().apiBaseUrl;
  const fetchImpl = deps.fetchImpl ?? ((input, init) => fetch(input, init));

  if (!accessToken) {
    throw new Error('auth-required');
  }

  if (!apiBaseUrl) {
    throw new Error('api-unavailable');
  }

  const response = await fetchImpl(`${apiBaseUrl.replace(/\/$/, '')}/garments`, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  });

  if (!response.ok) {
    throw new Error(await extractFetchErrorCode(response));
  }

  return (await response.json()) as WardrobeGarment[];
}

async function extractFetchErrorCode(response: Response) {
  if (response.status === 401) {
    return 'auth-required';
  }

  try {
    const body = (await response.json()) as { error?: string };

    switch (body.error) {
      case 'auth_required':
      case 'invalid_token':
        return 'auth-required';
      default:
        return 'fetch-failed';
    }
  } catch {
    return 'fetch-failed';
  }
}
