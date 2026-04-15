import {
  NormalizedWardrobeGarment,
  WardrobeColourGroup,
  WardrobeGarment,
  WardrobeGroupSection,
  WardrobeGroupingMode,
  WardrobeMachineTempBucket,
} from './types';

const WHITE_COLOURS = ['white', 'ivory', 'cream', 'off white', 'off-white'];
const LIGHT_COLOURS = [
  'beige',
  'tan',
  'khaki',
  'stone',
  'light blue',
  'light pink',
  'light gray',
  'light grey',
  'silver',
  'pastel',
];
const DARK_COLOURS = [
  'black',
  'navy',
  'charcoal',
  'dark blue',
  'dark green',
  'dark gray',
  'dark grey',
  'brown',
  'maroon',
  'burgundy',
];

export function groupGarments(
  garments: WardrobeGarment[],
  mode: WardrobeGroupingMode
): WardrobeGroupSection[] {
  const normalizedGarments = garments.map(normalizeWardrobeGarment).sort(sortGarmentsByName);

  if (mode === 'flat') {
    return [
      {
        key: 'all',
        title: 'All garments',
        description: `${normalizedGarments.length} saved item${normalizedGarments.length === 1 ? '' : 's'}`,
        garments: normalizedGarments,
      },
    ];
  }

  const sectionMap = new Map<string, WardrobeGroupSection>();

  for (const garment of normalizedGarments) {
    const section = getSectionDefinition(garment, mode);
    const existing = sectionMap.get(section.key);

    if (existing) {
      existing.garments.push(garment);
      continue;
    }

    sectionMap.set(section.key, {
      ...section,
      garments: [garment],
    });
  }

  return Array.from(sectionMap.values()).sort((left, right) => compareSections(left.key, right.key, mode));
}

export function normalizeWardrobeGarment(garment: WardrobeGarment): NormalizedWardrobeGarment {
  const normalizedCategory = normalizeCategory(garment.category);
  const careInstructions = garment.care_instructions.map((instruction) => instruction.toLowerCase());

  return {
    ...garment,
    category_key: normalizedCategory.key,
    category_label: normalizedCategory.label,
    care_method: inferCareMethod(careInstructions),
    colour_group: inferColourGroup(garment.primary_color),
    machine_temp_bucket: inferMachineTempBucket(garment.wash_temperature_c),
  };
}

function normalizeCategory(category: string | null) {
  const trimmed = category?.trim() ?? '';
  if (!trimmed) {
    return {
      key: 'uncategorized',
      label: 'Uncategorized',
    };
  }

  return {
    key: trimmed.toLowerCase(),
    label: trimmed,
  };
}

function inferCareMethod(careInstructions: string[]) {
  const combined = careInstructions.join(' ');

  if (combined.includes('dry clean')) {
    return 'dry_clean' as const;
  }

  if (combined.includes('hand wash')) {
    return 'hand_wash' as const;
  }

  return 'machine_wash' as const;
}

function inferColourGroup(primaryColor: string | null): WardrobeColourGroup {
  const normalized = primaryColor?.trim().toLowerCase() ?? '';

  if (!normalized) {
    return 'colour';
  }

  if (matchesColourGroup(normalized, WHITE_COLOURS)) {
    return 'white';
  }

  if (matchesColourGroup(normalized, LIGHT_COLOURS)) {
    return 'light';
  }

  if (matchesColourGroup(normalized, DARK_COLOURS)) {
    return 'dark';
  }

  return 'colour';
}

function matchesColourGroup(value: string, swatches: string[]) {
  return swatches.some((swatch) => value === swatch || value.includes(swatch));
}

function inferMachineTempBucket(washTemperatureC: number | null): WardrobeMachineTempBucket {
  if (washTemperatureC === null) {
    return 'unknown';
  }

  if (washTemperatureC <= 30) {
    return '30';
  }

  if (washTemperatureC <= 40) {
    return '40';
  }

  if (washTemperatureC <= 60) {
    return '60';
  }

  return '90';
}

function getSectionDefinition(garment: NormalizedWardrobeGarment, mode: WardrobeGroupingMode) {
  switch (mode) {
    case 'colour':
      if (garment.care_method !== 'machine_wash') {
        return {
          key: `special-care:${garment.care_method}`,
          title: specialCareTitle(garment.care_method),
          description: specialCareDescription(garment.care_method),
        };
      }

      return {
        key: `colour:${garment.colour_group}`,
        title: colourGroupTitle(garment.colour_group),
        description: colourGroupDescription(garment.colour_group),
      };
    case 'temperature':
      if (garment.care_method !== 'machine_wash') {
        return {
          key: `special-care:${garment.care_method}`,
          title: specialCareTitle(garment.care_method),
          description: specialCareDescription(garment.care_method),
        };
      }

      return {
        key: `temperature:${garment.machine_temp_bucket}`,
        title: machineTempBucketTitle(garment.machine_temp_bucket),
        description: machineTempBucketDescription(garment.machine_temp_bucket),
      };
    case 'category':
      return {
        key: `category:${garment.category_key}`,
        title: garment.category_label,
        description: 'Browsing group only; not a guaranteed safe load.',
      };
    default:
      return {
        key: 'all',
        title: 'All garments',
        description: 'Flat wardrobe list',
      };
  }
}

function compareSections(left: string, right: string, mode: WardrobeGroupingMode) {
  if (mode === 'colour') {
    return rankByOrder(left, right, [
      'colour:white',
      'colour:light',
      'colour:dark',
      'colour:colour',
      'special-care:hand_wash',
      'special-care:dry_clean',
    ]);
  }

  if (mode === 'temperature') {
    return rankByOrder(left, right, [
      'temperature:30',
      'temperature:40',
      'temperature:60',
      'temperature:90',
      'temperature:unknown',
      'special-care:hand_wash',
      'special-care:dry_clean',
    ]);
  }

  if (mode === 'category') {
    return left.localeCompare(right);
  }

  return 0;
}

function rankByOrder(left: string, right: string, order: string[]) {
  const leftIndex = order.indexOf(left);
  const rightIndex = order.indexOf(right);

  if (leftIndex === -1 || rightIndex === -1) {
    return left.localeCompare(right);
  }

  return leftIndex - rightIndex;
}

function sortGarmentsByName(left: NormalizedWardrobeGarment, right: NormalizedWardrobeGarment) {
  return left.name.localeCompare(right.name);
}

function colourGroupTitle(group: WardrobeColourGroup) {
  switch (group) {
    case 'white':
      return 'Whites';
    case 'light':
      return 'Lights';
    case 'dark':
      return 'Darks';
    default:
      return 'Colours';
  }
}

function colourGroupDescription(group: WardrobeColourGroup) {
  switch (group) {
    case 'white':
      return 'Bright whites and pale neutrals.';
    case 'light':
      return 'Soft tones and lighter shades.';
    case 'dark':
      return 'Deep shades that benefit from darker loads.';
    default:
      return 'Everything vivid, saturated, or uncategorized.';
  }
}

function machineTempBucketTitle(bucket: WardrobeMachineTempBucket) {
  switch (bucket) {
    case '30':
      return '30C loads';
    case '40':
      return '40C loads';
    case '60':
      return '60C loads';
    case '90':
      return '90C loads';
    default:
      return 'Unknown temperature';
  }
}

function machineTempBucketDescription(bucket: WardrobeMachineTempBucket) {
  switch (bucket) {
    case '30':
      return 'Cold and delicate machine-wash items.';
    case '40':
      return 'Standard everyday machine wash items.';
    case '60':
      return 'Hotter washes for durable fabrics.';
    case '90':
      return 'High-heat items only.';
    default:
      return 'Missing wash temperature, so review the label before grouping into a load.';
  }
}

function specialCareTitle(careMethod: NormalizedWardrobeGarment['care_method']) {
  switch (careMethod) {
    case 'hand_wash':
      return 'Hand wash only';
    case 'dry_clean':
      return 'Dry clean only';
    default:
      return 'Special care';
  }
}

function specialCareDescription(careMethod: NormalizedWardrobeGarment['care_method']) {
  switch (careMethod) {
    case 'hand_wash':
      return 'Keep these items out of machine-wash load groupings.';
    case 'dry_clean':
      return 'These garments should stay out of machine-wash load groupings.';
    default:
      return 'Review these items separately before building a load.';
  }
}
