import { groupGarments, normalizeWardrobeGarment } from '@/features/wardrobe/groupGarments';
import { WardrobeGarment } from '@/features/wardrobe/types';

import { LoadPlanningMode, PlannedLoadSummary, PlannedLoadType } from './types';

export function planLoads(garments: WardrobeGarment[], mode: LoadPlanningMode): PlannedLoadSummary[] {
  const normalizedGarments = garments.map(normalizeWardrobeGarment);

  if (normalizedGarments.length === 0) {
    return [];
  }

  if (mode === 'smart') {
    const loadMap = new Map<string, PlannedLoadSummary>();

    for (const garment of normalizedGarments) {
      const key = buildSmartLoadKey(garment);
      const existing = loadMap.get(key);

      if (existing) {
        existing.garments.push(garment);
        existing.hasUnknownTemperature =
          existing.hasUnknownTemperature || garment.machine_temp_bucket === 'unknown';
        continue;
      }

      loadMap.set(key, createSmartLoad(garment));
    }

    return Array.from(loadMap.values()).sort((left, right) => left.title.localeCompare(right.title));
  }

  return groupGarments(garments, mode).map((section) => {
    const loadType = inferLoadType(section.key);
    const hasUnknownTemperature = section.garments.some(
      (garment) => garment.care_method === 'machine_wash' && garment.machine_temp_bucket === 'unknown'
    );

    return {
      key: section.key,
      title: section.title,
      description: section.description,
      mode,
      loadType,
      garments: section.garments,
      hasUnknownTemperature,
    };
  });
}

export function getPlannedLoad(
  garments: WardrobeGarment[],
  mode: LoadPlanningMode,
  loadKey: string
) {
  return planLoads(garments, mode).find((load) => load.key === loadKey) ?? null;
}

function buildSmartLoadKey(garment: ReturnType<typeof normalizeWardrobeGarment>) {
  switch (garment.care_method) {
    case 'hand_wash':
      return 'smart:hand_wash';
    case 'dry_clean':
      return 'smart:dry_clean';
    default:
      return `smart:machine_wash:${garment.machine_temp_bucket}:${garment.colour_group}`;
  }
}

function createSmartLoad(garment: ReturnType<typeof normalizeWardrobeGarment>): PlannedLoadSummary {
  switch (garment.care_method) {
    case 'hand_wash':
      return {
        key: 'smart:hand_wash',
        title: 'Hand wash load',
        description: 'Wash separately by hand or on a hand-wash cycle only.',
        mode: 'smart',
        loadType: 'hand_wash',
        garments: [garment],
        hasUnknownTemperature: false,
      };
    case 'dry_clean':
      return {
        key: 'smart:dry_clean',
        title: 'Dry clean load',
        description: 'Set aside for dry cleaning instead of home washing.',
        mode: 'smart',
        loadType: 'dry_clean',
        garments: [garment],
        hasUnknownTemperature: false,
      };
    default:
      return {
        key: buildSmartLoadKey(garment),
        title: smartMachineLoadTitle(garment.machine_temp_bucket, garment.colour_group),
        description: smartMachineLoadDescription(garment.machine_temp_bucket, garment.colour_group),
        mode: 'smart',
        loadType: 'machine_wash',
        garments: [garment],
        hasUnknownTemperature: garment.machine_temp_bucket === 'unknown',
      };
  }
}

function smartMachineLoadTitle(
  bucket: ReturnType<typeof normalizeWardrobeGarment>['machine_temp_bucket'],
  colourGroup: ReturnType<typeof normalizeWardrobeGarment>['colour_group']
) {
  const colourLabel =
    colourGroup === 'white'
      ? 'whites'
      : colourGroup === 'light'
        ? 'lights'
        : colourGroup === 'dark'
          ? 'darks'
          : 'colours';

  if (bucket === 'unknown') {
    return `Unknown-temp ${colourLabel}`;
  }

  return `${bucket}C ${colourLabel}`;
}

function smartMachineLoadDescription(
  bucket: ReturnType<typeof normalizeWardrobeGarment>['machine_temp_bucket'],
  colourGroup: ReturnType<typeof normalizeWardrobeGarment>['colour_group']
) {
  const colourHint =
    colourGroup === 'white'
      ? 'white and pale neutral pieces.'
      : colourGroup === 'light'
        ? 'lighter-toned machine-wash pieces.'
        : colourGroup === 'dark'
          ? 'darker shades that should stay together.'
          : 'vibrant or uncategorized colours.';

  if (bucket === 'unknown') {
    return `Machine-wash garments with no reliable temperature ceiling. Review labels before washing ${colourHint}`;
  }

  return `Machine-wash garments capped at ${bucket}C for ${colourHint}`;
}

function inferLoadType(sectionKey: string): PlannedLoadType {
  if (sectionKey.includes('dry_clean')) {
    return 'dry_clean';
  }

  if (sectionKey.includes('hand_wash')) {
    return 'hand_wash';
  }

  if (sectionKey.startsWith('category:')) {
    return 'mixed';
  }

  return 'machine_wash';
}
