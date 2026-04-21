import { describe, expect, it } from 'vitest';

import { computeUpcomingReminderOccurrences } from './recurrence';

describe('computeUpcomingReminderOccurrences', () => {
  it('computes daily occurrences from the next local reminder time', () => {
    const occurrences = computeUpcomingReminderOccurrences('daily', {
      count: 3,
      from: new Date('2026-04-20T08:00:00'),
      hour: 9,
      minute: 30,
    });

    expect(occurrences.map((date) => date.toISOString())).toEqual([
      new Date('2026-04-20T09:30:00').toISOString(),
      new Date('2026-04-21T09:30:00').toISOString(),
      new Date('2026-04-22T09:30:00').toISOString(),
    ]);
  });

  it('rolls daily occurrences to tomorrow when today already passed', () => {
    const occurrences = computeUpcomingReminderOccurrences('daily', {
      count: 1,
      from: new Date('2026-04-20T10:00:00'),
      hour: 9,
      minute: 0,
    });

    expect(occurrences[0]?.toISOString()).toBe(new Date('2026-04-21T09:00:00').toISOString());
  });

  it('computes weekly occurrences for the requested weekday', () => {
    const occurrences = computeUpcomingReminderOccurrences('weekly:friday', {
      count: 2,
      from: new Date('2026-04-20T08:00:00'),
      hour: 9,
      minute: 0,
    });

    expect(occurrences.map((date) => date.toISOString())).toEqual([
      new Date('2026-04-24T09:00:00').toISOString(),
      new Date('2026-05-01T09:00:00').toISOString(),
    ]);
  });

  it('computes fortnightly occurrences as concrete future dates', () => {
    const occurrences = computeUpcomingReminderOccurrences('fortnightly', {
      count: 2,
      from: new Date('2026-04-20T08:00:00'),
      hour: 9,
      minute: 0,
      startDate: '2026-04-06',
    });

    expect(occurrences.map((date) => date.toISOString())).toEqual([
      new Date('2026-04-20T09:00:00').toISOString(),
      new Date('2026-05-04T09:00:00').toISOString(),
    ]);
  });

  it('rejects unsupported recurrence strings', () => {
    expect(() => computeUpcomingReminderOccurrences('monthly')).toThrow('invalid-recurrence');
  });

  it('requires a start date for fortnightly recurrence', () => {
    expect(() => computeUpcomingReminderOccurrences('fortnightly')).toThrow('start-date-required');
  });
});
