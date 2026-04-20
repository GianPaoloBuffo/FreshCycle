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
