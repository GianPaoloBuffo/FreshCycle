import { ImagePickerAsset } from 'expo-image-picker';

import { logAddGarmentError, logAddGarmentEvent } from '@/features/add-garment/observability';
import { AddGarmentErrorCode, ParsedLabelResult, SelectedLabelPhoto } from '@/features/add-garment/types';

type ParseCareLabelDeps = {
  delayMs?: number;
  now?: () => number;
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
  const startedAt = now();

  logAddGarmentEvent('label_parse_started', {
    source: photo.source,
    fileName: photo.fileName,
  });

  try {
    await wait(delayMs);

    const result: ParsedLabelResult = {
      preview: buildPreview(photo),
      durationMs: now() - startedAt,
      completedAt: new Date(now()).toISOString(),
    };

    logAddGarmentEvent('label_parse_succeeded', {
      source: photo.source,
      durationMs: result.durationMs,
      garmentName: result.preview.garmentName,
    });

    return result;
  } catch (error) {
    logAddGarmentError('label_parse_failed', error, {
      source: photo.source,
      fileName: photo.fileName,
    });
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
    case 'processing-failed':
    default:
      return 'FreshCycle could not process that label just yet. Try another photo or retry in a moment.';
  }
}

function buildPreview(photo: SelectedLabelPhoto): ParsedLabelResult['preview'] {
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
