# Laundry Symbol Taxonomy

FreshCycle uses one shared taxonomy for detector labels, deterministic rules, and scan-label API output.

References:

- GINETEX care-symbol categories: https://www.ginetex.net/GB/labelling/care-symbols.asp
- ISO care-labelling symbol standard overview: https://www.iso.org/standard/74401.html
- Public overview of common care-symbol meanings: https://en.wikipedia.org/wiki/Laundry_symbol

## Output Fields

The scan API maps symbols into these structured fields:

- `wash`: machine wash, hand wash, do not wash, dry-clean-only wash restriction, max Celsius temperature, cycle.
- `bleach`: allowed, non-chlorine only, do not bleach.
- `drying`: tumble dry, tumble dry heat, do not tumble dry, line dry, dry flat.
- `ironing`: allowed, low, medium, high, do not iron.
- `professional_cleaning`: dry clean allowed, dry clean only, do not dry clean, wet clean.

Every field includes confidence, explanation, and `needs_confirmation`.

## Detector Classes

Wash:

- `wash_tub`
- `wash_temperature_20`, `wash_temperature_30`, `wash_temperature_40`, `wash_temperature_50`, `wash_temperature_60`, `wash_temperature_70`, `wash_temperature_80`, `wash_temperature_90`, `wash_temperature_95`
- `wash_hand`
- `wash_do_not_wash`
- `wash_gentle`
- `wash_very_gentle`

Bleach:

- `bleach_allowed`
- `bleach_non_chlorine`
- `bleach_do_not_bleach`

Drying:

- `dry_tumble`
- `dry_tumble_low`
- `dry_tumble_medium`
- `dry_tumble_high`
- `dry_do_not_tumble`
- `dry_line`
- `dry_flat`
- `dry_drip`
- `dry_shade`

Ironing:

- `iron_allowed`
- `iron_low`
- `iron_medium`
- `iron_high`
- `iron_do_not_iron`

Professional cleaning:

- `professional_dry_clean`
- `professional_dry_clean_p`
- `professional_dry_clean_f`
- `professional_do_not_dry_clean`
- `professional_wet_clean`
- `professional_do_not_wet_clean`

Generic modifiers:

- `modifier_cross`
- `modifier_one_dot`
- `modifier_two_dots`
- `modifier_three_dots`
- `modifier_one_bar`
- `modifier_two_bars`
- `modifier_letter_p`
- `modifier_letter_f`
- `modifier_letter_w`
- `modifier_temperature`

## Deterministic Examples

- `wash_tub` + `wash_temperature_30` + `modifier_one_bar` => `wash.status = machine_wash`, `wash.max_temperature_c = 30`, `wash.cycle = delicate`.
- `bleach_do_not_bleach` or triangle + cross => `bleach.status = do_not_bleach`.
- `dry_tumble` + `modifier_one_dot` => `drying.status = tumble_dry`, `drying.temperature = low`.
- `dry_tumble` + `modifier_cross` => `drying.status = do_not_tumble_dry`.
- `iron_allowed` + `modifier_one_dot` => `ironing.status = iron_allowed`, `ironing.temperature = low`.
- `professional_dry_clean_p` + `modifier_one_bar` => `professional_cleaning.status = dry_clean_allowed`, `professional_cleaning.method = dry_clean`, lower confidence if no text confirms solvent.

