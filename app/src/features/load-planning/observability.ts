type LoadPlanningEventName =
  | 'load_planning_mode_selected'
  | 'load_planning_generated'
  | 'load_planning_generation_failed'
  | 'load_detail_opened'
  | 'load_detail_failed';

type EventPayload = Record<string, string | number | boolean | null | undefined>;

export function logLoadPlanningEvent(event: LoadPlanningEventName, payload: EventPayload = {}) {
  const timestamp = new Date().toISOString();
  console.info(`[freshcycle.load-planning] ${event}`, {
    timestamp,
    ...payload,
  });
}

export function logLoadPlanningError(
  event: LoadPlanningEventName,
  error: unknown,
  payload: EventPayload = {}
) {
  const timestamp = new Date().toISOString();
  console.error(`[freshcycle.load-planning] ${event}`, {
    timestamp,
    message: error instanceof Error ? error.message : String(error),
    ...payload,
  });
}
