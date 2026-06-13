import { manipulateAsync, SaveFormat } from 'expo-image-manipulator';

import { SelectedLabelPhoto } from '@/features/add-garment/types';
import { CropRegion } from '@/features/scan-label/types';

export const DEFAULT_LABEL_CROP: CropRegion = {
  originXRatio: 0.08,
  originYRatio: 0.32,
  widthRatio: 0.84,
  heightRatio: 0.34,
};

export function clampCropRegion(region: CropRegion): CropRegion {
  const widthRatio = clamp(region.widthRatio, 0.22, 1);
  const heightRatio = clamp(region.heightRatio, 0.16, 0.78);
  const originXRatio = clamp(region.originXRatio, 0, 1 - widthRatio);
  const originYRatio = clamp(region.originYRatio, 0, 1 - heightRatio);

  return {
    originXRatio,
    originYRatio,
    widthRatio,
    heightRatio,
  };
}

export function nudgeCropRegion(region: CropRegion, delta: Partial<CropRegion>): CropRegion {
  return clampCropRegion({
    originXRatio: region.originXRatio + (delta.originXRatio ?? 0),
    originYRatio: region.originYRatio + (delta.originYRatio ?? 0),
    widthRatio: region.widthRatio + (delta.widthRatio ?? 0),
    heightRatio: region.heightRatio + (delta.heightRatio ?? 0),
  });
}

export function cropCoverage(region: CropRegion) {
  const clamped = clampCropRegion(region);
  return clamped.widthRatio * clamped.heightRatio;
}

export async function cropLabelImage(photo: SelectedLabelPhoto, region: CropRegion): Promise<SelectedLabelPhoto> {
  const clamped = clampCropRegion(region);
  const crop = {
    originX: Math.round(photo.width * clamped.originXRatio),
    originY: Math.round(photo.height * clamped.originYRatio),
    width: Math.max(1, Math.round(photo.width * clamped.widthRatio)),
    height: Math.max(1, Math.round(photo.height * clamped.heightRatio)),
  };

  if (crop.originX + crop.width > photo.width) {
    crop.width = photo.width - crop.originX;
  }

  if (crop.originY + crop.height > photo.height) {
    crop.height = photo.height - crop.originY;
  }

  const result = await manipulateAsync(photo.uri, [{ crop }], {
    compress: 0.86,
    format: SaveFormat.JPEG,
  });

  return {
    ...photo,
    uri: result.uri,
    fileName: replaceExtension(photo.fileName ?? 'care-label.jpg', 'jpg'),
    mimeType: 'image/jpeg',
    width: result.width,
    height: result.height,
    fileSize: null,
  };
}

function replaceExtension(fileName: string, nextExtension: string) {
  const cleanExtension = nextExtension.replace(/^\./, '');
  const withoutExtension = fileName.replace(/\.[^.]+$/, '');
  return `${withoutExtension}-crop.${cleanExtension}`;
}

function clamp(value: number, min: number, max: number) {
  if (Number.isNaN(value)) {
    return min;
  }

  return Math.min(max, Math.max(min, value));
}
