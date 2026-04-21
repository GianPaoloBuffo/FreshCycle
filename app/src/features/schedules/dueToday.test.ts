import { describe, expect, it } from 'vitest';

import { getSchedulesDueToday, isScheduleDueToday } from './dueToday';
import { LaundrySchedule } from './types';

const baseSchedule: LaundrySchedule = {
  id: 'schedule-123',
  user_id: 'user-123',
  name: 'Weekly towels',
  recurrence: 'daily',
  starts_on: '2026-04-06',
  garment_ids: ['garment-1'],
  reminders_enabled: true,
  created_at: '2026-04-06T09:00:00.000Z',
};

describe('isScheduleDueToday', () => {
  it('treats daily schedules as due every day', () => {
    expect(isScheduleDueToday(baseSchedule, new Date('2026-04-21T12:00:00'))).toBe(true);
  });

  it('matches weekly schedules by local weekday', () => {
    expect(
      isScheduleDueToday(
        {
          ...baseSchedule,
          recurrence: 'weekly:tuesday',
        },
        new Date('2026-04-21T12:00:00')
      )
    ).toBe(true);
  });

  it('uses starts_on as the fortnightly anchor', () => {
    expect(
      isScheduleDueToday(
        {
          ...baseSchedule,
          recurrence: 'fortnightly',
          starts_on: '2026-04-07',
        },
        new Date('2026-04-21T12:00:00')
      )
    ).toBe(true);
  });

  it('ignores unsupported recurrence strings', () => {
    expect(
      isScheduleDueToday(
        {
          ...baseSchedule,
          recurrence: 'monthly',
        },
        new Date('2026-04-21T12:00:00')
      )
    ).toBe(false);
  });
});

describe('getSchedulesDueToday', () => {
  it('returns due schedules sorted by name', () => {
    const dueSchedules = getSchedulesDueToday(
      [
        {
          ...baseSchedule,
          id: 'b',
          name: 'Zebra bedding',
        },
        {
          ...baseSchedule,
          id: 'a',
          name: 'Aprons',
        },
        {
          ...baseSchedule,
          id: 'c',
          name: 'Friday towels',
          recurrence: 'weekly:friday',
        },
      ],
      new Date('2026-04-21T12:00:00')
    );

    expect(dueSchedules.map((schedule) => schedule.name)).toEqual(['Aprons', 'Zebra bedding']);
  });
});
