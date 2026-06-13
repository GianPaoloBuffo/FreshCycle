# Production OCR Accuracy Loop

This document maps GP-53 and its sub-issues to the production data loop now supported by the API, database, app scanner, and detector manifests.

## Implementation Order

1. Privacy and retention rules are defined before production collection starts.
2. Every authenticated `/scan-label` call records scan telemetry and failure signals.
3. Ambiguous scans enter a review queue with cropped-label metadata, OCR output, detector output, model result, and review reasons.
4. Corrected or needs-label review decisions promote examples into annotation candidates.
5. Detector iterations and evaluation runs are versioned against dataset manifests and held-out real-world scans.
6. Review feedback writes per-field accuracy events for wash temperature, wash cycle, hand wash, bleach, tumble dry, natural drying, iron, dry clean, and wet clean.
7. Active-learning candidates prioritize low-confidence scans, user corrections, conflicting detections, fallback routes, and uncommon symbol classes.

## Privacy And Retention

`scan_retention_policies` stores separate `production`, `debug`, and `test` policies. Production policy prefers `label_crop`, prohibits full-photo storage, and separates retention windows for cropped image references, OCR text, image hashes, annotations, and review records.

The scan endpoint processes the uploaded image but does not persist image bytes. Review records may hold a cropped label storage path and OCR text while their retention window is active. Privacy-delete review decisions redact cropped paths, image hashes, OCR text, detector output, model output, annotation examples, and active-learning candidates.

## Telemetry And Review Queue

`scan_events` records outcome, confidence, route/provider, cache and fallback metadata, capture quality, uncertain fields, field confidences, symbol classes, and image hash metadata.

`scan_review_queue` is populated when a scan is failed, low-confidence, uncertain, fallback-heavy, quality-impaired, conflicting, or contains uncommon symbol classes. The app also records `accept` or `correct` decisions after the garment review form is saved. A high-confidence scan that the user corrects can therefore still enter the review and active-learning loop.

## Annotation Loop

`scan_annotation_examples` and `scan_symbol_annotations` hold annotation candidates, bounding boxes, modifiers, quality state, class counts, imbalance buckets, and privacy-delete status. Corrected and needs-label review decisions are promoted as annotation candidates.

Dataset files remain outside git. Commit only manifests, schemas, and small documentation updates unless a dataset license explicitly allows redistribution.

## Model Iterations And Evaluation

`detector_model_iterations` tracks versioned detector runs, training config, artifact paths, model size, latency, and promotion state. `detector_evaluation_runs` records held-out real-world metrics, false positives, false negatives, latency, model size, and pass/fail decisions.

Promote a detector only when the held-out evaluation improves target metrics without unacceptable latency or size regression. Keep the previous promoted version available for rollback.

## Field Accuracy

`scan_field_accuracy_events` separates user-confirmed accuracy by field and error source. `scan_field_accuracy_summary` aggregates daily per-user field accuracy, average confidence, OCR error count, symbol-detection error count, rule-interpretation error count, and user-correction count.

The app currently derives review feedback from the garment review form. The form does not expose every care-label nuance, so wet-clean and wash-cycle values are conservative until richer review controls exist.

## Active Learning

`active_learning_candidates` connects production scan failures to annotation examples and later model improvement. Priority increases for low confidence, user correction, field conflict, fallback route, and uncommon symbol classes such as solvent letters, wet-clean marks, cycle bars, and shade/drip drying variants.
