type AddGarmentEventName =
  | 'label_selection_started'
  | 'label_selection_completed'
  | 'label_selection_failed'
  | 'label_parse_started'
  | 'label_parse_succeeded'
  | 'label_parse_failed'
  | 'label_upload_started'
  | 'label_upload_succeeded'
  | 'label_upload_failed'
  | 'review_form_submitted'
  | 'review_form_failed'
  | 'scan_review_event_recorded'
  | 'scan_review_event_failed'
  | 'garment_save_succeeded'
  | 'garment_save_failed';

type EventPayload = Record<string, unknown>;

export function logAddGarmentEvent(event: AddGarmentEventName, payload: EventPayload = {}) {
  const timestamp = new Date().toISOString();
  console.info(`[freshcycle.add-garment] ${event}`, {
    timestamp,
    ...payload,
  });
}

export function logAddGarmentError(event: AddGarmentEventName, error: unknown, payload: EventPayload = {}) {
  const timestamp = new Date().toISOString();
  console.error(`[freshcycle.add-garment] ${event}`, {
    timestamp,
    message: error instanceof Error ? error.message : String(error),
    ...payload,
  });
}
