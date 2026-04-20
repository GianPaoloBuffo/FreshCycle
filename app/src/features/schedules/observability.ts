type SchedulesEventName =
  | 'schedules_fetch_started'
  | 'schedules_fetch_succeeded'
  | 'schedules_fetch_failed'
  | 'schedules_navigation_opened'
  | 'new_schedule_opened'
  | 'new_schedule_garments_fetch_started'
  | 'new_schedule_garments_fetch_succeeded'
  | 'new_schedule_garments_fetch_failed'
  | 'new_schedule_validation_failed'
  | 'new_schedule_submission_started'
  | 'new_schedule_submission_succeeded'
  | 'new_schedule_submission_failed';

type EventPayload = Record<string, string | number | boolean | null | undefined>;

export function logSchedulesEvent(event: SchedulesEventName, payload: EventPayload = {}) {
  const timestamp = new Date().toISOString();
  console.info(`[freshcycle.schedules] ${event}`, {
    timestamp,
    ...payload,
  });
}

export function logSchedulesError(
  event: SchedulesEventName,
  error: unknown,
  payload: EventPayload = {}
) {
  const timestamp = new Date().toISOString();
  console.error(`[freshcycle.schedules] ${event}`, {
    timestamp,
    message: error instanceof Error ? error.message : String(error),
    ...payload,
  });
}
