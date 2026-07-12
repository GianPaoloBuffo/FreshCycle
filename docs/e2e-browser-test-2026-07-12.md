# Browser E2E and care-label accuracy test — 2026-07-12

## Environment

- Expo web app at `http://localhost:8081`
- Go API at `http://localhost:8080`
- Fresh local Supabase database with all checked-in migrations
- `LABEL_PARSER_PROVIDER=ocr`
- Tesseract 5.5.2 with English and Spanish language data
- Synthetic local account and test-only garment/schedule records

The downloaded images were used only as temporary local test data and were not committed to the repository.

## Browser workflow results

- Sign up and persistent sign-in: passed.
- Assisted label scan from a browser-selected image: passed.
- Structured review, private label upload, and garment save: passed.
- Wardrobe flat, temperature, and special-care grouping: passed.
- Smart load planning and mixed-category conflict detection: passed.
- Schedule create, list, delete, and due-today display: passed after the CORS fix described below.
- Scan feedback persistence: passed; review decisions produced field-accuracy and active-learning records.

The automated browser surface does not expose native file uploads. The existing Expo picker was exercised by temporarily supplying a browser `File` object created from a locally served test image; no production code was changed for this injection.

## Accuracy set

The strict score treats a field as correct only when its important modifier is also correct, such as wash temperature or tumble-dry heat.

| Image | Expected care fields | OCR-only result | Strict score |
| --- | --- | --- | ---: |
| [Wikimedia shirt care label](https://commons.wikimedia.org/wiki/File:Laundry_symbols_on_a_care_label_attached_to_a_shirt.JPG) | machine wash 30C; non-chlorine bleach; tumble dry low; warm/medium iron; dry clean P | First four correct after rule fixes; dry-clean symbol remained unknown | 4/5 |
| [Wikimedia modern care label](https://commons.wikimedia.org/wiki/File:CARE_LABEL.jpg) | machine wash 30C; no bleach; no-heat drying; medium iron; no dry clean | Wash status found but 30C missed; bleach, iron, and no-dry-clean correct; no-heat drying unsupported | 3/5 |
| [Wikimedia cryptic label](https://commons.wikimedia.org/wiki/File:Cryptic_clothing_label.jpg) | wash 30C; no bleach; no tumble dry; low iron; dry clean only | Perspective/blur caused unusable OCR; all care fields unknown | 0/5 |
| [Wikimedia wool label](https://commons.wikimedia.org/wiki/File:Label_with_care_symbols.JPG) | hand wash cold; no bleach; low iron; dry clean P | Woven white-on-black text and symbols caused unusable OCR | 0/4 |
| [Etsy two-label product image](https://www.etsy.com/listing/248093885/printed-satin-care-clothing-labels) | machine wash cold; no bleach; tumble dry low; cool iron | Angled duplicate labels and textured background caused unusable OCR | 0/4 |
| [Starlight Labels product image](https://starlightlabels.com/products/machine-wash-cold-tumble-dry-low-qty-100) | machine wash cold; tumble dry low | Both instructions correct after noisy-temperature rule fix | 2/2 |

Strict result: **9/25 expected instruction fields (36%)** on this small, deliberately varied set.

This is not a statistically representative production metric. It is a regression-oriented smoke set showing that OCR works on clear, upright text but degrades sharply with perspective, woven text, symbol-only evidence, duplicated labels, and busy backgrounds.

## Defects fixed during the run

1. OCR noise between `tumble` and `dry` prevented valid tumble-dry instructions from being recognized.
2. Dryer `low heat` text incorrectly overrode a later `warm iron` instruction.
3. OCR noise between `tumble dry` and `low` lost the drying temperature.
4. Browser schedule deletion failed because CORS did not allow `DELETE`.
5. Wardrobe grouping treated `Do not dry clean` as dry-clean-only and failed to recognize `Professional clean only`.
6. Garment saves duplicated specific scan instructions with generic flags.
7. The installed Tesseract integration test used a PNG fixture rejected by current libpng; the test now generates a valid PNG.

## Remaining accuracy gaps

- No trained symbol detector is present, so symbol-only instructions are usually invisible to the OCR-only route.
- The schema cannot represent tumble drying with no heat/air only.
- Browser image picking does not provide a real crop editor even though the UI asks for a label-only crop.
- Perspective correction, background removal, contrast normalization, and woven-label preprocessing are not implemented.
- OCR-only confidence can be high even when a visible symbol modifier, such as 30C, is missed.
- The hybrid provider was not tested because no Gemini key was configured for this local run.
