import { describe, expect, it } from 'vitest';

import { groupGarments, normalizeWardrobeGarment } from './groupGarments';

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
    name: 'Navy Hoodie',
    category: 'Outerwear',
    primary_color: 'Navy',
    wash_temperature_c: 40,
    care_instructions: ['Machine washable'],
    label_image_path: null,
  },
  {
    id: '3',
    user_id: 'user-123',
    name: 'Silk Blouse',
    category: null,
    primary_color: 'Pink',
    wash_temperature_c: null,
    care_instructions: ['Dry clean only'],
    label_image_path: null,
  },
];

describe('normalizeWardrobeGarment', () => {
  it('normalizes category, care method, colour group, and temperature bucket', () => {
    const result = normalizeWardrobeGarment(garments[2]!);

    expect(result.category_key).toBe('uncategorized');
    expect(result.category_label).toBe('Uncategorized');
    expect(result.care_method).toBe('dry_clean');
    expect(result.colour_group).toBe('colour');
    expect(result.machine_temp_bucket).toBe('unknown');
  });
});

describe('groupGarments', () => {
  it('returns a flat section when flat mode is selected', () => {
    const sections = groupGarments(garments, 'flat');

    expect(sections).toHaveLength(1);
    expect(sections[0]?.title).toBe('All garments');
    expect(sections[0]?.garments).toHaveLength(3);
  });

  it('groups garments by normalized colour buckets', () => {
    const sections = groupGarments(garments, 'colour');

    expect(sections.map((section) => section.title)).toEqual(['Whites', 'Darks', 'Dry clean only']);
    expect(sections[1]?.garments[0]?.name).toBe('Navy Hoodie');
    expect(sections[2]?.garments[0]?.name).toBe('Silk Blouse');
  });

  it('groups garments by machine temperature buckets', () => {
    const sections = groupGarments(garments, 'temperature');

    expect(sections.map((section) => section.title)).toEqual(['30C loads', '40C loads', 'Dry clean only']);
    expect(sections[2]?.description).toContain('stay out of machine-wash load groupings');
  });

  it('uses normalized category labels when grouping by category', () => {
    const sections = groupGarments(garments, 'category');

    expect(sections.map((section) => section.title)).toEqual([
      'Outerwear',
      'Tops',
      'Uncategorized',
    ]);
    expect(sections[2]?.description).toContain('Browsing group only');
  });
});
