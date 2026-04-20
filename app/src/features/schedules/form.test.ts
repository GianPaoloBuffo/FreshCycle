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
      garmentIds: [],
      remindersEnabled: true,
    });
  });

  it('requires a non-empty trimmed name', () => {
    expect(
      validateScheduleDraft({
        name: '   ',
        recurrence: 'daily',
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
        garmentIds: ['garment-1'],
        remindersEnabled: true,
      }).recurrence
    ).toBeTruthy();
  });

  it('builds a trimmed payload with unique garment ids', () => {
    expect(
      buildCreateSchedulePayload({
        name: '  Weekly towels  ',
        recurrence: 'fortnightly',
        garmentIds: ['garment-1', 'garment-1', 'garment-2'],
        remindersEnabled: false,
      })
    ).toEqual({
      name: 'Weekly towels',
      recurrence: 'fortnightly',
      garment_ids: ['garment-1', 'garment-2'],
      reminders_enabled: false,
    });
  });
});
