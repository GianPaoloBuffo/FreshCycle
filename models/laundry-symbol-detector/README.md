# Laundry Symbol Detector Artifacts

Detector artifacts are versioned by directory.

- `v0.1.0`: integrated deterministic taxonomy/rules detector. This is the backend-friendly artifact used by the current scan pipeline while the first labelled image dataset is being assembled.

Future neural exports should add `model.onnx`, `model.tflite`, or `model.mlpackage` next to the version manifest after training. Do not commit third-party datasets or large model weights without confirming license and repository-size impact.

