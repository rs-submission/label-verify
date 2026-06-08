# Install And Run Locally

This guide runs the label verification prototype on a local machine. The Docker Compose path is the recommended setup.

## Prerequisites

- Docker Desktop or another Docker Compose compatible runtime
- Make

For direct, non-Docker development:

- Go 1.25 or newer
- Python 3.13 compatible environment
- `uv`
- Postgres 16

For GCP deployment:

- Bash. Prefer running deployment with `bash scripts/deploy_gcp_interactive.sh` or executing the script directly.
- Google Cloud CLI (`gcloud`): https://cloud.google.com/sdk/docs/install
- Authenticated Google Cloud CLI:

```sh
gcloud auth login
gcloud config set project YOUR_PROJECT_ID
```

- A GCP project with billing enabled.
- Required APIs enabled in that project:
  - `run.googleapis.com`
  - `artifactregistry.googleapis.com`
  - `cloudbuild.googleapis.com`
  - `sqladmin.googleapis.com`
  - `aiplatform.googleapis.com`
- Existing Artifact Registry Docker repository in the deployment region.
- Existing Cloud SQL PostgreSQL instance, database, and user.
- The Cloud SQL database password available during deployment.
- IAM permission to submit Cloud Build jobs, push to Artifact Registry, deploy Cloud Run services, attach Cloud SQL to Cloud Run, and grant `roles/aiplatform.user` to the Judge service account.

The deployment helper exposes the API as the public app entrypoint. For this prototype it also exposes OCR and Judge with public Cloud Run ingress so the API can call their `run.app` URLs directly. A hardened production deployment should replace that with authenticated service-to-service calls or private networking.

The deployment helper defaults `VERIFY_TIMEOUT_MS` to `7000` for GCP. Local development still defaults to `5000` unless you override it.

## Recommended: Docker Compose

From the repository root:

```sh
make dev
```

This runs:

- Local UI at `http://localhost:8080`
- Postgres at `localhost:5432`
- OCR service at `http://localhost:8001`
- API service at `http://localhost:8080`

The API runs database migrations automatically on startup.

Stop the stack:

```sh
docker compose down
```

Reset local database data:

```sh
docker compose down -v
```

## Smoke Test

Check the OCR service:

```sh
curl -sS http://localhost:8001/healthz
```

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

Verify a local image file:

```sh
curl -sS -X POST http://localhost:8080/api/verify \
  -F application_id=generated-bourbon-with-contents \
  -F image=@evals/images/generated_bourbon_with_contents.png
```

Supported upload extensions are `.jpg`, `.jpeg`, `.png`, and `.webp`. Uploads are capped at 10 MB.

The fixture smoke test should return a `consistent` verdict. If the application fields do not match the uploaded image text, the API returns `flagged`.

## Batch Verification

Batch verification is exposed for backend/terminal use first. A batch may contain up to 100 records. Each record keeps the same per-image verification budget used by single-image verification.

Preferred local command:

```sh
cd services/api
go run ./cmd/batch \
  --manifest ../../evals/batches/sample.json \
  --api http://localhost:8080 \
  --concurrency 1 \
  --judge-enabled=false \
  --out ../../evals/batches/results.json
```

Manifest files may be either a JSON array or an object with an `items` array:

```json
{
  "items": [
    {
      "id": "row-1",
      "application_id": "generated-rye-whiskey",
      "image": "../images/generated_rye_whiskey.png"
    },
    {
      "id": "row-2",
      "application_id": "generated-apple-brandy",
      "image": "../images/generated_apple_brandy.png",
      "application": {
        "Brand": "POM CREEK DISTILLING COMPANY, LLC",
        "ClassType": "Apple Brandy",
        "NetContents": "750 mL",
        "ABV": "40% ALC/VOL (80 Proof)",
        "GovernmentWarning": "GOVERNMENT WARNING",
        "NameAndAddress": "POM CREEK DISTILLING COMPANY, LLC LOUDOUN COUNTY, VA",
        "ForeignBlocks": []
      }
    }
  ]
}
```

If an item includes `application`, the command upserts that application before starting the batch. Image paths are resolved relative to the manifest file.

The underlying HTTP endpoint is:

```sh
curl -sS -X POST http://localhost:8080/api/verify/batch \
  -F 'items=[{"id":"generated-bourbon-with-contents","application_id":"generated-bourbon-with-contents","image_field":"image_1"}]' \
  -F image_1=@evals/images/generated_bourbon_with_contents.png \
  -F max_concurrency=1 \
  -F judge_enabled=false
```

The response contains aggregate counts and one result per record. One failed record does not fail the whole batch.

The browser UI intentionally runs batch rows as individual `/api/verify` requests instead of one large multipart request. This avoids Cloud Run and proxy request-size limits when several label images are 6-7 MB each.

Start with concurrency `1` on local CPU to avoid pushing individual OCR calls past the local 5s cap. Remote GCP deployment currently uses a 7s cap. Raise `--concurrency` / `max_concurrency` only after benchmarking the target environment.

In the local UI, application IDs are generated from uploaded image filenames. For example, `generated_bourbon_with_contents.png` becomes `generated-bourbon-with-contents`.

The current UI/prototype flow supports English-language label text only. There is no language selector in the UI; API language fields are legacy/advisory and should be left unset for local use.

## Run Tests

From the repository root:

```sh
make test
```

Equivalent direct commands:

```sh
cd services/api
GOCACHE=$(pwd)/../../.cache/go-build go test ./...

cd ../ocr
UV_CACHE_DIR=$(pwd)/../../.cache/uv uv run pytest
```

## Direct Service Development

Start Postgres:

```sh
docker compose up db
```

Start OCR:

```sh
cd services/ocr
uv sync --frozen
uv run uvicorn app:app --host 0.0.0.0 --port 8001
```

Start API in another terminal:

```sh
cd services/api
DATABASE_URL='postgres://labelverify:labelverify@localhost:5432/labelverify?sslmode=disable' \
OCR_ADDR='http://localhost:8001' \
MIGRATIONS_DIR='migrations' \
PORT='8080' \
go run ./cmd/api
```

## Local Model Files

The OCR service uses local ONNX files in `services/ocr/models`. They are copied into the OCR Docker image and loaded explicitly at runtime.

The prototype does not download OCR model files at runtime.

## Local AI Label Reader

The prototype can optionally use a local Pydantic-validated reader service backed by an Ollama model to extract structured label text in parallel with deterministic OCR. The reader is disabled by default and is only applied to selected fields that fail deterministic matching.

Current model split:

- Local / Docker Compose: `gemma4:latest` through Ollama.
- GCP / Cloud Run: Vertex AI with `gemini-2.5-flash`, using the Vertex `global` location by default.

```sh
VERIFY_TIMEOUT_MS=5000
LLM_ENABLED=false
LLM_MODE=shadow
LLM_ADDR=http://localhost:8002
LLM_MODEL=gemma4:latest
LLM_TIMEOUT_MS=2500
LLM_ALLOWED_FIELDS=brand,class_type
LLM_DENIED_FIELDS=abv,net_contents,government_warning,name_address
LLM_MIN_ELIGIBLE_FAILING_FIELDS=1

JUDGE_BASE_URL=http://localhost:11434
JUDGE_PROVIDER=ollama
JUDGE_MODEL=gemma4:latest
JUDGE_TIMEOUT_MS=2500
```

Default policy:

- Each single-image verification has a default local end-to-end budget of `VERIFY_TIMEOUT_MS=5000`. Remote GCP deployment uses `VERIFY_TIMEOUT_MS=7000` by default.
- In the UI, the operator selects which fields may be reconciled by the AI Label Reader for each single or batch run.
- UI and batch command runs use reader output only for failing selected fields. If deterministic OCR and matching pass every field, the reader is reported as not needed. Score-window gating is reserved for eval experiments only.
- For non-UI clients that do not send an explicit field list, the environment defaults still apply.
- Reader calls time out after `LLM_TIMEOUT_MS` milliseconds by default. The overall request is still bounded by `VERIFY_TIMEOUT_MS`; enabling reader review spends part of that budget on model extraction.
- `LLM_MODE=shadow` compares reader output without changing the final verdict. `LLM_MODE=override` can allow deterministic rechecks of reader output to pass selected failed fields, but should only be used after eval coverage supports it.

Recommended local model for this prototype is `gemma4:latest` via Ollama because it has proven fast on the label-image extraction prompt and is strong enough for typed output. Pull it locally before enabling reader review:

```sh
ollama pull gemma4
ollama serve
```

For Docker Compose, the reader service uses `JUDGE_PROVIDER=ollama` and reaches Ollama on the host through `JUDGE_BASE_URL=http://host.docker.internal:11434`. In GCP, set `JUDGE_PROVIDER=google-cloud` and point the service directly at Vertex AI with `JUDGE_PROJECT`, `JUDGE_LOCATION`, and a Gemini `JUDGE_MODEL`. The API reaches the service through `LLM_ADDR=http://judge:8002` locally. Then set:

```sh
LLM_ENABLED=true
```

If the reader service is unavailable, the configured Ollama model is missing, Ollama times out, Pydantic output validation fails, or the reader output does not pass deterministic recheck, the API falls back to the deterministic result.

For GCP, the deployment helper defaults `JUDGE_LOCATION=global`, `JUDGE_TIMEOUT_MS=2500`, and retries Vertex `429 RESOURCE_EXHAUSTED` responses briefly before reporting AI Reader as rate-limited. Override `JUDGE_LOCATION`, `JUDGE_TIMEOUT_MS`, `JUDGE_RETRY_ATTEMPTS`, or `JUDGE_RETRY_INITIAL_MS` only after benchmarking the target project and model.

## Troubleshooting

If API startup fails with a database connection error, wait for Postgres to finish booting and restart the API container:

```sh
docker compose restart api
```

If OCR verification fails with `unsupported content-type`, confirm the uploaded file has a supported image extension and that the file bytes are a valid image.

If local tests cannot find `uv`, install it first or use the Docker Compose workflow for running the services.

If Docker hangs while loading base image metadata, Docker Desktop's credential helper may be stuck. `make dev` uses a repo-local Docker config under `.cache/docker-config` to bypass that helper for this project's public base images.
