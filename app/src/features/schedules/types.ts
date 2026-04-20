export type LaundrySchedule = {
  id: string;
  user_id: string;
  name: string;
  recurrence: string;
  garment_ids: string[];
  reminders_enabled: boolean;
  created_at: string;
};

export type CreateSchedulePayload = {
  name: string;
  recurrence: string;
  garment_ids: string[];
  reminders_enabled: boolean;
};

export type ScheduleFormValues = {
  name: string;
  recurrence: string;
  garmentIds: string[];
  remindersEnabled: boolean;
};

export type SchedulesViewState =
  | 'setup_required'
  | 'auth_loading'
  | 'signed_out'
  | 'loading'
  | 'error'
  | 'empty'
  | 'ready';
