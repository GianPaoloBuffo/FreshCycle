import { NormalizedWardrobeGarment } from '@/features/wardrobe/types';

export type LoadPlanningMode = 'smart' | 'colour' | 'temperature' | 'category';

export type PlannedLoadType = 'machine_wash' | 'hand_wash' | 'dry_clean' | 'mixed';

export type PlannedLoadSummary = {
  key: string;
  title: string;
  description: string;
  mode: LoadPlanningMode;
  loadType: PlannedLoadType;
  garments: NormalizedWardrobeGarment[];
  hasUnknownTemperature: boolean;
};
