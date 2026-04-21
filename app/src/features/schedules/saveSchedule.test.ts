import { describe, expect, it, vi } from 'vitest';

import { saveSchedule } from './saveSchedule';

describe('saveSchedule', () => {
  const payload = {
    name: 'Weekly towels',
    recurrence: 'weekly:monday',
    starts_on: '2026-04-20',
    garment_ids: ['garment-1', 'garment-2'],
    reminders_enabled: true,
  };

  it('posts the schedule payload with bearer auth', async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          id: 'schedule-123',
          user_id: 'user-123',
          created_at: '2026-04-20T09:00:00.000Z',
          ...payload,
        }),
    });

    const result = await saveSchedule(payload, {
      accessToken: 'token-123',
      apiBaseUrl: 'https://api.example.com',
      fetchImpl: fetchImpl as never,
    });

    expect(fetchImpl).toHaveBeenCalledWith(
      'https://api.example.com/schedules',
      expect.objectContaining({
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: 'Bearer token-123',
        },
      })
    );
    expect(result.name).toBe('Weekly towels');
  });

  it('maps validation errors into stable schedule error codes', async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: false,
      json: () =>
        Promise.resolve({
          error: 'invalid_recurrence',
        }),
    });

    await expect(
      saveSchedule(payload, {
        accessToken: 'token-123',
        apiBaseUrl: 'https://api.example.com',
        fetchImpl: fetchImpl as never,
      })
    ).rejects.toThrow('recurrence-invalid');
  });

  it('maps missing endpoints to not-ready', async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      json: () =>
        Promise.resolve({
          error: 'route_not_found',
        }),
    });

    await expect(
      saveSchedule(payload, {
        accessToken: 'token-123',
        apiBaseUrl: 'https://api.example.com',
        fetchImpl: fetchImpl as never,
      })
    ).rejects.toThrow('not-ready');
  });
});
