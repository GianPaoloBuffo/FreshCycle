# Laundry Symbol Datasets

Real-world care-label datasets are intentionally not committed to this repository. Store only manifests, checksums, split counts, source/license notes, and class coverage summaries here.

## Layout

```text
datasets/laundry-symbols/<dataset-version>/
  manifest.json
  data.yaml              # local only
  train/images/          # local only
  train/labels/          # local only
  val/images/            # local only
  val/labels/            # local only
  test/images/           # local only
  test/labels/           # local only
```

## Manifest Requirements

Each dataset manifest should include:

- Dataset version and creation date.
- Source list with license, download date, and whether redistribution is allowed.
- Hashes for local archives or export bundles.
- Split counts for train, validation, and held-out test.
- Class counts using `docs/laundry-symbol-taxonomy.md`.
- Annotation quality summary, including reviewer count and known disagreements.
- Privacy status for FreshCycle production examples, including redaction or deletion counts.

Use `manifest.schema.json` as the contract for future manifests.
