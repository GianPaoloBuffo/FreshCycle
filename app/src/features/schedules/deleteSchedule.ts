import { getAppEnv } from '@/lib/env';

type DeleteScheduleDeps = {
  accessToken?: string | null;
  apiBaseUrl?: string | null;
  fetchImpl?: typeof fetch;
};

export async function deleteSchedule(scheduleId: string, deps: DeleteScheduleDeps = {}) {
  const accessToken = deps.accessToken ?? null;
  const apiBaseUrl = deps.apiBaseUrl ?? getAppEnv().apiBaseUrl;
  const fetchImpl = deps.fetchImpl ?? ((input, init) => fetch(input, init));

  if (!accessToken) {
    throw new Error('auth-required');
  }

  if (!apiBaseUrl) {
    throw new Error('api-unavailable');
  }

  const response = await fetchImpl(
    `${apiBaseUrl.replace(/\/$/, '')}/schedules/${encodeURIComponent(scheduleId)}`,
    {
      method: 'DELETE',
      headers: {
        Authorization: `Bearer ${accessToken}`,
      },
    }
  );

  if (!response.ok) {
    throw new Error(await extractDeleteErrorCode(response));
  }
}

async function extractDeleteErrorCode(response: Response) {
  if (response.status === 401) {
    return 'auth-required';
  }

  if (response.status === 404) {
    return 'schedule-not-found';
  }

  try {
    const body = (await response.json()) as { error?: string };

    switch (body.error) {
      case 'auth_required':
      case 'invalid_token':
        return 'auth-required';
      case 'invalid_schedule_id':
        return 'invalid-schedule-id';
      case 'schedule_not_found':
        return 'schedule-not-found';
      case 'route_not_found':
        return 'not-ready';
      default:
        return 'delete-failed';
    }
  } catch {
    return 'delete-failed';
  }
}
