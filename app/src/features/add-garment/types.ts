export type LabelPhotoSource = 'camera' | 'library';

export type SelectedLabelPhoto = {
  uri: string;
  fileName: string | null;
  mimeType: string | null;
  width: number;
  height: number;
  fileSize: number | null;
  source: LabelPhotoSource;
};

export type ParsedLabelPreview = {
  garmentName: string;
  suggestedCategory: string;
  careSummary: string;
  confidenceLabel: 'Review needed' | 'Mostly confident';
  notes: string[];
};

export type ParsedLabelResult = {
  preview: ParsedLabelPreview;
  durationMs: number;
  completedAt: string;
};

export type AddGarmentErrorCode =
  | 'camera-permission-denied'
  | 'photo-library-permission-denied'
  | 'camera-unavailable'
  | 'selection-empty'
  | 'auth-required'
  | 'processing-failed';
