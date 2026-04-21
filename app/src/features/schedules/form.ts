import { CreateSchedulePayload, ScheduleFormValues } from './types';

export const recurrenceOptions = [
  {
    label: 'Daily',
    description: 'Great for high-turnover garments like gym wear or uniforms.',
    value: 'daily',
  },
  {
    label: 'Weekly on Monday',
    description: 'A simple weekly cadence anchored to the start of the week.',
    value: 'weekly:monday',
  },
  {
    label: 'Weekly on Friday',
    description: 'Useful for end-of-week resets like towels or bedding.',
    value: 'weekly:friday',
  },
  {
    label: 'Every two weeks',
    description: 'A lighter cadence for less-frequent laundry jobs.',
    value: 'fortnightly',
  },
] as const;

type ScheduleFormField = 'name' | 'recurrence' | 'startsOn' | 'garmentIds';

export type ScheduleFormErrors = Partial<Record<ScheduleFormField, string>>;

export function createInitialScheduleFormValues(): ScheduleFormValues {
  return {
    name: '',
    recurrence: recurrenceOptions[0].value,
    startsOn: formatLocalDate(new Date()),
    garmentIds: [],
    remindersEnabled: true,
  };
}

export function validateScheduleDraft(values: ScheduleFormValues): ScheduleFormErrors {
  const errors: ScheduleFormErrors = {};
  const trimmedName = values.name.trim();

  if (!trimmedName) {
    errors.name = 'Give this schedule a clear name before saving.';
  }

  if (!recurrenceOptions.some((option) => option.value === values.recurrence)) {
    errors.recurrence = 'Pick one of the supported recurrence options.';
  }

  if (!isValidLocalDate(values.startsOn)) {
    errors.startsOn = 'Use YYYY-MM-DD for the schedule start date.';
  }

  if (values.garmentIds.length === 0) {
    errors.garmentIds = 'Choose at least one garment to track with this schedule.';
  }

  return errors;
}

export function buildCreateSchedulePayload(values: ScheduleFormValues): CreateSchedulePayload {
  return {
    name: values.name.trim(),
    recurrence: values.recurrence,
    starts_on: values.startsOn,
    garment_ids: Array.from(new Set(values.garmentIds)),
    reminders_enabled: values.remindersEnabled,
  };
}

export function formatRecurrenceLabel(recurrence: string) {
  return recurrenceOptions.find((option) => option.value === recurrence)?.label ?? recurrence;
}

export function formatLocalDate(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');

  return `${year}-${month}-${day}`;
}

function isValidLocalDate(value: string) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return false;
  }

  const date = new Date(`${value}T00:00:00`);
  return !Number.isNaN(date.getTime()) && formatLocalDate(date) === value;
}
