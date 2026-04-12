import { describe, expect, it, vi } from 'vitest';

import { saveGarment } from './saveGarment';

describe('saveGarment', () => {
  const payload = {
    id: '29ce43cd-f095-476d-a7cb-1ee7850c14f1',
    name: 'Navy Hoodie',
    category: 'Knitwear',
    primary_color: 'Navy',
    wash_temperature_c: 30,
    care_instructions: ['Machine washable'],
    label_image_path: 'user-123/labels/29ce43cd-f095-476d-a7cb-1ee7850c14f1.jpg',
  };

  it('posts the garment payload with bearer auth', async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          ...payload,
          user_id: 'user-123',
        }),
    });

    const result = await saveGarment(payload, {
      accessToken: 'token-123',
      apiBaseUrl: 'https://api.example.com',
      fetchImpl: fetchImpl as never,
    });

    expect(fetchImpl).toHaveBeenCalledWith(
      'https://api.example.com/garments',
      expect.objectContaining({
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: 'Bearer token-123',
        },
      })
    );
    expect(result.label_image_path).toBe(payload.label_image_path);
  });

  it('maps known API validation errors into add-garment error codes', async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: false,
      json: () =>
        Promise.resolve({
          error: 'invalid_label_image_path',
        }),
    });

    await expect(
      saveGarment(payload, {
        accessToken: 'token-123',
        apiBaseUrl: 'https://api.example.com',
        fetchImpl: fetchImpl as never,
      })
    ).rejects.toThrow('invalid-label-image-path');
  });
});
