import { describe, expect, it, vi } from 'vitest';

import { fetchSchedules } from './fetchSchedules';

describe('fetchSchedules', () => {
  it('requests schedules with bearer auth', async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve([
          {
            id: 'schedule-123',
            user_id: 'user-123',
            name: 'Weekly towels',
            recurrence: 'weekly:monday',
            garment_ids: ['garment-1', 'garment-2'],
            reminders_enabled: true,
            created_at: '2026-04-20T10:00:00.000Z',
          },
        ]),
    });

    const result = await fetchSchedules({
      accessToken: 'token-123',
      apiBaseUrl: 'https://api.example.com',
      fetchImpl: fetchImpl as never,
    });

    expect(fetchImpl).toHaveBeenCalledWith('https://api.example.com/schedules', {
      method: 'GET',
      headers: {
        Authorization: 'Bearer token-123',
      },
    });
    expect(result[0]?.name).toBe('Weekly towels');
  });

  it('maps auth failures to auth-required', async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: () =>
        Promise.resolve({
          error: 'invalid_token',
        }),
    });

    await expect(
      fetchSchedules({
        accessToken: 'token-123',
        apiBaseUrl: 'https://api.example.com',
        fetchImpl: fetchImpl as never,
      })
    ).rejects.toThrow('auth-required');
  });

  it('maps missing schedules endpoints to not-ready', async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      json: () =>
        Promise.resolve({
          error: 'route_not_found',
        }),
    });

    await expect(
      fetchSchedules({
        accessToken: 'token-123',
        apiBaseUrl: 'https://api.example.com',
        fetchImpl: fetchImpl as never,
      })
    ).rejects.toThrow('not-ready');
  });
});
