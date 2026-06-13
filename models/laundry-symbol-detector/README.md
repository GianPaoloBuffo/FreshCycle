# Laundry Symbol Detector Artifacts

Detector artifacts are versioned by directory.

- `v0.1.0`: integrated deterministic taxonomy/rules detector. This is the backend-friendly artifact used by the current scan pipeline while the first labelled image dataset is being assembled.
- `v0.2.0`: planned real-world neural detector candidate. It must not be promoted until a licensed dataset manifest, held-out evaluation run, latency benchmark, and privacy review are complete.

Future neural exports should add `model.onnx`, `model.tflite`, or `model.mlpackage` next to the version manifest after training. Do not commit third-party datasets or large model weights without confirming license and repository-size impact.

Use `manifest.schema.json` for future detector manifests. Production model iterations and evaluation runs are tracked in Supabase tables added by `20260613184616_gp53_scan_accuracy_loop.sql`.
