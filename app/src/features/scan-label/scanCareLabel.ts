import { AddGarmentActionError } from '@/features/add-garment/parseCareLabel';
import { ParsedGarmentFields, ParsedLabelResult, SelectedLabelPhoto } from '@/features/add-garment/types';
import { getAppEnv } from '@/lib/env';
import {
  BleachValue,
  CaptureQualityResult,
  ClientOCRResult,
  ClientSymbolDetection,
  DryingValue,
  IroningValue,
  ProfessionalCleaningValue,
  ScanField,
  ScanLabelCareInstructions,
  ScanLabelClientResult,
  ScanLabelResponse,
  WashValue,
} from '@/features/scan-label/types';

type FetchLike = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

type ScanCareLabelDeps = {
  apiBaseUrl?: string | null;
  accessToken?: string | null;
  fetchImpl?: FetchLike;
  platform?: string;
  now?: () => number;
};

type ScanCareLabelInput = {
  photo: SelectedLabelPhoto;
  clientOCR: ClientOCRResult | null;
  clientSymbols?: ClientSymbolDetection[];
  quality: CaptureQualityResult;
};

const unknownCare: ScanLabelCareInstructions = {
  wash: {
    value: 'unknown',
    temperatureC: null,
    confidence: 0,
    explanation: 'No wash instruction was confidently detected.',
    needsConfirmation: true,
  },
  bleach: {
    value: 'unknown',
    confidence: 0,
    explanation: 'No bleach instruction was confidently detected.',
    needsConfirmation: true,
  },
  drying: {
    value: 'unknown',
    confidence: 0,
    explanation: 'No drying instruction was confidently detected.',
    needsConfirmation: true,
  },
  ironing: {
    value: 'unknown',
    confidence: 0,
    explanation: 'No ironing instruction was confidently detected.',
    needsConfirmation: true,
  },
  professionalCleaning: {
    value: 'unknown',
    confidence: 0,
    explanation: 'No professional cleaning instruction was confidently detected.',
    needsConfirmation: true,
  },
};

export async function scanCareLabelPhoto(
  input: ScanCareLabelInput,
  deps: ScanCareLabelDeps = {}
): Promise<ScanLabelClientResult> {
  const apiBaseUrl = deps.apiBaseUrl ?? getAppEnv().apiBaseUrl;
  const fetchImpl = deps.fetchImpl ?? ((request, init) => fetch(request, init));
  const platform = deps.platform ?? inferRuntimePlatform();
  const accessToken = deps.accessToken ?? null;
  const now = deps.now ?? Date.now;

  if (!apiBaseUrl) {
    throw new AddGarmentActionError('api-unavailable');
  }

  if (!accessToken) {
    throw new AddGarmentActionError('auth-required');
  }

  const startedAt = now();
  const body = await buildScanMultipartBody(input, { fetchImpl, platform });
  const response = await fetchImpl(`${apiBaseUrl.replace(/\/$/, '')}/scan-label`, {
    method: 'POST',
    body,
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  });

  if (response.status === 401) {
    throw new AddGarmentActionError('auth-required');
  }

  if (!response.ok) {
    throw new AddGarmentActionError('processing-failed');
  }

  const rawResponse = await response.json();
  const scan = normalizeScanLabelResponse(rawResponse, input.clientOCR);
  const completedAt = new Date(now()).toISOString();

  return {
    photo: input.photo,
    scan,
    ocr: input.clientOCR,
    quality: input.quality,
    parsedLabel: buildParsedLabelResult(scan, {
      durationMs: now() - startedAt,
      completedAt,
    }),
  };
}

export function normalizeScanLabelResponse(rawResponse: unknown, clientOCR: ClientOCRResult | null = null): ScanLabelResponse {
  const raw = asRecord(rawResponse) ?? {};
  const root = asRecord(raw.result) ?? raw;
  const care = asRecord(root.care) ?? asRecord(root.care_instructions) ?? root;

  const normalizedCare: ScanLabelCareInstructions = {
    wash: normalizeWashField(care.wash, confidenceLike(root.confidence), isFieldUncertain(root, 'wash')),
    bleach: normalizeBackendField<BleachValue>(
      care.bleach,
      unknownCare.bleach,
      {
        allowed: 'allowed',
        non_chlorine_only: 'non_chlorine_only',
        do_not_bleach: 'do_not_bleach',
        unknown: 'unknown',
      },
      confidenceLike(root.confidence),
      isFieldUncertain(root, 'bleach')
    ),
    drying: normalizeDryingField(care.drying ?? care.dry, confidenceLike(root.confidence), isFieldUncertain(root, 'drying')),
    ironing: normalizeIroningField(care.ironing ?? care.iron, confidenceLike(root.confidence), isFieldUncertain(root, 'ironing')),
    professionalCleaning: normalizeBackendField<ProfessionalCleaningValue>(
      care.professionalCleaning ?? care.professional_cleaning ?? care.cleaning,
      unknownCare.professionalCleaning,
      {
        dry_clean_allowed: 'dry_clean',
        dry_clean_only: 'professional_clean_only',
        do_not_dry_clean: 'do_not_dry_clean',
        unknown: 'unknown',
      },
      confidenceLike(root.confidence),
      isFieldUncertain(root, 'professional_cleaning')
    ),
  };

  const explicitUncertain = normalizeUncertainFields(root.uncertainFields ?? root.uncertain_fields);
  const inferredUncertain = inferUncertainFields(normalizedCare);
  const uncertainFields = Array.from(new Set([...explicitUncertain, ...inferredUncertain]));
  const confidence = clampConfidence(root.confidence ?? averageCareConfidence(normalizedCare));
  const rawText = stringValue(root.rawText ?? root.raw_text ?? root.raw_label_text ?? clientOCR?.text ?? '');

  return {
    nameSuggestion: stringValue(root.nameSuggestion ?? root.name_suggestion ?? 'Care Label'),
    fabricNotes: stringArray(root.fabricNotes ?? root.fabric_notes),
    rawText,
    care: normalizedCare,
    confidence,
    explanation: stringValue(root.explanation ?? buildExplanation(normalizedCare, clientOCR)),
    uncertainFields,
    needsUserConfirmation: Boolean(
      root.needsUserConfirmation ??
        root.needs_user_confirmation ??
        (confidence < 0.76 || uncertainFields.length > 0)
    ),
    provider: nullableString(root.provider ?? root.parser ?? root.source),
  };
}

function normalizeWashField(rawValue: unknown, rootConfidence: number, forcedUncertain: boolean): ScanLabelCareInstructions['wash'] {
  const raw = asRecord(rawValue);
  const field = normalizeBackendField<WashValue>(
    rawValue,
    unknownCare.wash,
    {
      machine_wash: 'machine_wash',
      hand_wash: 'hand_wash',
      do_not_wash: 'do_not_wash',
      dry_clean_only: 'do_not_wash',
      unknown: 'unknown',
    },
    rootConfidence,
    forcedUncertain
  );

  return {
    ...field,
    temperatureC: asNullableNumber(getFirst(raw, ['temperatureC', 'temperature_c', 'max_temperature_c', 'wash_temp_max'])),
  };
}

function normalizeDryingField(rawValue: unknown, rootConfidence: number, forcedUncertain: boolean): ScanField<DryingValue> {
  const raw = asRecord(rawValue);
  const status = stringValue(raw?.status);
  const temperature = stringValue(raw?.temperature);
  const rawValueName = stringValue(raw?.value || raw?.instruction);
  const value = rawValueName
    ? rawValueName
    : status === 'tumble_dry' && temperature === 'low'
      ? 'tumble_dry_low'
      : mapStatus<DryingValue>(status, {
          tumble_dry: 'tumble_dry',
          do_not_tumble_dry: 'do_not_tumble_dry',
          line_dry: 'line_dry',
          dry_flat: 'dry_flat',
          unknown: 'unknown',
        });

  return normalizeBackendField<DryingValue>(
    {
      ...raw,
      value,
    },
    unknownCare.drying,
    {},
    rootConfidence,
    forcedUncertain
  );
}

function normalizeIroningField(rawValue: unknown, rootConfidence: number, forcedUncertain: boolean): ScanField<IroningValue> {
  const raw = asRecord(rawValue);
  const status = stringValue(raw?.status);
  const temperature = stringValue(raw?.temperature);
  const rawValueName = stringValue(raw?.value || raw?.instruction);
  const value = rawValueName
    ? rawValueName
    : status === 'iron_allowed' && ['low', 'medium', 'high'].includes(temperature)
      ? (temperature as IroningValue)
      : mapStatus<IroningValue>(status, {
          iron_allowed: 'allowed',
          do_not_iron: 'do_not_iron',
          unknown: 'unknown',
        });

  return normalizeBackendField<IroningValue>(
    {
      ...raw,
      value,
    },
    unknownCare.ironing,
    {},
    rootConfidence,
    forcedUncertain
  );
}

function normalizeBackendField<TValue extends string>(
  rawValue: unknown,
  fallback: ScanField<TValue>,
  statusMap: Record<string, TValue | 'unknown'>,
  rootConfidence: number,
  forcedUncertain: boolean
): ScanField<TValue> {
  const raw = asRecord(rawValue);
  if (!raw) {
    return fallback;
  }

  const rawStatus = stringValue(raw.status);
  const rawValueName = stringValue(raw.value || raw.instruction);
  const value = (rawValueName || mapStatus(rawStatus, statusMap) || fallback.value) as TValue | 'unknown';
  const confidence = clampConfidence(raw.confidence ?? (forcedUncertain || value === 'unknown' ? Math.min(rootConfidence, 0.62) : rootConfidence));

  return {
    value,
    confidence,
    explanation: stringValue(raw.explanation ?? raw.reason ?? raw.summary ?? fallback.explanation),
    needsConfirmation: Boolean(
      raw.needsConfirmation ??
        raw.needs_confirmation ??
        raw.uncertain ??
        (forcedUncertain || confidence < 0.72 || value === 'unknown')
    ),
  };
}

function mapStatus<TValue extends string>(status: string, statusMap: Record<string, TValue | 'unknown'>) {
  if (!status) {
    return '';
  }

  return statusMap[status] ?? '';
}

function confidenceLike(value: unknown) {
  const confidence = clampConfidence(value);
  return confidence > 0 ? confidence : 0.62;
}

function isFieldUncertain(root: Record<string, unknown>, field: ScanLabelResponse['uncertainFields'][number]) {
  return normalizeUncertainFields(root.uncertainFields ?? root.uncertain_fields).includes(field);
}

export function buildParsedLabelResult(
  scan: ScanLabelResponse,
  meta: { durationMs: number; completedAt: string }
): ParsedLabelResult {
  const parsed = buildParsedFields(scan);
  const confidenceLabel = scan.needsUserConfirmation || scan.confidence < 0.76 ? 'Review needed' : 'Mostly confident';
  const notes = [
    ...scan.fabricNotes,
    scan.explanation,
    scan.uncertainFields.length ? `Needs review: ${scan.uncertainFields.join(', ')}` : '',
    scan.rawText.trim() ? `Detected label text: ${scan.rawText.trim()}` : '',
  ].filter(Boolean);

  return {
    parsed,
    durationMs: meta.durationMs,
    completedAt: meta.completedAt,
    preview: {
      garmentName: scan.nameSuggestion || 'Care Label',
      suggestedCategory: scan.fabricNotes[0] ?? 'Needs review',
      careSummary: buildCareSummary(scan),
      confidenceLabel,
      notes,
      confidenceScore: scan.confidence,
      explanation: scan.explanation,
      needsUserConfirmation: scan.needsUserConfirmation,
      uncertainFields: scan.uncertainFields,
    },
  };
}

async function buildScanMultipartBody(
  input: ScanCareLabelInput,
  deps: { fetchImpl: FetchLike; platform: string }
) {
  const formData = new FormData();
  const fileName = input.photo.fileName ?? 'care-label.jpg';
  const mimeType = input.photo.mimeType ?? 'image/jpeg';

  if (deps.platform === 'web') {
    const response = await deps.fetchImpl(input.photo.uri);
    const blob = await response.blob();
    formData.append('image', blob, fileName);
  } else {
    formData.append('image', {
      uri: input.photo.uri,
      name: fileName,
      type: mimeType,
    } as never);
  }

  if (input.clientOCR) {
    formData.append('client_ocr', JSON.stringify(input.clientOCR));
  }

  if (input.clientSymbols?.length) {
    formData.append('client_symbols', JSON.stringify(input.clientSymbols));
  }

  return formData;
}

function buildParsedFields(scan: ScanLabelResponse): ParsedGarmentFields {
  return {
    nameSuggestion: scan.nameSuggestion || 'Care Label',
    category: '',
    primaryColor: '',
    washTemperatureC: scan.care.wash.temperatureC,
    careInstructions: buildCareInstructions(scan),
    machineWashable: scan.care.wash.value === 'machine_wash' || scan.care.wash.temperatureC !== null,
    tumbleDry: scan.care.drying.value === 'tumble_dry' || scan.care.drying.value === 'tumble_dry_low',
    dryCleanOnly: scan.care.professionalCleaning.value === 'professional_clean_only',
    ironAllowed: ['allowed', 'low', 'medium', 'high'].includes(scan.care.ironing.value),
    ironTemp: ['low', 'medium', 'high'].includes(scan.care.ironing.value)
      ? (scan.care.ironing.value as 'low' | 'medium' | 'high')
      : null,
    bleachAllowed: scan.care.bleach.value === 'allowed' || scan.care.bleach.value === 'non_chlorine_only',
    fabricNotes: scan.fabricNotes,
    rawLabelText: scan.rawText,
    confidenceScore: scan.confidence,
    needsUserConfirmation: scan.needsUserConfirmation,
    uncertainFields: scan.uncertainFields,
    fieldConfidence: {
      wash: scan.care.wash.confidence,
      bleach: scan.care.bleach.confidence,
      drying: scan.care.drying.confidence,
      ironing: scan.care.ironing.confidence,
      professional_cleaning: scan.care.professionalCleaning.confidence,
    },
  };
}

function buildCareInstructions(scan: ScanLabelResponse) {
  const instructions = [];
  const { wash, bleach, drying, ironing, professionalCleaning } = scan.care;

  if (wash.value === 'machine_wash') {
    instructions.push(wash.temperatureC ? `Machine wash up to ${wash.temperatureC}C` : 'Machine washable');
  } else if (wash.value === 'hand_wash') {
    instructions.push('Hand wash');
  } else if (wash.value === 'do_not_wash') {
    instructions.push('Do not wash');
  }

  if (bleach.value === 'allowed') {
    instructions.push('Bleach allowed');
  } else if (bleach.value === 'non_chlorine_only') {
    instructions.push('Only non-chlorine bleach');
  } else if (bleach.value === 'do_not_bleach') {
    instructions.push('Do not bleach');
  }

  if (drying.value === 'tumble_dry') {
    instructions.push('Tumble dry allowed');
  } else if (drying.value === 'tumble_dry_low') {
    instructions.push('Tumble dry low');
  } else if (drying.value === 'line_dry') {
    instructions.push('Line dry');
  } else if (drying.value === 'dry_flat') {
    instructions.push('Dry flat');
  } else if (drying.value === 'do_not_tumble_dry') {
    instructions.push('Do not tumble dry');
  }

  if (ironing.value === 'allowed') {
    instructions.push('Iron allowed');
  } else if (['low', 'medium', 'high'].includes(ironing.value)) {
    instructions.push(`Iron on ${ironing.value} heat`);
  } else if (ironing.value === 'do_not_iron') {
    instructions.push('Do not iron');
  }

  if (professionalCleaning.value === 'dry_clean') {
    instructions.push('Dry clean allowed');
  } else if (professionalCleaning.value === 'professional_clean_only') {
    instructions.push('Professional clean only');
  } else if (professionalCleaning.value === 'do_not_dry_clean') {
    instructions.push('Do not dry clean');
  }

  return instructions;
}

function buildCareSummary(scan: ScanLabelResponse) {
  const instructions = buildCareInstructions(scan);
  if (instructions.length === 0) {
    return 'Care instructions need review.';
  }

  return instructions.join('. ');
}

function normalizeField<TValue extends string>(rawValue: unknown, fallback: ScanField<TValue>): ScanField<TValue> {
  const raw = asRecord(rawValue);
  if (!raw) {
    return fallback;
  }

  const value = stringValue(raw.value || raw.instruction || fallback.value) as TValue | 'unknown';
  const confidence = clampConfidence(raw.confidence ?? fallback.confidence);
  return {
    value,
    confidence,
    explanation: stringValue(raw.explanation ?? raw.reason ?? fallback.explanation),
    needsConfirmation: Boolean(
      raw.needsConfirmation ?? raw.needs_confirmation ?? raw.uncertain ?? (confidence < 0.72 || value === 'unknown')
    ),
  };
}

function normalizeUncertainFields(value: unknown) {
  return stringArray(value)
    .map((field) => field.trim().split('.')[0])
    .filter((field): field is ScanLabelResponse['uncertainFields'][number] =>
      ['wash', 'bleach', 'drying', 'ironing', 'professional_cleaning', 'fabric', 'raw_text', 'name'].includes(field)
    );
}

function inferUncertainFields(care: ScanLabelCareInstructions): ScanLabelResponse['uncertainFields'] {
  const values: ScanLabelResponse['uncertainFields'] = [];
  if (care.wash.needsConfirmation) values.push('wash');
  if (care.bleach.needsConfirmation) values.push('bleach');
  if (care.drying.needsConfirmation) values.push('drying');
  if (care.ironing.needsConfirmation) values.push('ironing');
  if (care.professionalCleaning.needsConfirmation) values.push('professional_cleaning');
  return values;
}

function averageCareConfidence(care: ScanLabelCareInstructions) {
  const values = [
    care.wash.confidence,
    care.bleach.confidence,
    care.drying.confidence,
    care.ironing.confidence,
    care.professionalCleaning.confidence,
  ];
  return values.reduce((total, value) => total + value, 0) / values.length;
}

function buildExplanation(care: ScanLabelCareInstructions, clientOCR: ClientOCRResult | null) {
  const detectedText = clientOCR?.text.trim();
  if (detectedText) {
    return `Interpreted from on-device OCR text and the cropped label image.`;
  }

  const uncertainCount = inferUncertainFields(care).length;
  return uncertainCount > 0
    ? 'Some care fields were not confidently visible in the cropped label image.'
    : 'Interpreted from the cropped label image.';
}

function asRecord(value: unknown): Record<string, unknown> | null {
  if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
    return value as Record<string, unknown>;
  }

  return null;
}

function getFirst(record: Record<string, unknown> | null, keys: string[]) {
  if (!record) {
    return null;
  }

  for (const key of keys) {
    if (record[key] !== undefined) {
      return record[key];
    }
  }

  return null;
}

function asNullableNumber(value: unknown): number | null {
  if (value === null || value === undefined || value === '') {
    return null;
  }

  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

function clampConfidence(value: unknown) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    return 0;
  }

  return Math.max(0, Math.min(1, parsed));
}

function stringValue(value: unknown) {
  return typeof value === 'string' ? value.trim() : '';
}

function nullableString(value: unknown) {
  const nextValue = stringValue(value);
  return nextValue ? nextValue : null;
}

function stringArray(value: unknown) {
  if (!Array.isArray(value)) {
    return [];
  }

  return value.map((entry) => stringValue(entry)).filter(Boolean);
}

function inferRuntimePlatform() {
  return typeof document !== 'undefined' ? 'web' : 'native';
}
