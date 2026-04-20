import { supabase } from '@/lib/supabase';

import { logNotificationError, logNotificationEvent } from './observability';

type SavePushTokenDeps = {
  dbClient?: Pick<typeof supabase, 'from'>;
};

export async function savePushToken(
  userId: string | null | undefined,
  pushToken: string | null,
  deps: SavePushTokenDeps = {}
) {
  const dbClient = deps.dbClient ?? supabase;
  const normalizedPushToken = pushToken?.trim() ?? null;

  if (!userId) {
    throw new Error('auth-required');
  }

  if (normalizedPushToken === '') {
    throw new Error('invalid-push-token');
  }

  logNotificationEvent('push_token_persist_started', {
    hasPushToken: normalizedPushToken !== null,
  });

  const { error } = await dbClient.from('user_profiles').upsert(
    {
      id: userId,
      push_token: normalizedPushToken,
    },
    {
      onConflict: 'id',
    }
  );

  if (error) {
    logNotificationError('push_token_persist_failed', error, {
      hasPushToken: normalizedPushToken !== null,
    });
    throw new Error('push-token-save-failed');
  }

  logNotificationEvent('push_token_persist_succeeded', {
    hasPushToken: normalizedPushToken !== null,
  });
}
