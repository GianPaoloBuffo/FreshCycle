import { describe, expect, it } from 'vitest';

import { assessCaptureQuality, buildLiveQualityHints } from './captureQuality';

describe('capture quality', () => {
  it('blocks capture while framing is unstable', () => {
    const hints = buildLiveQualityHints(300);

    expect(hints[0]?.issue).toBe('unstable_framing');
    expect(hints[0]?.blocksCapture).toBe(true);
  });

  it('allows usable captures while surfacing non-blocking review hints', () => {
    const quality = assessCaptureQuality({
      photo: {
        width: 1200,
        height: 1600,
        fileSize: 200_000,
      },
      cropCoverage: 0.28,
      stableForMs: 1600,
      ocrText: 'Machine wash cold do not bleach',
      ocrLineCount: 2,
    });

    expect(quality.canSubmit).toBe(true);
    expect(quality.score).toBeGreaterThan(0.7);
  });
});
