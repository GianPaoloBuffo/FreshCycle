import { getAppEnv } from '@/lib/env';

import { CreateSchedulePayload, LaundrySchedule } from './types';

type SaveScheduleDeps = {
  accessToken?: string | null;
  apiBaseUrl?: string | null;
  fetchImpl?: typeof fetch;
};

export async function saveSchedule(payload: CreateSchedulePayload, deps: SaveScheduleDeps = {}) {
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
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error(await extractSaveErrorCode(response));
  }

  return (await response.json()) as LaundrySchedule;
}

async function extractSaveErrorCode(response: Response) {
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
      case 'schedule_name_required':
        return 'name-required';
      case 'garment_ids_required':
        return 'garments-required';
      case 'invalid_garment_id':
        return 'invalid-garment-id';
      case 'invalid_recurrence':
        return 'recurrence-invalid';
      case 'invalid_start_date':
        return 'start-date-invalid';
      case 'route_not_found':
        return 'not-ready';
      default:
        return 'save-failed';
    }
  } catch {
    return 'save-failed';
  }
}
