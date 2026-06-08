# Alcohol Label Verification

Prototype service for checking whether a submitted alcohol label image is textually consistent with the application data linked to it by application ID.

The system uses local OCR for perception and deterministic matching for the final consistency verdict. The API also serves a minimal local operator UI at `http://localhost:8080`.

## What Is Included

- Go API service on port `8080`
- FastAPI OCR service on port `8001`
- FastAPI/Pydantic AI Label Reader service on port `8002`
- Postgres database on port `5432`
- Local RapidOCR ONNX model files under `services/ocr/models`
- Database migrations run by the API at startup
- Unit and contract tests for OCR, matching, storage, API routing, and verification orchestration

## Current Prototype Scope

- English-language label text only in the current UI/prototype flow.
- Latin-script OCR path only; non-Latin input behavior is undefined.
- API `langs` / `DeclaredLanguages` inputs are legacy/advisory and are not exposed in the UI.
- Optional AI Label Reader reconciliation is available through a Pydantic-validated service: local deployments call Ollama directly, while GCP deployments can switch the same service to Vertex AI / Gemini with environment variables. It is only applied to selected fields that fail deterministic matching; all-pass deterministic runs report the reader as not needed.
- Batch verification is exposed through the API, terminal command, and local UI with a 100-record cap.
- The WebSocket progress endpoint is a stub.

## Quick Start

See [INSTALL.md](/Users/rs/Projects/ttb/INSTALL.md) for full setup instructions.

```sh
make dev
```

That starts:

- App: `http://localhost:8080`
- API: `http://localhost:8080`
- OCR: `http://localhost:8001`
- AI Label Reader: `http://localhost:8002`
- Postgres: `localhost:5432`

Run tests:

```sh
make test
```

## API Overview

Create or replace an application record:

```sh
curl -sS -X PUT http://localhost:8080/api/applications/generated-bourbon-with-contents \
  -H 'Content-Type: application/json' \
  -d '{
    "Brand": "POM BOURBON",
    "ClassType": "Bourbon",
    "NetContents": "750 mL",
    "ABV": "45% ALC/VOL (90 Proof)",
    "GovernmentWarning": "GOVERNMENT WARNING",
    "NameAndAddress": "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
    "ForeignBlocks": []
  }'
```

Verify a label image:

```sh
curl -sS -X POST http://localhost:8080/api/verify \
  -F application_id=generated-bourbon-with-contents \
  -F image=@evals/images/generated_bourbon_with_contents.png
```

Run a batch from a manifest, up to 100 records:

```sh
cd services/api
go run ./cmd/batch \
  --manifest ../../evals/batches/sample.json \
  --api http://localhost:8080 \
  --concurrency 1 \
  --out ../../evals/batches/results.json
```

Delete an application and its stored verification images:

```sh
curl -sS -X DELETE http://localhost:8080/api/applications/generated-bourbon-with-contents
```

Expected response shape:

```json
{
  "Status": "consistent",
  "Confidence": 0.98,
  "Fields": [
    {
      "Field": "brand",
      "Expected": "POM BOURBON",
      "Extracted": "POM BOURBON",
      "MatchType": "fuzzy",
      "Score": 1,
      "Pass": true,
      "Diff": ""
    }
  ]
}
```

The OCR service is internal to the app stack, but its contract is documented in [docs/ocr-api.md](/Users/rs/Projects/ttb/docs/ocr-api.md).
Deployment guidance is documented in [docs/deployment.md](/Users/rs/Projects/ttb/docs/deployment.md).

## Project Layout

```text
contracts/       Shared golden contract fixtures
docs/            API notes and planning docs
services/api/    Go API, matcher, persistence, verification orchestration
services/ocr/    FastAPI OCR service, preprocessing, local ONNX recognizer
```
