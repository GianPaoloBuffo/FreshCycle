import { groupGarments, normalizeWardrobeGarment } from '@/features/wardrobe/groupGarments';
import { WardrobeGarment } from '@/features/wardrobe/types';

import { LoadPlanningMode, PlannedLoadIssue, PlannedLoadSummary, PlannedLoadType } from './types';

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

    return Array.from(loadMap.values())
      .map((load) => addIssues(load))
      .sort((left, right) => left.title.localeCompare(right.title));
  }

  return groupGarments(garments, mode).map((section) =>
    addIssues({
      key: section.key,
      title: section.title,
      description: section.description,
      mode,
      loadType: inferLoadType(section.key),
      garments: section.garments,
      hasUnknownTemperature: section.garments.some(
        (garment) => garment.care_method === 'machine_wash' && garment.machine_temp_bucket === 'unknown'
      ),
      issues: [],
      conflictCount: 0,
      warningCount: 0,
    })
  );
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
        issues: [],
        conflictCount: 0,
        warningCount: 0,
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
        issues: [],
        conflictCount: 0,
        warningCount: 0,
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
        issues: [],
        conflictCount: 0,
        warningCount: 0,
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

function addIssues(load: PlannedLoadSummary): PlannedLoadSummary {
  const issues = detectLoadIssues(load);

  return {
    ...load,
    issues,
    conflictCount: issues.filter((issue) => issue.severity === 'conflict').length,
    warningCount: issues.filter((issue) => issue.severity === 'warning').length,
  };
}

function detectLoadIssues(load: PlannedLoadSummary): PlannedLoadIssue[] {
  const issues: PlannedLoadIssue[] = [];
  const dryCleanGarments = load.garments.filter((garment) => garment.care_method === 'dry_clean');
  const handWashGarments = load.garments.filter((garment) => garment.care_method === 'hand_wash');
  const machineWashGarments = load.garments.filter((garment) => garment.care_method === 'machine_wash');
  const hasDryCleanConflict =
    dryCleanGarments.length > 0 &&
    (load.loadType === 'machine_wash' ||
      (load.loadType === 'mixed' && dryCleanGarments.length !== load.garments.length));
  const hasHandWashConflict =
    handWashGarments.length > 0 &&
    (load.loadType === 'machine_wash' ||
      (load.loadType === 'mixed' && handWashGarments.length !== load.garments.length));
  const machineWashBuckets = Array.from(
    new Set(machineWashGarments.map((garment) => garment.machine_temp_bucket))
  );

  if (hasDryCleanConflict) {
    issues.push({
      code: 'dry_clean_in_home_load',
      severity: 'conflict',
      title: 'Dry-clean pieces are mixed into this load',
      message: 'Remove dry-clean-only garments before treating this group as a home wash load.',
    });
  }

  if (hasHandWashConflict) {
    issues.push({
      code: 'hand_wash_in_machine_load',
      severity: 'conflict',
      title: 'Hand-wash garments are mixed into this load',
      message: 'Hand-wash-only pieces should be separated before running this load in the machine.',
    });
  }

  if (
    machineWashGarments.length > 1 &&
    machineWashBuckets.length > 1 &&
    (load.loadType === 'machine_wash' || load.loadType === 'mixed')
  ) {
    issues.push({
      code: 'mixed_machine_temperatures',
      severity: 'conflict',
      title: 'Machine-wash temperatures do not match',
      message: 'This group contains garments with different machine temperature ceilings, so it is not a safe single load yet.',
    });
  }

  if (load.mode === 'temperature' && load.key === 'temperature:unknown' && load.hasUnknownTemperature) {
    issues.push({
      code: 'unknown_temperature',
      severity: 'warning',
      title: 'Temperature is unknown for this group',
      message: 'Review garment labels before washing because FreshCycle could not infer a safe machine temperature.',
    });
  }

  return issues;
}
