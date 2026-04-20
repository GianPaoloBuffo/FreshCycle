import { getAppEnv } from '@/lib/env';

import { LaundrySchedule } from './types';

type FetchSchedulesDeps = {
  accessToken?: string | null;
  apiBaseUrl?: string | null;
  fetchImpl?: typeof fetch;
};

export async function fetchSchedules(deps: FetchSchedulesDeps = {}) {
  const accessToken = deps.accessToken ?? null;
  const apiBaseUrl = deps.apiBaseUrl ?? getAppEnv().apiBaseUrl;
  const fetchImpl = deps.fetchImpl ?? ((input, init) => fetch(input, init));

  if (!accessToken) {
    throw new Error('auth-required');
  }

  if (!apiBaseUrl) {
    throw new Error('api-unavailable');
  }

  const response = await fetchImpl(`${apiBaseUrl.replace(/\/$/, '')}/schedules`, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  });

  if (!response.ok) {
    throw new Error(await extractFetchErrorCode(response));
  }

  return (await response.json()) as LaundrySchedule[];
}

async function extractFetchErrorCode(response: Response) {
  if (response.status === 401) {
    return 'auth-required';
  }

  if (response.status === 404) {
    return 'not-ready';
  }

  try {
    const body = (await response.json()) as { error?: string };

    switch (body.error) {
      case 'auth_required':
      case 'invalid_token':
        return 'auth-required';
      case 'route_not_found':
        return 'not-ready';
      default:
        return 'fetch-failed';
    }
  } catch {
    return 'fetch-failed';
  }
}
