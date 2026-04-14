import { describe, expect, it, vi } from 'vitest';

import { fetchGarments } from './fetchGarments';

describe('fetchGarments', () => {
  it('requests garments with bearer auth', async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve([
          {
            id: 'garment-123',
            user_id: 'user-123',
            name: 'Navy Hoodie',
            category: 'Knitwear',
            primary_color: 'Navy',
            wash_temperature_c: 30,
            care_instructions: ['Machine washable'],
            label_image_path: null,
          },
        ]),
    });

    const result = await fetchGarments({
      accessToken: 'token-123',
      apiBaseUrl: 'https://api.example.com',
      fetchImpl: fetchImpl as never,
    });

    expect(fetchImpl).toHaveBeenCalledWith('https://api.example.com/garments', {
      method: 'GET',
      headers: {
        Authorization: 'Bearer token-123',
      },
    });
    expect(result).toHaveLength(1);
    expect(result[0]?.name).toBe('Navy Hoodie');
  });

  it('maps auth failures to a stable wardrobe error code', async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: false,
      json: () =>
        Promise.resolve({
          error: 'auth_required',
        }),
    });

    await expect(
      fetchGarments({
        accessToken: 'token-123',
        apiBaseUrl: 'https://api.example.com',
        fetchImpl: fetchImpl as never,
      })
    ).rejects.toThrow('auth-required');
  });

  it('treats expired sessions as auth-required', async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: () =>
        Promise.resolve({
          error: 'invalid_token',
        }),
    });

    await expect(
      fetchGarments({
        accessToken: 'token-123',
        apiBaseUrl: 'https://api.example.com',
        fetchImpl: fetchImpl as never,
      })
    ).rejects.toThrow('auth-required');
  });
});
