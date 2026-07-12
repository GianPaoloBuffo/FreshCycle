export type CareInstructionFlags = {
  machineWashable: boolean;
  tumbleDry: boolean;
  dryCleanOnly: boolean;
  ironAllowed: boolean;
  ironTemp: string;
  bleachAllowed: boolean;
};

export function mergeCareInstructions(existing: string[], flags: CareInstructionFlags) {
  const instructions = Array.from(new Set(existing.map((value) => value.trim()).filter(Boolean)));
  const contains = (needle: string) => instructions.some((value) => value.toLowerCase().includes(needle));

  if (flags.machineWashable && !contains('machine wash')) {
    instructions.push('Machine washable');
  }
  if (flags.tumbleDry && !contains('tumble dry')) {
    instructions.push('Tumble dry allowed');
  }
  if (flags.dryCleanOnly && !contains('dry clean only') && !contains('professional clean only')) {
    instructions.push('Dry clean only');
  }
  if (flags.ironAllowed && !contains('iron')) {
    instructions.push(flags.ironTemp ? `Iron on ${flags.ironTemp} heat` : 'Iron allowed');
  }
  if (!flags.bleachAllowed && !contains('bleach')) {
    instructions.push('Do not bleach');
  }

  return instructions;
}
