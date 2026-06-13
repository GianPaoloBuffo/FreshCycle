import {
  CaptureQualityHint,
  CaptureQualityInput,
  CaptureQualityIssue,
  CaptureQualityResult,
} from '@/features/scan-label/types';

const minimumStableMs = 1200;

const issueMessages: Record<CaptureQualityIssue, string> = {
  possible_blur: 'Hold still and retake if the text looks soft.',
  poor_lighting: 'Add more light or turn the label toward the light.',
  small_label_area: 'Move closer so the care label fills the guide.',
  unstable_framing: 'Hold still for a moment before capture.',
  low_ocr_signal: 'Flatten the label and make the care text visible.',
};

export function assessCaptureQuality(input: CaptureQualityInput): CaptureQualityResult {
  const hints: CaptureQualityHint[] = [];
  const shortestSide = Math.min(input.photo.width, input.photo.height);
  const fileSize = input.photo.fileSize ?? null;
  const normalizedText = (input.ocrText ?? '').replace(/\s+/g, ' ').trim();
  const wordCount = normalizedText ? normalizedText.split(' ').filter(Boolean).length : 0;

  if (input.stableForMs < minimumStableMs) {
    hints.push(buildHint('unstable_framing', true));
  }

  if (shortestSide > 0 && shortestSide < 900) {
    hints.push(buildHint('small_label_area', false));
  }

  if (input.cropCoverage < 0.18) {
    hints.push(buildHint('small_label_area', false));
  }

  if (fileSize !== null && fileSize > 0 && fileSize < 70_000) {
    hints.push(buildHint('possible_blur', false));
  }

  if (wordCount > 0 && wordCount < 3) {
    hints.push(buildHint('low_ocr_signal', false));
  }

  if ((input.ocrLineCount ?? 0) === 0 && normalizedText === '' && input.stableForMs >= minimumStableMs) {
    hints.push(buildHint('poor_lighting', false));
  }

  const uniqueHints = dedupeHints(hints);
  const score = Math.max(0, 1 - uniqueHints.reduce((total, hint) => total + (hint.blocksCapture ? 0.35 : 0.16), 0));

  return {
    score,
    hints: uniqueHints,
    canSubmit: !uniqueHints.some((hint) => hint.blocksCapture),
  };
}

export function buildLiveQualityHints(stableForMs: number): CaptureQualityHint[] {
  if (stableForMs < minimumStableMs) {
    return [buildHint('unstable_framing', true)];
  }

  return [
    {
      issue: 'small_label_area',
      message: 'Fill the guide with the label and keep it flat.',
      blocksCapture: false,
    },
  ];
}

function buildHint(issue: CaptureQualityIssue, blocksCapture: boolean): CaptureQualityHint {
  return {
    issue,
    message: issueMessages[issue],
    blocksCapture,
  };
}

function dedupeHints(hints: CaptureQualityHint[]) {
  const seen = new Set<CaptureQualityIssue>();
  return hints.filter((hint) => {
    if (seen.has(hint.issue)) {
      return false;
    }

    seen.add(hint.issue);
    return true;
  });
}
