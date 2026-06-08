# Alcohol Label Verification Summary

This codebase implements an AI-assisted alcohol label verification prototype. It uses local OCR models to extract text from label images, deterministic matching logic to compare that text against submitted application data, and an optional AI Label Reader that can run in parallel with OCR as a second image reader.

Current UI/prototype scope is English-language label text only. Non-English and non-Latin behavior is not defined for the prototype, and the UI does not expose language configuration.

Short description:

> AI-powered OCR with deterministic compliance matching and optional parallel AI label reading for selected failed fields.

## Identified Requirements

- **Per-Image Response Budget** — Local single-image verification defaults to a 5-second budget. Remote GCP deployment uses a 7-second budget to account for Cloud Run, OCR warm-up, and network overhead. The budget is enforced via `VERIFY_TIMEOUT_MS`.
- **Full Verbatim Government Warning** — The mandatory government warning is fixed statutory text and must match **verbatim**: the complete required statement, with the "GOVERNMENT WARNING" header in exact uppercase, tolerating only OCR character noise — not wording changes, omissions, reordering, or paraphrase. This is stricter than ordinary fuzzy field matching; semantic/paraphrase equivalence is never accepted.
- **Visual Representation Check** — Ensure the label image visually represents the submitted application data (the image's fields correspond to the application record being verified).
- **UI/UX for Non-Technical Users** — The interface must be usable by non-technical reviewers, not just engineers.
- **Single & Batch Upload Support** — Support both single-image verification and batch upload/verification of multiple labels.
- **In-Boundary Processing** — All processing must stay within the same cloud environment boundary. LLM/ML API usage is acceptable provided the services reside in-boundary (e.g., Azure OpenAI, GCP Vertex AI); external third-party endpoints are not permitted, as they are blocked by the corporate firewall.


## Why It Is AI-Powered

The stack uses AI in two places:

1. **OCR / computer vision**

   The Python OCR service uses local ONNX OCR models through a RapidOCR/PaddleOCR-style recognition path. This is the primary AI component: it converts label images into structured text regions with bounding boxes and confidence scores.

2. **Optional Parallel AI Label Reader**

The system can optionally call a Pydantic-validated reader service backed by Ollama locally or Vertex AI remotely. When enabled, the Go API starts the AI image read while the deterministic OCR request is also running. After deterministic OCR/matching produces field results, the API looks only at selected fields that failed deterministic matching. AI output is then treated as alternate extracted label text and rechecked by the same deterministic field matcher. If deterministic matching passes every field, the AI result is reported as not needed and does not affect the verdict. The feature is disabled by default and compare-only by default; reconcile mode can allow matching AI-reader evidence to pass selected failed fields.

Current model split:

- Local / Docker Compose: `gemma4:latest` through Ollama.
- GCP / Cloud Run: Vertex AI with `gemini-2.5-flash`, using the Vertex `global` location by default to improve availability and reduce shared-capacity 429s.

If Vertex AI is slow, quota-constrained, or returns unreadable structured output, the AI Label Reader may retry briefly and then show as unavailable, rate-limited, timed out, or unusable. The deterministic OCR and verification path should still return normally, with the reader issue recorded as field-level AI Reader metadata rather than treated as a verification failure.

## Execution Model

Single-label verification follows this shape:

1. Load the saved application data.
2. If AI Label Reader is enabled, start the AI image-read request in parallel.
3. Run OCR through the local OCR service.
4. Match OCR text against the application fields with deterministic rules.
5. For selected fields that failed, compare the AI reader's extracted value.
6. Re-run deterministic field matching against the AI reader value.
7. In compare-only mode, record the AI reader evidence without changing the verdict.
8. In reconcile mode, allow only deterministic rechecks of AI reader text to change selected failed fields to pass.
9. Aggregate the final field results and persist the verification result.

This means the AI Label Reader is parallel in timing but gated in authority: it may provide alternate text evidence, but it does not directly decide compliance.

## What Remains Deterministic

The core verification decision is still primarily deterministic:

- The Go API orchestrates the workflow.
- OCR text is matched against application data.
- The AI reader is only a second source of extracted label text; each proposed value is still rechecked by the deterministic field matcher before it can affect a result.
- Compare-only mode records AI evidence without changing pass/fail.
- Reconcile mode can change only selected failed fields, and only after deterministic recheck passes.
- The matcher uses normalization, format parsing, class/type taxonomy, warning-header logic, and assignment/candidate selection.
- Results include field-level pass/fail, extracted text, score, and diff.
