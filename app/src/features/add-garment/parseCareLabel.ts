import { ImagePickerAsset } from 'expo-image-picker';

import { logAddGarmentError, logAddGarmentEvent } from '@/features/add-garment/observability';
import { AddGarmentErrorCode, ParsedLabelResult, SelectedLabelPhoto } from '@/features/add-garment/types';
import { getAppEnv } from '@/lib/env';

type ParseCareLabelDeps = {
  delayMs?: number;
  now?: () => number;
  apiBaseUrl?: string | null;
  fetchImpl?: FetchLike;
  platform?: string;
  accessToken?: string | null;
};

type FetchLike = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

type ParseLabelApiResponse = {
  name_suggestion: string;
  fabric_notes: string[];
  wash_temp_max: number | null;
  machine_washable: boolean;
  tumble_dry: boolean;
  dry_clean_only: boolean;
  iron_allowed: boolean;
  iron_temp: 'low' | 'medium' | 'high' | null;
  bleach_allowed: boolean;
  raw_label_text: string;
};

export class AddGarmentActionError extends Error {
  readonly code: AddGarmentErrorCode;

  constructor(code: AddGarmentErrorCode, message = describeAddGarmentError(code)) {
    super(message);
    this.code = code;
  }
}

export function createSelectedLabelPhoto(
  asset: ImagePickerAsset,
  source: SelectedLabelPhoto['source']
): SelectedLabelPhoto {
  return {
    uri: asset.uri,
    fileName: asset.fileName ?? inferFileNameFromUri(asset.uri),
    mimeType: asset.mimeType ?? null,
    width: asset.width,
    height: asset.height,
    fileSize: asset.fileSize ?? null,
    source,
  };
}

export async function parseCareLabelPhoto(
  photo: SelectedLabelPhoto,
  deps: ParseCareLabelDeps = {}
): Promise<ParsedLabelResult> {
  const now = deps.now ?? Date.now;
  const delayMs = deps.delayMs ?? 1200;
  const fetchImpl = deps.fetchImpl ?? ((input, init) => fetch(input, init));
  const apiBaseUrl = deps.apiBaseUrl ?? getAppEnv().apiBaseUrl;
  const platform = deps.platform ?? inferRuntimePlatform();
  const accessToken = deps.accessToken ?? null;
  const startedAt = now();

  logAddGarmentEvent('label_parse_started', {
    source: photo.source,
    fileName: photo.fileName,
  });

  try {
    const apiResult = apiBaseUrl
      ? await parseViaAPI(photo, {
          apiBaseUrl,
          fetchImpl,
          platform,
          accessToken,
        })
      : null;

    if (!apiResult) {
      await wait(delayMs);
    }

    const result: ParsedLabelResult = apiResult
      ? {
          preview: buildPreviewFromAPI(apiResult),
          durationMs: now() - startedAt,
          completedAt: new Date(now()).toISOString(),
        }
      : {
          preview: buildStubPreview(photo),
          durationMs: now() - startedAt,
          completedAt: new Date(now()).toISOString(),
        };

    logAddGarmentEvent('label_parse_succeeded', {
      source: photo.source,
      durationMs: result.durationMs,
      garmentName: result.preview.garmentName,
      parserMode: apiResult ? 'api' : 'stub',
    });

    return result;
  } catch (error) {
    logAddGarmentError('label_parse_failed', error, {
      source: photo.source,
      fileName: photo.fileName,
    });
    if (error instanceof AddGarmentActionError) {
      throw error;
    }
    throw new AddGarmentActionError('processing-failed');
  }
}

export function describeAddGarmentError(code: AddGarmentErrorCode) {
  switch (code) {
    case 'camera-permission-denied':
      return 'Camera access is required to take a care-label photo. You can still continue from the photo library.';
    case 'photo-library-permission-denied':
      return 'Photo library access is required to choose an existing care-label image.';
    case 'camera-unavailable':
      return 'Camera capture is not available on this device. Choose a photo from your library instead.';
    case 'selection-empty':
      return 'No image was returned by the picker. Try again with a clear care-label photo.';
    case 'auth-required':
      return 'Your FreshCycle session needs attention before we can call the API. Sign in again and retry.';
    case 'processing-failed':
    default:
      return 'FreshCycle could not process that label just yet. Try another photo or retry in a moment.';
  }
}

function buildStubPreview(photo: SelectedLabelPhoto): ParsedLabelResult['preview'] {
  const baseName = sanitizeFileStem(photo.fileName ?? inferFileNameFromUri(photo.uri));
  const prettyName = titleize(baseName || 'care label upload');
  const sizeLabel = `${photo.width || '?'}x${photo.height || '?'}`;

  return {
    garmentName: prettyName,
    suggestedCategory: 'Needs review',
    careSummary: 'Label captured and queued for structured parsing. Review the extracted garment details in the next step.',
    confidenceLabel: photo.width >= 900 ? 'Mostly confident' : 'Review needed',
    notes: [
      `Captured from ${photo.source}.`,
      `Image size: ${sizeLabel}.`,
      'Parsing is currently running through a local Phase 2 stub until the API-backed parser lands.',
    ],
  };
}

function buildPreviewFromAPI(result: ParseLabelApiResponse): ParsedLabelResult['preview'] {
  const notes = [
    ...result.fabric_notes.map((note) => note.trim()).filter(Boolean),
    buildInstructionSummary(result),
  ].filter(Boolean);

  if (result.raw_label_text.trim()) {
    notes.push(`Detected label text: ${result.raw_label_text.trim()}`);
  }

  return {
    garmentName: result.name_suggestion.trim() || 'Needs review',
    suggestedCategory: result.fabric_notes[0] ?? 'Needs review',
    careSummary: buildCareSummary(result),
    confidenceLabel: result.raw_label_text.trim() ? 'Mostly confident' : 'Review needed',
    notes,
  };
}

async function parseViaAPI(
  photo: SelectedLabelPhoto,
  deps: {
    apiBaseUrl: string;
    fetchImpl: FetchLike;
    platform: string;
    accessToken: string | null;
  }
) {
  if (!deps.accessToken) {
    throw new AddGarmentActionError('auth-required');
  }

  const requestBody = await buildMultipartBody(photo, deps);
  const response = await deps.fetchImpl(`${deps.apiBaseUrl.replace(/\/$/, '')}/garments/parse-label`, {
    method: 'POST',
    body: requestBody,
    headers: {
      Authorization: `Bearer ${deps.accessToken}`,
    },
  });

  if (response.status === 401) {
    throw new AddGarmentActionError('auth-required');
  }

  if (!response.ok) {
    throw new AddGarmentActionError('processing-failed');
  }

  return (await response.json()) as ParseLabelApiResponse;
}

async function buildMultipartBody(
  photo: SelectedLabelPhoto,
  deps: Required<Pick<ParseCareLabelDeps, 'fetchImpl' | 'platform'>>
) {
  const formData = new FormData();
  const fileName = photo.fileName ?? inferFileNameFromUri(photo.uri);
  const mimeType = photo.mimeType ?? 'image/jpeg';

  if (deps.platform === 'web') {
    const response = await deps.fetchImpl(photo.uri);
    const blob = await response.blob();
    formData.append('image', blob, fileName);
    return formData;
  }

  formData.append('image', {
    uri: photo.uri,
    name: fileName,
    type: mimeType,
  } as never);
  return formData;
}

function buildCareSummary(result: ParseLabelApiResponse) {
  const instructions = [
    result.machine_washable
      ? result.wash_temp_max
        ? `Machine wash up to ${result.wash_temp_max}C`
        : 'Machine washable'
      : result.dry_clean_only
        ? 'Dry clean only'
        : 'Wash method needs review',
    result.tumble_dry ? 'Tumble dry allowed' : 'Avoid tumble drying',
    result.iron_allowed
      ? result.iron_temp
        ? `Iron on ${result.iron_temp} heat`
        : 'Iron allowed'
      : 'Do not iron',
    result.bleach_allowed ? 'Bleach allowed' : 'Do not bleach',
  ];

  return instructions.join('. ');
}

function buildInstructionSummary(result: ParseLabelApiResponse) {
  const instructions = [];

  if (result.machine_washable) {
    instructions.push(result.wash_temp_max ? `wash <= ${result.wash_temp_max}C` : 'machine washable');
  }
  if (result.dry_clean_only) {
    instructions.push('dry clean only');
  }
  if (result.tumble_dry) {
    instructions.push('tumble dry');
  }
  if (result.iron_allowed) {
    instructions.push(result.iron_temp ? `iron ${result.iron_temp}` : 'iron allowed');
  }
  if (!result.bleach_allowed) {
    instructions.push('no bleach');
  }

  return instructions.length ? `Detected care instructions: ${instructions.join(', ')}.` : '';
}

function inferRuntimePlatform() {
  return typeof document !== 'undefined' ? 'web' : 'native';
}

function sanitizeFileStem(fileName: string) {
  return fileName
    .replace(/\.[^.]+$/, '')
    .replace(/[-_]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

function inferFileNameFromUri(uri: string) {
  const lastSegment = uri.split('/').pop() ?? 'care-label';
  return lastSegment.split('?')[0] || 'care-label';
}

function titleize(value: string) {
  return value
    .split(' ')
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

function wait(delayMs: number) {
  return new Promise<void>((resolve) => {
    setTimeout(resolve, delayMs);
  });
}
