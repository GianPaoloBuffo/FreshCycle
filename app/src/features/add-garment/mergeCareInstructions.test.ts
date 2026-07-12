import { describe, expect, it } from 'vitest';

import { mergeCareInstructions } from './mergeCareInstructions';

const baseFlags = {
  machineWashable: true,
  tumbleDry: true,
  dryCleanOnly: false,
  ironAllowed: true,
  ironTemp: 'medium',
  bleachAllowed: true,
};

describe('mergeCareInstructions', () => {
  it('does not duplicate detailed scan instructions with generic flags', () => {
    expect(
      mergeCareInstructions(
        ['Machine wash up to 30C', 'Only non-chlorine bleach', 'Tumble dry low', 'Iron on medium heat'],
        baseFlags
      )
    ).toEqual(['Machine wash up to 30C', 'Only non-chlorine bleach', 'Tumble dry low', 'Iron on medium heat']);
  });

  it('adds missing review flags when no equivalent instruction exists', () => {
    expect(mergeCareInstructions([], { ...baseFlags, bleachAllowed: false })).toEqual([
      'Machine washable',
      'Tumble dry allowed',
      'Iron on medium heat',
      'Do not bleach',
    ]);
  });

  it('recognizes the scanner wording for professional-clean-only care', () => {
    expect(
      mergeCareInstructions(['Professional clean only'], {
        ...baseFlags,
        machineWashable: false,
        tumbleDry: false,
        dryCleanOnly: true,
        ironAllowed: false,
      })
    ).toEqual(['Professional clean only']);
  });
});
