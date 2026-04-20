import { NormalizedWardrobeGarment } from '@/features/wardrobe/types';

export type LoadPlanningMode = 'smart' | 'colour' | 'temperature' | 'category';

export type PlannedLoadType = 'machine_wash' | 'hand_wash' | 'dry_clean' | 'mixed';

export type PlannedLoadIssueSeverity = 'warning' | 'conflict';

export type PlannedLoadIssueCode =
  | 'dry_clean_in_home_load'
  | 'hand_wash_in_machine_load'
  | 'mixed_machine_temperatures'
  | 'unknown_temperature';

export type PlannedLoadIssue = {
  code: PlannedLoadIssueCode;
  severity: PlannedLoadIssueSeverity;
  title: string;
  message: string;
};

export type PlannedLoadSummary = {
  key: string;
  title: string;
  description: string;
  mode: LoadPlanningMode;
  loadType: PlannedLoadType;
  garments: NormalizedWardrobeGarment[];
  hasUnknownTemperature: boolean;
  issues: PlannedLoadIssue[];
  conflictCount: number;
  warningCount: number;
};
