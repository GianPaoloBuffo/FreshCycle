import { describe, expect, it } from 'vitest';

import { buildParsedLabelResult, normalizeScanLabelResponse } from './scanCareLabel';

describe('normalizeScanLabelResponse', () => {
  it('maps the Go scan-label status schema into review values', () => {
    const scan = normalizeScanLabelResponse({
      wash: {
        status: 'machine_wash',
        max_temperature_c: 30,
        cycle: null,
        summary: 'Machine wash at or below 30C.',
      },
      bleach: {
        status: 'do_not_bleach',
        kind: null,
        summary: 'Do not bleach.',
      },
      drying: {
        status: 'tumble_dry',
        temperature: 'low',
        summary: 'Tumble dry on low heat.',
      },
      ironing: {
        status: 'iron_allowed',
        temperature: 'low',
        summary: 'Iron on low heat.',
      },
      professional_cleaning: {
        status: 'do_not_dry_clean',
        method: null,
        summary: 'Do not dry clean.',
      },
      raw_text: 'Machine wash 30C. Do not bleach. Tumble dry low.',
      confidence: 0.82,
      explanation: 'FreshCycle inferred structured care instructions from server OCR.',
      uncertain_fields: ['wash.max_temperature_c'],
      needs_user_confirmation: true,
    });

    expect(scan.care.wash.value).toBe('machine_wash');
    expect(scan.care.wash.temperatureC).toBe(30);
    expect(scan.care.drying.value).toBe('tumble_dry_low');
    expect(scan.care.ironing.value).toBe('low');
    expect(scan.care.professionalCleaning.value).toBe('do_not_dry_clean');
    expect(scan.uncertainFields).toContain('wash');
  });

  it('accepts snake_case strict scan responses and preserves uncertainty', () => {
    const scan = normalizeScanLabelResponse({
      name_suggestion: 'Linen Shirt',
      fabric_notes: ['100% linen'],
      raw_text: 'Machine wash 30C. Do not bleach. Cool iron.',
      confidence: 0.68,
      explanation: 'OCR was readable but drying was not visible.',
      needs_user_confirmation: true,
      uncertain_fields: ['drying'],
      care_instructions: {
        wash: {
          value: 'machine_wash',
          temperature_c: 30,
          confidence: 0.91,
          explanation: 'Machine wash and 30C were visible.',
        },
        bleach: {
          value: 'do_not_bleach',
          confidence: 0.86,
          explanation: 'Do not bleach text was visible.',
        },
        drying: {
          value: 'unknown',
          confidence: 0.2,
          explanation: 'No drying symbol was visible.',
        },
        ironing: {
          value: 'low',
          confidence: 0.76,
          explanation: 'Cool iron text was visible.',
        },
        professional_cleaning: {
          value: 'unknown',
          confidence: 0.15,
          explanation: 'No dry-cleaning instruction was visible.',
        },
      },
    });

    expect(scan.nameSuggestion).toBe('Linen Shirt');
    expect(scan.care.wash.temperatureC).toBe(30);
    expect(scan.care.bleach.value).toBe('do_not_bleach');
    expect(scan.uncertainFields).toContain('drying');
    expect(scan.needsUserConfirmation).toBe(true);
  });

  it('preserves backend field metadata, symbols, and cache routing details', () => {
    const scan = normalizeScanLabelResponse({
      provider: 'server_ocr',
      route: 'local_rules',
      cache_hit: true,
      image_hash: 'abc123',
      paid_fallback_used: false,
      fallback_calls_avoided: 1,
      routing_reasons: ['image_hash_cache_hit'],
      confidence: 0.88,
      symbol_detections: [
        {
          class: 'bleach_do_not_bleach',
          label: 'do not bleach',
          confidence: 0.93,
          bounding_box: { x: 10, y: 20, width: 30, height: 40 },
          source: 'client_detector',
        },
      ],
      wash: {
        status: 'machine_wash',
        max_temperature_c: 30,
        confidence: 0.91,
        explanation: 'Tub and 30C were detected.',
        needs_confirmation: false,
      },
      bleach: {
        status: 'do_not_bleach',
        confidence: 0.93,
        explanation: 'Crossed triangle was detected.',
        needs_confirmation: false,
      },
      drying: { status: 'unknown', confidence: 0.31, explanation: 'No drying mark.', needs_confirmation: true },
      ironing: { status: 'iron_allowed', temperature: 'low', confidence: 0.82, explanation: 'One-dot iron.', needs_confirmation: false },
      professional_cleaning: { status: 'unknown', confidence: 0.25, explanation: 'No circle mark.', needs_confirmation: true },
    });

    expect(scan.provider).toBe('server_ocr');
    expect(scan.route).toBe('local_rules');
    expect(scan.cacheHit).toBe(true);
    expect(scan.imageHash).toBe('abc123');
    expect(scan.fallbackCallsAvoided).toBe(1);
    expect(scan.routingReasons).toContain('image_hash_cache_hit');
    expect(scan.care.wash.confidence).toBe(0.91);
    expect(scan.care.wash.explanation).toBe('Tub and 30C were detected.');
    expect(scan.care.drying.needsConfirmation).toBe(true);
    expect(scan.symbolDetections[0]).toEqual({
      className: 'bleach_do_not_bleach',
      label: 'do not bleach',
      confidence: 0.93,
      boundingBox: { x: 10, y: 20, width: 30, height: 40 },
      source: 'client_detector',
    });
  });
});

describe('buildParsedLabelResult', () => {
  it('maps scan fields into the add-garment review model', () => {
    const scan = normalizeScanLabelResponse({
      nameSuggestion: 'Navy Hoodie',
      fabricNotes: ['80% cotton'],
      rawText: 'Tumble dry low. Do not dry clean.',
      confidence: 0.84,
      care: {
        wash: { value: 'machine_wash', temperatureC: 40, confidence: 0.9, explanation: '' },
        bleach: { value: 'do_not_bleach', confidence: 0.9, explanation: '' },
        drying: { value: 'tumble_dry_low', confidence: 0.85, explanation: '' },
        ironing: { value: 'unknown', confidence: 0.4, explanation: '' },
        professionalCleaning: { value: 'do_not_dry_clean', confidence: 0.8, explanation: '' },
      },
    });

    const parsed = buildParsedLabelResult(scan, {
      completedAt: '2026-06-13T10:00:00.000Z',
      durationMs: 250,
    });

    expect(parsed.parsed.washTemperatureC).toBe(40);
    expect(parsed.parsed.tumbleDry).toBe(true);
    expect(parsed.parsed.careInstructions).toContain('Tumble dry low');
    expect(parsed.parsed.fieldConfidence?.ironing).toBe(0.4);
    expect(parsed.preview.confidenceLabel).toBe('Review needed');
  });
});
