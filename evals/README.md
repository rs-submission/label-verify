# Label Verification Evals

This directory contains offline evaluation cases for the label verification pipeline.

The first eval mode uses cached OCR regions, so matcher and verification changes can be tested quickly and deterministically without rerunning OCR. Latency metrics in this mode are matcher/verifier runtime only; they do **not** represent OCR or preprocessing latency.

Run from the repository root:

```sh
make eval
```

Reports are written to:

```text
evals/reports/latest.json
evals/reports/latest.md
```

Case files live in `evals/cases/*.json`. Each case includes:

- application payload
- cached OCR regions
- expected final status
- expected pass/fail outcome by field

The eval runner tracks false passes and false flags separately. False passes are especially important for regulated fields such as `abv`, `net_contents`, and `government_warning`.
