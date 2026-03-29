import { describe, expect, it, vi } from 'vitest';

import {
  AddGarmentActionError,
  createSelectedLabelPhoto,
  describeAddGarmentError,
  parseCareLabelPhoto,
} from './parseCareLabel';

describe('createSelectedLabelPhoto', () => {
  it('fills in a file name from the asset uri when one is missing', () => {
    const photo = createSelectedLabelPhoto(
      {
        uri: 'file:///tmp/care-label-shot.jpg',
        width: 1200,
        height: 1600,
        fileName: null,
        fileSize: 4096,
        mimeType: 'image/jpeg',
        type: 'image',
        assetId: null,
        duration: null,
        exif: null,
        base64: null,
        file: undefined,
        pairedVideoAsset: null,
      },
      'camera'
    );

    expect(photo.fileName).toBe('care-label-shot.jpg');
    expect(photo.source).toBe('camera');
  });
});

describe('parseCareLabelPhoto', () => {
  it('returns a deterministic preview and duration', async () => {
    const now = vi.fn().mockReturnValueOnce(100).mockReturnValueOnce(1450).mockReturnValueOnce(1450);

    const result = await parseCareLabelPhoto(
      {
        uri: 'file:///tmp/linen-shirt.png',
        fileName: 'linen-shirt.png',
        mimeType: 'image/png',
        width: 1200,
        height: 1600,
        fileSize: 8192,
        source: 'library',
      },
      {
        delayMs: 0,
        now,
      }
    );

    expect(result.durationMs).toBe(1350);
    expect(result.preview.garmentName).toBe('Linen Shirt');
    expect(result.preview.confidenceLabel).toBe('Mostly confident');
    expect(result.preview.notes[2]).toContain('Phase 2 stub');
  });
});

describe('describeAddGarmentError', () => {
  it('returns guidance that points denied camera users to the library fallback', () => {
    expect(describeAddGarmentError('camera-permission-denied')).toContain('photo library');
  });

  it('wraps processing failures in a typed add-garment error', () => {
    const error = new AddGarmentActionError('processing-failed');

    expect(error.code).toBe('processing-failed');
    expect(error.message).toBe(describeAddGarmentError('processing-failed'));
  });
});
