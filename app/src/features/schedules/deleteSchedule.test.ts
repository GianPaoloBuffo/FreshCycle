import { describe, expect, it, vi } from 'vitest';

import { deleteSchedule } from './deleteSchedule';

describe('deleteSchedule', () => {
  it('deletes a schedule with bearer auth', async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: true,
    });

    await deleteSchedule('schedule-123', {
      accessToken: 'token-123',
      apiBaseUrl: 'https://api.example.com',
      fetchImpl: fetchImpl as never,
    });

    expect(fetchImpl).toHaveBeenCalledWith('https://api.example.com/schedules/schedule-123', {
      method: 'DELETE',
      headers: {
        Authorization: 'Bearer token-123',
      },
    });
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
      deleteSchedule('schedule-123', {
        accessToken: 'token-123',
        apiBaseUrl: 'https://api.example.com',
        fetchImpl: fetchImpl as never,
      })
    ).rejects.toThrow('auth-required');
  });

  it('maps not-found responses to schedule-not-found', async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      json: () =>
        Promise.resolve({
          error: 'schedule_not_found',
        }),
    });

    await expect(
      deleteSchedule('schedule-123', {
        accessToken: 'token-123',
        apiBaseUrl: 'https://api.example.com',
        fetchImpl: fetchImpl as never,
      })
    ).rejects.toThrow('schedule-not-found');
  });
});
