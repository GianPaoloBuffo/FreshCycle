import { describe, expect, it } from 'vitest';

import {
  buildCreateSchedulePayload,
  createInitialScheduleFormValues,
  validateScheduleDraft,
} from './form';

describe('schedule form helpers', () => {
  it('creates stable initial defaults', () => {
    expect(createInitialScheduleFormValues()).toEqual({
      name: '',
      recurrence: 'daily',
      startsOn: expect.stringMatching(/^\d{4}-\d{2}-\d{2}$/),
      garmentIds: [],
      remindersEnabled: true,
    });
  });

  it('requires a non-empty trimmed name', () => {
    expect(
      validateScheduleDraft({
        name: '   ',
        recurrence: 'daily',
        startsOn: '2026-04-20',
        garmentIds: ['garment-1'],
        remindersEnabled: true,
      }).name
    ).toBeTruthy();
  });

  it('requires at least one selected garment', () => {
    expect(
      validateScheduleDraft({
        name: 'Weekly towels',
        recurrence: 'daily',
        startsOn: '2026-04-20',
        garmentIds: [],
        remindersEnabled: true,
      }).garmentIds
    ).toBeTruthy();
  });

  it('requires a supported recurrence value', () => {
    expect(
      validateScheduleDraft({
        name: 'Weekly towels',
        recurrence: 'monthly',
        startsOn: '2026-04-20',
        garmentIds: ['garment-1'],
        remindersEnabled: true,
      }).recurrence
    ).toBeTruthy();
  });

  it('requires a valid local start date', () => {
    expect(
      validateScheduleDraft({
        name: 'Weekly towels',
        recurrence: 'daily',
        startsOn: '2026-99-99',
        garmentIds: ['garment-1'],
        remindersEnabled: true,
      }).startsOn
    ).toBeTruthy();
  });

  it('builds a trimmed payload with unique garment ids', () => {
    expect(
      buildCreateSchedulePayload({
        name: '  Weekly towels  ',
        recurrence: 'fortnightly',
        startsOn: '2026-04-20',
        garmentIds: ['garment-1', 'garment-1', 'garment-2'],
        remindersEnabled: false,
      })
    ).toEqual({
      name: 'Weekly towels',
      recurrence: 'fortnightly',
      starts_on: '2026-04-20',
      garment_ids: ['garment-1', 'garment-2'],
      reminders_enabled: false,
    });
  });
});
