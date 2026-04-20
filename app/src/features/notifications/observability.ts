type NotificationEventName =
  | 'push_token_persist_started'
  | 'push_token_persist_succeeded'
  | 'push_token_persist_failed';

type EventPayload = Record<string, string | number | boolean | null | undefined>;

export function logNotificationEvent(event: NotificationEventName, payload: EventPayload = {}) {
  const timestamp = new Date().toISOString();
  console.info(`[freshcycle.notifications] ${event}`, {
    timestamp,
    ...payload,
  });
}

export function logNotificationError(
  event: NotificationEventName,
  error: unknown,
  payload: EventPayload = {}
) {
  const timestamp = new Date().toISOString();
  console.error(`[freshcycle.notifications] ${event}`, {
    timestamp,
    message: error instanceof Error ? error.message : String(error),
    ...payload,
  });
}
