type NotificationEventName =
  | 'notification_handler_configured'
  | 'notification_permission_denied'
  | 'push_token_capture_started'
  | 'push_token_capture_succeeded'
  | 'push_token_capture_failed'
  | 'push_token_persist_started'
  | 'push_token_persist_succeeded'
  | 'push_token_persist_failed'
  | 'local_notifications_cancel_started'
  | 'local_notifications_cancel_succeeded'
  | 'local_notifications_cancel_failed'
  | 'local_notifications_schedule_started'
  | 'local_notifications_schedule_succeeded'
  | 'local_notifications_schedule_failed';

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
