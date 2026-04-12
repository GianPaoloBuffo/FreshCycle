import { describe, expect, it, vi } from 'vitest';

vi.mock('@/lib/supabase', () => ({
  supabase: {
    storage: {
      from: vi.fn(),
    },
  },
}));

import { createSignedLabelImageUrl, uploadLabelImage } from './uploadLabelImage';

describe('uploadLabelImage', () => {
  it('uploads to the authenticated user label folder and returns the object path', async () => {
    const upload = vi.fn().mockResolvedValue({ error: null });
    const from = vi.fn().mockReturnValue({
      upload,
    });

    const result = await uploadLabelImage(
      {
        uri: 'https://example.com/label.jpeg',
        fileName: 'label.jpeg',
        mimeType: 'image/jpeg',
        width: 900,
        height: 1200,
        fileSize: 2048,
        webFile: new Blob(['label-bytes'], { type: 'image/jpeg' }),
        source: 'library',
      },
      'user-123',
      {
        garmentId: 'garment-456',
        storageClient: { from } as never,
      }
    );

    expect(from).toHaveBeenCalledWith('garment-labels');
    expect(upload).toHaveBeenCalledWith(
      'user-123/labels/garment-456.jpeg',
      expect.any(ArrayBuffer),
      expect.objectContaining({
        contentType: 'image/jpeg',
        upsert: false,
      })
    );
    expect(result).toEqual({
      garmentId: 'garment-456',
      objectPath: 'user-123/labels/garment-456.jpeg',
    });
  });

  it('falls back to reading the image uri when a web file is unavailable', async () => {
    const upload = vi.fn().mockResolvedValue({ error: null });
    const from = vi.fn().mockReturnValue({
      upload,
    });
    const fetchImpl = vi.fn().mockResolvedValue({
      ok: true,
      arrayBuffer: () => Promise.resolve(new Uint8Array([1, 2, 3]).buffer),
    });

    await uploadLabelImage(
      {
        uri: 'https://example.com/label.png',
        fileName: 'label.png',
        mimeType: 'image/png',
        width: 900,
        height: 1200,
        fileSize: 2048,
        source: 'library',
      },
      'user-123',
      {
        garmentId: 'garment-789',
        fetchImpl: fetchImpl as never,
        storageClient: { from } as never,
      }
    );

    expect(fetchImpl).toHaveBeenCalledWith('https://example.com/label.png');
  });
});

describe('createSignedLabelImageUrl', () => {
  it('returns the signed url from private storage', async () => {
    const createSignedUrl = vi.fn().mockResolvedValue({
      data: { signedUrl: 'https://example.com/signed' },
      error: null,
    });
    const from = vi.fn().mockReturnValue({
      createSignedUrl,
    });

    const signedUrl = await createSignedLabelImageUrl(
      'user-123/labels/garment-456.jpeg',
      1800,
      { from } as never
    );

    expect(from).toHaveBeenCalledWith('garment-labels');
    expect(createSignedUrl).toHaveBeenCalledWith('user-123/labels/garment-456.jpeg', 1800);
    expect(signedUrl).toBe('https://example.com/signed');
  });
});
