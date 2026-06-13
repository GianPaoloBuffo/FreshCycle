export type LabelPhotoSource = 'camera' | 'library';

export type SelectedLabelPhoto = {
  uri: string;
  fileName: string | null;
  mimeType: string | null;
  width: number;
  height: number;
  fileSize: number | null;
  webFile?: Blob | null;
  source: LabelPhotoSource;
};

export type ParsedLabelPreview = {
  garmentName: string;
  suggestedCategory: string;
  careSummary: string;
  confidenceLabel: 'Review needed' | 'Mostly confident';
  notes: string[];
  confidenceScore?: number;
  explanation?: string;
  needsUserConfirmation?: boolean;
  uncertainFields?: string[];
};

export type ParsedGarmentFields = {
  nameSuggestion: string;
  category: string;
  primaryColor: string;
  washTemperatureC: number | null;
  careInstructions: string[];
  machineWashable: boolean;
  tumbleDry: boolean;
  dryCleanOnly: boolean;
  ironAllowed: boolean;
  ironTemp: 'low' | 'medium' | 'high' | null;
  bleachAllowed: boolean;
  fabricNotes: string[];
  rawLabelText: string;
  confidenceScore?: number;
  needsUserConfirmation?: boolean;
  uncertainFields?: string[];
  fieldConfidence?: Partial<Record<CareLabelReviewField, number>>;
};

export type CareLabelReviewField =
  | 'wash'
  | 'bleach'
  | 'drying'
  | 'ironing'
  | 'professional_cleaning'
  | 'fabric'
  | 'raw_text'
  | 'name';

export type ParsedLabelResult = {
  parsed: ParsedGarmentFields;
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
  | 'api-unavailable'
  | 'invalid-garment-id'
  | 'invalid-label-image-path'
  | 'invalid-wash-temperature'
  | 'name-required'
  | 'save-failed'
  | 'upload-failed'
  | 'processing-failed';
