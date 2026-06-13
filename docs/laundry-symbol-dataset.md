# Laundry Symbol Dataset Bootstrap

This document defines the first detector dataset plan for GP-62 and the handoff into model export for GP-63.

## Candidate Sources

Public discovery should start with:

- Roboflow Universe: https://universe.roboflow.com/
- GINETEX symbol references attached to Linear GP-61 and the public category reference: https://www.ginetex.net/GB/labelling/care-symbols.asp
- Internal FreshCycle test captures collected through the scanner flow.

Do not commit third-party images or labels unless their license explicitly allows redistribution. For Roboflow projects, pin the workspace, project, version, export format, license, and download date in the dataset manifest before training.

## Dataset Layout

Use YOLO-style object-detection exports so the first training run can move through YOLO, ONNX, TFLite, or CoreML export paths.

```text
datasets/laundry-symbols/v0.1.0/
  data.yaml
  train/images/
  train/labels/
  val/images/
  val/labels/
  test/images/
  test/labels/
  manifest.json
```

The dataset itself is not committed to git. Store only a manifest with source, license, class coverage, split counts, checksum, and known gaps.

## Required Coverage

Class coverage must match [the taxonomy](laundry-symbol-taxonomy.md), with priority on:

- do not bleach
- iron low
- do not tumble dry
- hand wash
- dry clean letters
- wash temperatures
- cycle bars
- natural drying lines

Image condition coverage:

- woven labels
- printed satin labels
- warped or folded tags
- low-light captures
- overexposed captures
- English, Spanish, and symbol-only labels

## Known Initial Gaps

The repo currently has no committed labelled image corpus. The integrated `v0.1.0` detector artifact is therefore deterministic and rule-backed. Before training a neural detector, collect or license at least:

- 50 labelled instances for each priority class.
- 20 examples each for low-light, warped, woven, printed, and multilingual conditions.
- A holdout set that contains full label crops, not isolated icon screenshots.

## Training And Export

Recommended first neural baseline:

```bash
pip install ultralytics
yolo detect train model=yolov8n.pt data=datasets/laundry-symbols/v0.1.0/data.yaml imgsz=640 epochs=80 batch=16
yolo export model=runs/detect/train/weights/best.pt format=onnx
yolo export model=runs/detect/train/weights/best.pt format=tflite
```

Record model size, validation mAP, median inference latency, export checksum, and dataset manifest checksum in `models/laundry-symbol-detector/<version>/manifest.json`.
