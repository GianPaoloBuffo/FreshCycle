type WardrobeEventName =
  | 'wardrobe_fetch_started'
  | 'wardrobe_fetch_succeeded'
  | 'wardrobe_fetch_failed'
  | 'wardrobe_refresh_started'
  | 'wardrobe_refresh_succeeded'
  | 'wardrobe_refresh_failed';

type EventPayload = Record<string, string | number | boolean | null | undefined>;

export function logWardrobeEvent(event: WardrobeEventName, payload: EventPayload = {}) {
  const timestamp = new Date().toISOString();
  console.info(`[freshcycle.wardrobe] ${event}`, {
    timestamp,
    ...payload,
  });
}

export function logWardrobeError(event: WardrobeEventName, error: unknown, payload: EventPayload = {}) {
  const timestamp = new Date().toISOString();
  console.error(`[freshcycle.wardrobe] ${event}`, {
    timestamp,
    message: error instanceof Error ? error.message : String(error),
    ...payload,
  });
}
