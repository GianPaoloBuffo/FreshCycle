import { supabase } from '@/lib/supabase';
import { SelectedLabelPhoto } from '@/features/add-garment/types';

export const LABEL_IMAGE_BUCKET = 'garment-labels';

type UploadLabelImageResult = {
  garmentId: string;
  objectPath: string;
};

type UploadLabelImageDeps = {
  fetchImpl?: typeof fetch;
  storageClient?: Pick<typeof supabase.storage, 'from'>;
  garmentId?: string;
};

export async function uploadLabelImage(
  photo: SelectedLabelPhoto,
  userId: string,
  deps: UploadLabelImageDeps = {}
): Promise<UploadLabelImageResult> {
  const fetchImpl = deps.fetchImpl ?? ((input, init) => fetch(input, init));
  const storageClient = deps.storageClient ?? supabase.storage;
  const garmentId = deps.garmentId ?? createGarmentId();
  const fileName = photo.fileName ?? inferFileNameFromUri(photo.uri);
  const extension = inferFileExtension(fileName, photo.mimeType);
  const objectPath = `${userId}/labels/${garmentId}.${extension}`;
  const body = await buildUploadBody(photo, fetchImpl);

  const { error } = await storageClient.from(LABEL_IMAGE_BUCKET).upload(objectPath, body, {
    contentType: photo.mimeType ?? inferMimeTypeFromExtension(extension),
    upsert: false,
  });

  if (error) {
    throw new Error('upload-failed');
  }

  return {
    garmentId,
    objectPath,
  };
}

export async function createSignedLabelImageUrl(
  objectPath: string,
  expiresInSeconds = 3600,
  storageClient: Pick<typeof supabase.storage, 'from'> = supabase.storage
) {
  const { data, error } = await storageClient
    .from(LABEL_IMAGE_BUCKET)
    .createSignedUrl(objectPath, expiresInSeconds);

  if (error) {
    throw new Error(error.message);
  }

  return data.signedUrl;
}

async function buildUploadBody(photo: SelectedLabelPhoto, fetchImpl: typeof fetch) {
  if (photo.webFile) {
    return await photo.webFile.arrayBuffer();
  }

  const response = await fetchImpl(photo.uri);

  if (!response.ok) {
    throw new Error('upload-failed');
  }

  return await response.arrayBuffer();
}

function createGarmentId() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }

  return `garment-${Date.now()}-${Math.random().toString(16).slice(2, 10)}`;
}

function inferFileNameFromUri(uri: string) {
  const segments = uri.split('/');
  return segments[segments.length - 1] || 'care-label.jpg';
}

function inferFileExtension(fileName: string, mimeType: string | null) {
  const fromName = fileName.split('.').pop()?.trim().toLowerCase();
  if (fromName && fromName !== fileName.trim().toLowerCase()) {
    return sanitizeExtension(fromName);
  }

  switch (mimeType) {
    case 'image/png':
      return 'png';
    case 'image/webp':
      return 'webp';
    case 'image/jpeg':
    case 'image/jpg':
    default:
      return 'jpg';
  }
}

function inferMimeTypeFromExtension(extension: string) {
  switch (extension) {
    case 'png':
      return 'image/png';
    case 'webp':
      return 'image/webp';
    case 'jpeg':
    case 'jpg':
    default:
      return 'image/jpeg';
  }
}

function sanitizeExtension(extension: string) {
  const cleaned = extension.replace(/[^a-z0-9]/g, '');
  if (!cleaned) {
    return 'jpg';
  }

  if (cleaned === 'jpeg' || cleaned === 'jpg' || cleaned === 'png' || cleaned === 'webp') {
    return cleaned;
  }

  return 'jpg';
}
