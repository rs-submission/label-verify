#!/usr/bin/env bash

if [ -z "${BASH_VERSION:-}" ]; then
  exec bash "$0" "$@"
fi

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

DRY_RUN=0
SKIP_BUILD=0

usage() {
  cat <<'EOF'
Usage: scripts/deploy_gcp_interactive.sh [--dry-run] [--skip-build]

Interactive deployment helper for the current TTB label verification stack.

The script builds/pushes api, ocr, and judge images, deploys them to Cloud Run,
configures the API to call OCR/Judge, and grants Vertex AI access to the judge
service account.

Manual prerequisites are printed at startup.
EOF
}

print_prerequisites() {
  cat <<'EOF'
Prerequisites

Local machine:
  - gcloud CLI is installed.
  - You are authenticated with gcloud:
      gcloud auth login
  - gcloud can submit Cloud Build jobs from this machine.

GCP project:
  - The target project exists and billing is enabled.
  - Required APIs are enabled:
      run.googleapis.com
      artifactregistry.googleapis.com
      cloudbuild.googleapis.com
      sqladmin.googleapis.com
      aiplatform.googleapis.com
  - Artifact Registry Docker repository already exists in the deploy region.
  - Cloud SQL PostgreSQL instance already exists.
  - Cloud SQL database and database user already exist.
  - You have the database password available.

IAM for the deploying user:
  - Permission to push/build images with Cloud Build and Artifact Registry.
  - Permission to deploy Cloud Run services.
  - Permission to attach Cloud SQL instances to Cloud Run services.
  - Permission to grant roles/aiplatform.user to the Judge service account.

Runtime/security notes:
  - The API service is deployed as the public app entrypoint.
  - OCR and Judge are deployed with public ingress so the API can call their
    run.app URLs without extra VPC or identity-token plumbing.
  - The script does not configure custom domains, IAP, Secret Manager, or
    authenticated service-to-service calls.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    --skip-build)
      SKIP_BUILD=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    if [[ "$1" == "gcloud" ]]; then
      cat >&2 <<'EOF'

Install the Google Cloud CLI before running this deployment script:
  https://cloud.google.com/sdk/docs/install

After installation, authenticate and select the target project:
  gcloud auth login
  gcloud config set project YOUR_PROJECT_ID

Then rerun:
  bash scripts/deploy_gcp_interactive.sh
EOF
    fi
    exit 1
  fi
}

prompt() {
  local label="$1"
  local default="${2:-}"
  local value
  if [[ -n "$default" ]]; then
    read -r -p "$label [$default]: " value
    printf '%s' "${value:-$default}"
  else
    read -r -p "$label: " value
    printf '%s' "$value"
  fi
}

prompt_secret() {
  local label="$1"
  local value
  read -r -s -p "$label: " value
  echo >&2
  printf '%s' "$value"
}

confirm() {
  local label="$1"
  local default="${2:-y}"
  local value
  local normalized
  read -r -p "$label [$default]: " value
  value="${value:-$default}"
  normalized="$(printf '%s' "$value" | tr '[:upper:]' '[:lower:]')"
  case "$normalized" in
    y|yes|true|1) return 0 ;;
    *) return 1 ;;
  esac
}

run() {
  printf '\n> '
  printf '%q ' "$@"
  printf '\n'
  if [[ "$DRY_RUN" -eq 0 ]]; then
    "$@"
  fi
}

capture() {
  local fallback="${*: -1}"
  local count=$(($# - 1))
  local cmd=("${@:1:$count}")
  if [[ "$DRY_RUN" -eq 1 ]]; then
    printf '%s' "$fallback"
    return 0
  fi
  "${cmd[@]}"
}

env_payload() {
  local IFS='~'
  printf '^~^%s' "$*"
}

require_command gcloud

echo "TTB GCP interactive deployment"
echo "Working tree: $ROOT_DIR"
echo
print_prerequisites
echo

if ! confirm "Manual prerequisites are complete"; then
  echo "Cancelled. Complete the prerequisites first, then rerun this script."
  exit 0
fi

echo

PROJECT_ID="$(prompt "GCP project ID" "${PROJECT_ID:-}")"
REGION="$(prompt "GCP Cloud Run region" "${REGION:-us-central1}")"
REPO="$(prompt "Artifact Registry repo name" "${REPO:-ttb}")"

API_SERVICE="$(prompt "Cloud Run API service name" "${API_SERVICE:-api}")"
OCR_SERVICE="$(prompt "Cloud Run OCR service name" "${OCR_SERVICE:-ocr}")"
JUDGE_SERVICE="$(prompt "Cloud Run Judge service name" "${JUDGE_SERVICE:-judge}")"

DB_INSTANCE="$(prompt "Cloud SQL instance name" "${DB_INSTANCE:-labelverify-db}")"
DB_NAME="$(prompt "Cloud SQL database name" "${DB_NAME:-labelverify}")"
DB_USER="$(prompt "Cloud SQL database user" "${DB_USER:-labelverify}")"

DEFAULT_CONNECTION_NAME="${PROJECT_ID}:${REGION}:${DB_INSTANCE}"
INSTANCE_CONNECTION_NAME="$(prompt "Cloud SQL instance connection name" "${INSTANCE_CONNECTION_NAME:-$DEFAULT_CONNECTION_NAME}")"

echo
echo "Database password is used to build DATABASE_URL for the API Cloud Run service."
echo "If it contains URL-reserved characters, provide DATABASE_URL explicitly below."
DB_PASSWORD="$(prompt_secret "Cloud SQL database password")"
DEFAULT_DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@localhost/${DB_NAME}?host=/cloudsql/${INSTANCE_CONNECTION_NAME}"
DATABASE_URL="$(prompt "DATABASE_URL" "${DATABASE_URL:-$DEFAULT_DATABASE_URL}")"

VERIFY_TIMEOUT_MS="$(prompt "Per-image verify timeout ms" "${VERIFY_TIMEOUT_MS:-7000}")"
JUDGE_TIMEOUT_MS="$(prompt "AI Label Reader timeout ms" "${JUDGE_TIMEOUT_MS:-2500}")"
JUDGE_LOCATION="$(prompt "Vertex/Gemini location for judge service" "${JUDGE_LOCATION:-global}")"
JUDGE_MODEL="$(prompt "Vertex/Gemini model for judge service" "${JUDGE_MODEL:-gemini-2.5-flash}")"
JUDGE_RETRY_ATTEMPTS="$(prompt "Vertex retry attempts for 429s" "${JUDGE_RETRY_ATTEMPTS:-2}")"
JUDGE_RETRY_INITIAL_MS="$(prompt "Vertex initial retry backoff ms" "${JUDGE_RETRY_INITIAL_MS:-250}")"
LLM_MODE="$(prompt "Default AI reader mode: shadow or override" "${LLM_MODE:-shadow}")"
LLM_ENABLED="$(prompt "Enable AI Label Reader by default at API level" "${LLM_ENABLED:-false}")"

if [[ "$LLM_MODE" != "shadow" && "$LLM_MODE" != "override" ]]; then
  echo "LLM_MODE must be shadow or override" >&2
  exit 1
fi

API_IMAGE="${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPO}/api:latest"
OCR_IMAGE="${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPO}/ocr:latest"
JUDGE_IMAGE="${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPO}/judge:latest"

echo
echo "Deployment summary"
echo "  Project: $PROJECT_ID"
echo "  Region: $REGION"
echo "  Vertex location: $JUDGE_LOCATION"
echo "  Images:"
echo "    $API_IMAGE"
echo "    $OCR_IMAGE"
echo "    $JUDGE_IMAGE"
echo "  Cloud SQL: $INSTANCE_CONNECTION_NAME"
echo "  API public: yes"
echo "  OCR/Judge ingress: public"
echo

if ! confirm "Proceed"; then
  echo "Cancelled."
  exit 0
fi

run gcloud config set project "$PROJECT_ID"

if [[ "$SKIP_BUILD" -eq 0 ]]; then
  run gcloud builds submit "$ROOT_DIR/services/ocr" --tag "$OCR_IMAGE"
  run gcloud builds submit "$ROOT_DIR/services/judge" --tag "$JUDGE_IMAGE"
  run gcloud builds submit "$ROOT_DIR/services/api" --tag "$API_IMAGE"
else
  echo "Skipping image builds because --skip-build was provided."
fi

run gcloud run deploy "$OCR_SERVICE" \
  --image "$OCR_IMAGE" \
  --region "$REGION" \
  --port 8001 \
  --memory 4Gi \
  --cpu 4 \
  --ingress all \
  --allow-unauthenticated \
  --set-env-vars "$(env_payload \
    "OCR_MAX_DIM=1400" \
    "OCR_DENOISE=false" \
    "OCR_BOTTOM_PASS=true" \
    "OCR_RIGHT_EDGE_PASS=false" \
    "OCR_INTRA_OP_THREADS=4" \
    "OCR_INTER_OP_THREADS=1")"

JUDGE_ENV="$(env_payload \
  "JUDGE_PROVIDER=google-cloud" \
  "JUDGE_PROJECT=${PROJECT_ID}" \
  "JUDGE_LOCATION=${JUDGE_LOCATION}" \
  "JUDGE_MODEL=${JUDGE_MODEL}" \
  "JUDGE_TIMEOUT_MS=${JUDGE_TIMEOUT_MS}" \
  "JUDGE_RETRY_ATTEMPTS=${JUDGE_RETRY_ATTEMPTS}" \
  "JUDGE_RETRY_INITIAL_MS=${JUDGE_RETRY_INITIAL_MS}")"

run gcloud run deploy "$JUDGE_SERVICE" \
  --image "$JUDGE_IMAGE" \
  --region "$REGION" \
  --port 8002 \
  --memory 1Gi \
  --cpu 1 \
  --ingress all \
  --allow-unauthenticated \
  --set-env-vars "$JUDGE_ENV"

OCR_URL="$(capture gcloud run services describe "$OCR_SERVICE" --region "$REGION" --format='value(status.url)' "https://${OCR_SERVICE}-DRYRUN.a.run.app")"
JUDGE_URL="$(capture gcloud run services describe "$JUDGE_SERVICE" --region "$REGION" --format='value(status.url)' "https://${JUDGE_SERVICE}-DRYRUN.a.run.app")"

JUDGE_SA="$(capture gcloud run services describe "$JUDGE_SERVICE" --region "$REGION" --format='value(spec.template.spec.serviceAccountName)' "${PROJECT_ID}@dryrun.invalid")"
if [[ -n "$JUDGE_SA" ]]; then
  run gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member "serviceAccount:${JUDGE_SA}" \
    --role "roles/aiplatform.user"
fi

API_ENV="$(env_payload \
  "DATABASE_URL=${DATABASE_URL}" \
  "OCR_ADDR=${OCR_URL}" \
  "MIGRATIONS_DIR=/app/migrations" \
  "VERIFY_TIMEOUT_MS=${VERIFY_TIMEOUT_MS}" \
  "LLM_ENABLED=${LLM_ENABLED}" \
  "LLM_MODE=${LLM_MODE}" \
  "LLM_ADDR=${JUDGE_URL}" \
  "LLM_MODEL=${JUDGE_MODEL}" \
  "LLM_TIMEOUT_MS=${JUDGE_TIMEOUT_MS}" \
  "LLM_ALLOWED_FIELDS=brand,class_type" \
  "LLM_DENIED_FIELDS=abv,net_contents,government_warning,name_address" \
  "LLM_MIN_ELIGIBLE_FAILING_FIELDS=1")"

run gcloud run deploy "$API_SERVICE" \
  --image "$API_IMAGE" \
  --region "$REGION" \
  --port 8080 \
  --memory 1Gi \
  --cpu 1 \
  --allow-unauthenticated \
  --add-cloudsql-instances "$INSTANCE_CONNECTION_NAME" \
  --set-env-vars "$API_ENV"

API_URL="$(capture gcloud run services describe "$API_SERVICE" --region "$REGION" --format='value(status.url)' "https://${API_SERVICE}-DRYRUN.a.run.app")"

echo
echo "Deployment complete."
echo "  App URL:   $API_URL"
echo "  OCR URL:   $OCR_URL"
echo "  Judge URL: $JUDGE_URL"
echo
echo "Smoke test:"
echo "  curl -sS \"$API_URL/api/applications\""
