import { describe, expect, it, vi } from 'vitest';

vi.mock('@/lib/supabase', () => ({
  supabase: {
    from: vi.fn(),
  },
}));

import { savePushToken } from './savePushToken';

describe('savePushToken', () => {
  it('upserts the authenticated user profile token', async () => {
    const upsert = vi.fn().mockResolvedValue({
      error: null,
    });
    const from = vi.fn().mockReturnValue({
      upsert,
    });

    await savePushToken('user-123', 'ExponentPushToken[abc]', {
      dbClient: { from } as never,
    });

    expect(from).toHaveBeenCalledWith('user_profiles');
    expect(upsert).toHaveBeenCalledWith(
      {
        id: 'user-123',
        push_token: 'ExponentPushToken[abc]',
      },
      {
        onConflict: 'id',
      }
    );
  });

  it('allows clearing an existing token by saving null', async () => {
    const upsert = vi.fn().mockResolvedValue({
      error: null,
    });
    const from = vi.fn().mockReturnValue({
      upsert,
    });

    await savePushToken('user-123', null, {
      dbClient: { from } as never,
    });

    expect(upsert).toHaveBeenCalledWith(
      {
        id: 'user-123',
        push_token: null,
      },
      {
        onConflict: 'id',
      }
    );
  });

  it('rejects blank push tokens', async () => {
    await expect(savePushToken('user-123', '   ')).rejects.toThrow('invalid-push-token');
  });

  it('surfaces persistence failures as a stable error code', async () => {
    const upsert = vi.fn().mockResolvedValue({
      error: { message: 'write rejected' },
    });
    const from = vi.fn().mockReturnValue({
      upsert,
    });

    await expect(
      savePushToken('user-123', 'ExponentPushToken[abc]', {
        dbClient: { from } as never,
      })
    ).rejects.toThrow('push-token-save-failed');
  });
});
