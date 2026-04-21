type ReminderOccurrenceOptions = {
  count?: number;
  from?: Date;
  hour?: number;
  minute?: number;
  startDate?: string | Date | null;
};

const WEEKDAY_INDEX: Record<string, number> = {
  sunday: 0,
  monday: 1,
  tuesday: 2,
  wednesday: 3,
  thursday: 4,
  friday: 5,
  saturday: 6,
};

export function computeUpcomingReminderOccurrences(
  recurrence: string,
  options: ReminderOccurrenceOptions = {}
) {
  const count = Math.max(0, options.count ?? 4);
  const from = options.from ?? new Date();
  const hour = options.hour ?? 9;
  const minute = options.minute ?? 0;

  if (count === 0) {
    return [];
  }

  if (recurrence === 'daily') {
    return buildOccurrenceSeries(firstDailyOccurrence(from, hour, minute), count, 1);
  }

  if (recurrence === 'fortnightly') {
    const startDate = normalizeStartDate(options.startDate);
    return buildOccurrenceSeries(firstFortnightlyOccurrence(from, startDate, hour, minute), count, 14);
  }

  if (recurrence.startsWith('weekly:')) {
    const weekday = parseWeeklyRecurrence(recurrence);

    if (weekday === null) {
      throw new Error('invalid-recurrence');
    }

    return buildOccurrenceSeries(firstWeeklyOccurrence(from, weekday, hour, minute), count, 7);
  }

  throw new Error('invalid-recurrence');
}

function firstDailyOccurrence(from: Date, hour: number, minute: number) {
  const candidate = atLocalTime(from, hour, minute);

  if (candidate.getTime() <= from.getTime()) {
    candidate.setDate(candidate.getDate() + 1);
  }

  return candidate;
}

function firstWeeklyOccurrence(from: Date, weekday: number, hour: number, minute: number) {
  const candidate = atLocalTime(from, hour, minute);
  const daysUntilTarget = (weekday - candidate.getDay() + 7) % 7;
  candidate.setDate(candidate.getDate() + daysUntilTarget);

  if (candidate.getTime() <= from.getTime()) {
    candidate.setDate(candidate.getDate() + 7);
  }

  return candidate;
}

function firstFortnightlyOccurrence(from: Date, startDate: Date, hour: number, minute: number) {
  const candidate = atLocalTime(startDate, hour, minute);

  while (candidate.getTime() <= from.getTime()) {
    candidate.setDate(candidate.getDate() + 14);
  }

  return candidate;
}

function buildOccurrenceSeries(firstOccurrence: Date, count: number, intervalDays: number) {
  return Array.from({ length: count }, (_unused, index) => {
    const occurrence = new Date(firstOccurrence);
    occurrence.setDate(firstOccurrence.getDate() + index * intervalDays);
    return occurrence;
  });
}

function atLocalTime(from: Date, hour: number, minute: number) {
  const candidate = new Date(from);
  candidate.setHours(hour, minute, 0, 0);
  return candidate;
}

function parseWeeklyRecurrence(recurrence: string) {
  const weekdayName = recurrence.split(':')[1]?.trim().toLowerCase() ?? '';
  return WEEKDAY_INDEX[weekdayName] ?? null;
}

function normalizeStartDate(value: string | Date | null | undefined) {
  if (!value) {
    throw new Error('start-date-required');
  }

  const date = typeof value === 'string' ? new Date(`${value}T00:00:00`) : new Date(value);
  if (Number.isNaN(date.getTime())) {
    throw new Error('invalid-start-date');
  }

  return date;
}
