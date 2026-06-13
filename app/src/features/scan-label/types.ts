import { CareLabelReviewField, ParsedLabelResult, SelectedLabelPhoto } from '@/features/add-garment/types';

export type OCRFrame = {
  x: number;
  y: number;
  width: number;
  height: number;
};

export type OCRTextElement = {
  text: string;
  frame: OCRFrame | null;
  confidence: number | null;
};

export type OCRTextLine = {
  text: string;
  frame: OCRFrame | null;
  confidence: number | null;
  elements: OCRTextElement[];
};

export type OCRTextBlock = {
  text: string;
  frame: OCRFrame | null;
  confidence: number | null;
  lines: OCRTextLine[];
};

export type ClientOCRResult = {
  source: 'mlkit-text-recognition-v2';
  text: string;
  blocks: OCRTextBlock[];
  averageConfidence: number | null;
};

export type ClientSymbolDetection = {
  symbol: string;
  class?: string;
  label?: string;
  confidence: number | null;
  frame?: OCRFrame | null;
};

export type ScanLabelSymbolDetection = {
  className: string;
  label: string;
  confidence: number;
  boundingBox: OCRFrame | null;
  source: string | null;
};

export type ScanField<TValue extends string> = {
  value: TValue | 'unknown';
  confidence: number;
  explanation: string;
  needsConfirmation: boolean;
};

export type WashValue = 'machine_wash' | 'hand_wash' | 'do_not_wash';
export type BleachValue = 'allowed' | 'non_chlorine_only' | 'do_not_bleach';
export type DryingValue =
  | 'tumble_dry'
  | 'tumble_dry_low'
  | 'line_dry'
  | 'dry_flat'
  | 'do_not_tumble_dry';
export type IroningValue = 'allowed' | 'low' | 'medium' | 'high' | 'do_not_iron';
export type ProfessionalCleaningValue = 'dry_clean' | 'professional_clean_only' | 'do_not_dry_clean';

export type ScanLabelCareInstructions = {
  wash: ScanField<WashValue> & {
    temperatureC: number | null;
  };
  bleach: ScanField<BleachValue>;
  drying: ScanField<DryingValue>;
  ironing: ScanField<IroningValue>;
  professionalCleaning: ScanField<ProfessionalCleaningValue>;
};

export type ScanLabelResponse = {
  nameSuggestion: string;
  fabricNotes: string[];
  rawText: string;
  care: ScanLabelCareInstructions;
  confidence: number;
  explanation: string;
  uncertainFields: CareLabelReviewField[];
  needsUserConfirmation: boolean;
  provider: string | null;
  source: string | null;
  route: string | null;
  cacheHit: boolean;
  imageHash: string | null;
  paidFallbackUsed: boolean;
  fallbackCallsAvoided: number;
  routingReasons: string[];
  symbolDetections: ScanLabelSymbolDetection[];
};

export type ScanLabelClientResult = {
  photo: SelectedLabelPhoto;
  parsedLabel: ParsedLabelResult;
  scan: ScanLabelResponse;
  ocr: ClientOCRResult | null;
  quality: CaptureQualityResult;
};

export type CaptureQualityIssue =
  | 'possible_blur'
  | 'poor_lighting'
  | 'small_label_area'
  | 'unstable_framing'
  | 'low_ocr_signal';

export type CaptureQualityHint = {
  issue: CaptureQualityIssue;
  message: string;
  blocksCapture: boolean;
};

export type CaptureQualityInput = {
  photo: Pick<SelectedLabelPhoto, 'width' | 'height' | 'fileSize'>;
  cropCoverage: number;
  stableForMs: number;
  ocrText?: string | null;
  ocrLineCount?: number;
};

export type CaptureQualityResult = {
  score: number;
  hints: CaptureQualityHint[];
  canSubmit: boolean;
};

export type CropRegion = {
  originXRatio: number;
  originYRatio: number;
  widthRatio: number;
  heightRatio: number;
};
