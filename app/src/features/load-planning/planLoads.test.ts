import { describe, expect, it } from 'vitest';

import { getPlannedLoad, planLoads } from './planLoads';

const garments = [
  {
    id: '1',
    user_id: 'user-123',
    name: 'White Tee',
    category: 'Tops',
    primary_color: 'White',
    wash_temperature_c: 30,
    care_instructions: ['Machine washable'],
    label_image_path: null,
  },
  {
    id: '2',
    user_id: 'user-123',
    name: 'White Pillowcase',
    category: 'Bedding',
    primary_color: 'White',
    wash_temperature_c: 30,
    care_instructions: ['Machine washable'],
    label_image_path: null,
  },
  {
    id: '3',
    user_id: 'user-123',
    name: 'Navy Hoodie',
    category: 'Outerwear',
    primary_color: 'Navy',
    wash_temperature_c: 40,
    care_instructions: ['Machine washable'],
    label_image_path: null,
  },
  {
    id: '4',
    user_id: 'user-123',
    name: 'Merino Sweater',
    category: 'Knitwear',
    primary_color: 'Gray',
    wash_temperature_c: null,
    care_instructions: ['Hand wash only'],
    label_image_path: null,
  },
  {
    id: '5',
    user_id: 'user-123',
    name: 'Silk Blouse',
    category: null,
    primary_color: 'Pink',
    wash_temperature_c: null,
    care_instructions: ['Dry clean only'],
    label_image_path: null,
  },
];

describe('planLoads', () => {
  it('creates canonical smart loads from care method, temp bucket, and colour group', () => {
    const loads = planLoads(garments, 'smart');

    expect(loads.map((load) => load.title)).toEqual([
      '30C whites',
      '40C darks',
      'Dry clean load',
      'Hand wash load',
    ]);
    expect(loads[0]?.garments).toHaveLength(2);
    expect(loads[0]?.loadType).toBe('machine_wash');
  });

  it('surfaces unknown-temperature machine wash loads distinctly', () => {
    const loads = planLoads(
      [
        {
          id: '6',
          user_id: 'user-123',
          name: 'Mystery Tee',
          category: 'Tops',
          primary_color: 'Blue',
          wash_temperature_c: null,
          care_instructions: ['Machine washable'],
          label_image_path: null,
        },
      ],
      'smart'
    );

    expect(loads[0]?.title).toBe('Unknown-temp colours');
    expect(loads[0]?.hasUnknownTemperature).toBe(true);
  });

  it('flags mixed care conflicts in category loads', () => {
    const loads = planLoads(
      [
        {
          id: '6',
          user_id: 'user-123',
          name: 'Cotton Tee',
          category: 'Tops',
          primary_color: 'White',
          wash_temperature_c: 30,
          care_instructions: ['Machine washable'],
          label_image_path: null,
        },
        {
          id: '7',
          user_id: 'user-123',
          name: 'Wool Shell',
          category: 'Tops',
          primary_color: 'Cream',
          wash_temperature_c: null,
          care_instructions: ['Hand wash only'],
          label_image_path: null,
        },
      ],
      'category'
    );
    const mixedTopsLoad = loads.find((load) => load.title === 'Tops');

    expect(mixedTopsLoad?.conflictCount).toBe(1);
    expect(mixedTopsLoad?.issues[0]?.code).toBe('hand_wash_in_machine_load');
  });

  it('does not flag a single special-care category load as a conflict by itself', () => {
    const loads = planLoads(
      [
        {
          id: '6',
          user_id: 'user-123',
          name: 'Wool Shell',
          category: 'Tops',
          primary_color: 'Cream',
          wash_temperature_c: null,
          care_instructions: ['Hand wash only'],
          label_image_path: null,
        },
      ],
      'category'
    );

    expect(loads[0]?.issues).toEqual([]);
  });

  it('flags mixed machine temperature buckets inside non-smart loads', () => {
    const loads = planLoads(
      [
        {
          id: '6',
          user_id: 'user-123',
          name: 'Blue Tee',
          category: 'Tops',
          primary_color: 'Blue',
          wash_temperature_c: 30,
          care_instructions: ['Machine washable'],
          label_image_path: null,
        },
        {
          id: '7',
          user_id: 'user-123',
          name: 'Red Tee',
          category: 'Tops',
          primary_color: 'Red',
          wash_temperature_c: 40,
          care_instructions: ['Machine washable'],
          label_image_path: null,
        },
      ],
      'category'
    );

    expect(loads[0]?.issues.some((issue) => issue.code === 'mixed_machine_temperatures')).toBe(true);
  });

  it('warns when a temperature-planned load still has unknown temperature labels', () => {
    const loads = planLoads(
      [
        {
          id: '6',
          user_id: 'user-123',
          name: 'Mystery Tee',
          category: 'Tops',
          primary_color: 'Blue',
          wash_temperature_c: null,
          care_instructions: ['Machine washable'],
          label_image_path: null,
        },
      ],
      'temperature'
    );

    expect(loads[0]?.warningCount).toBe(1);
    expect(loads[0]?.issues[0]?.code).toBe('unknown_temperature');
  });

  it('supports category planning mode for later conflict detection work', () => {
    const loads = planLoads(garments, 'category');

    expect(loads.some((load) => load.loadType === 'mixed')).toBe(true);
    expect(loads.find((load) => load.title === 'Uncategorized')?.garments[0]?.name).toBe('Silk Blouse');
  });

  it('returns no loads for an empty wardrobe', () => {
    expect(planLoads([], 'smart')).toEqual([]);
  });

  it('can retrieve a specific planned load by key for detail screens', () => {
    const load = getPlannedLoad(garments, 'smart', 'smart:machine_wash:30:white');

    expect(load?.title).toBe('30C whites');
    expect(load?.garments).toHaveLength(2);
  });
});
