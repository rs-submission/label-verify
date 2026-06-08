import json
import os
import base64
import asyncio
from typing import Any, Literal, Protocol

import httpx
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field, ValidationError


DecisionValue = Literal["equivalent", "not_equivalent", "uncertain"]
ProviderValue = Literal["ollama", "openai", "google-cloud"]


class AdjudicateRequest(BaseModel):
    field: str
    expected: str
    extracted: str
    deterministic_score: float = Field(ge=0.0, le=1.0)
    ocr_candidates: list[str] = Field(default_factory=list, max_length=50)


class JudgeDecision(BaseModel):
    decision: DecisionValue
    confidence: float = Field(ge=0.0, le=1.0)
    matched_text: str = ""
    reason: str = Field(default="", max_length=300)


class ApplicationEvidence(BaseModel):
    brand: str = ""
    class_type: str = ""
    net_contents: str = ""
    abv: str = ""
    government_warning: str = ""
    name_and_address: str = ""


class LabelReadRequest(BaseModel):
    image_base64: str
    mime_type: str = "image/png"
    application: ApplicationEvidence = Field(default_factory=ApplicationEvidence)


class LabelReadResponse(BaseModel):
    brand: str | None = None
    class_type: str | None = None
    net_contents: str | None = None
    alcohol_contents: str | None = None
    name_and_address: str | None = None
    government_warning: str | None = None


class Settings(BaseModel):
    provider: ProviderValue = "ollama"
    model: str = "gemma4:latest"
    base_url: str = "http://localhost:11434"
    api_key: str = ""
    project: str = ""
    location: str = "us-central1"
    timeout_ms: int = Field(default=4500, gt=0)
    retry_attempts: int = Field(default=2, ge=1, le=5)
    retry_initial_ms: int = Field(default=250, ge=0, le=5000)
    output_strategy: Literal["json_schema"] = "json_schema"


SYSTEM_PROMPT = """Alcohol label field judge. Return only JSON with keys:
decision, confidence, matched_text, reason.
decision must be equivalent, not_equivalent, or uncertain.
Use only supplied evidence. Be conservative; do not invent missing text.
confidence is 0..1. reason is one short sentence."""


class JudgeClient(Protocol):
    async def adjudicate(self, request: AdjudicateRequest) -> JudgeDecision:
        ...

    async def read_label(self, request: LabelReadRequest) -> LabelReadResponse:
        ...


def load_settings() -> Settings:
    provider = os.getenv("JUDGE_PROVIDER", "ollama").strip().lower() or "ollama"
    if provider == "openai":
        provider = "ollama"
    default_model = "gemini-2.5-flash" if provider == "google-cloud" else "gemma4:latest"
    default_location = "global" if provider == "google-cloud" else "us-central1"
    return Settings(
        provider=provider,
        model=os.getenv("JUDGE_MODEL", default_model),
        base_url=normalize_base_url(os.getenv("JUDGE_BASE_URL", "http://localhost:11434")),
        api_key=os.getenv("JUDGE_API_KEY", ""),
        project=os.getenv("JUDGE_PROJECT", ""),
        location=os.getenv("JUDGE_LOCATION", default_location),
        timeout_ms=int(os.getenv("JUDGE_TIMEOUT_MS", "4500")),
        retry_attempts=int(os.getenv("JUDGE_RETRY_ATTEMPTS", "2")),
        retry_initial_ms=int(os.getenv("JUDGE_RETRY_INITIAL_MS", "250")),
    )


def normalize_base_url(value: str) -> str:
    base_url = value.strip().rstrip("/") or "http://localhost:11434"
    if base_url.endswith("/v1"):
        return base_url[:-3]
    return base_url


def build_judge(settings: Settings) -> JudgeClient:
    if settings.provider == "ollama":
        return OllamaJudgeClient(settings)
    if settings.provider == "google-cloud":
        return VertexJudgeClient(settings)
    raise ValueError(f"unsupported judge provider: {settings.provider}")


class OllamaJudgeClient:
    def __init__(self, settings: Settings):
        self.settings = settings

    async def adjudicate(self, request: AdjudicateRequest) -> JudgeDecision:
        payload = {
            "model": self.settings.model,
            "prompt": user_prompt(request),
            "system": SYSTEM_PROMPT,
            "stream": False,
            "format": "json",
            "keep_alive": "10m",
            "options": {
                "temperature": 0,
                "num_predict": 120,
            },
        }
        timeout = httpx.Timeout(self.settings.timeout_ms / 1000)
        async with httpx.AsyncClient(timeout=timeout) as client:
            try:
                response = await client.post(f"{self.settings.base_url}/api/generate", json=payload)
            except httpx.TimeoutException as exc:
                raise TimeoutError(f"Ollama adjudication timed out after {self.settings.timeout_ms}ms") from exc
            response.raise_for_status()
        body = response.json()
        return parse_decision(body.get("response", ""))

    async def read_label(self, request: LabelReadRequest) -> LabelReadResponse:
        payload = {
            "model": self.settings.model,
            "prompt": label_reader_prompt(request.application),
            "images": [request.image_base64],
            "stream": False,
            "format": "json",
            "keep_alive": "10m",
            "options": {
                "temperature": 0,
                "num_predict": 220,
            },
        }
        timeout = httpx.Timeout(self.settings.timeout_ms / 1000)
        async with httpx.AsyncClient(timeout=timeout) as client:
            try:
                response = await client.post(f"{self.settings.base_url}/api/generate", json=payload)
            except httpx.TimeoutException as exc:
                raise TimeoutError(f"Ollama label read timed out after {self.settings.timeout_ms}ms") from exc
            response.raise_for_status()
        body = response.json()
        return parse_label_read(body.get("response", ""))


class VertexJudgeClient:
    def __init__(self, settings: Settings):
        self.settings = settings

    async def adjudicate(self, request: AdjudicateRequest) -> JudgeDecision:
        from google import genai
        from google.genai import types

        client = genai.Client(
            vertexai=True,
            project=self.settings.project or None,
            location=self.settings.location,
            http_options=types.HttpOptions(timeout=self.settings.timeout_ms),
        )
        response = await self._generate_content_with_retry(
            client,
            types,
            model=self.settings.model,
            contents=user_prompt(request),
            config=types.GenerateContentConfig(
                system_instruction=SYSTEM_PROMPT,
                temperature=0,
                max_output_tokens=160,
                response_mime_type="application/json",
                response_schema=JudgeDecision,
            ),
        )
        parsed = getattr(response, "parsed", None)
        if isinstance(parsed, JudgeDecision):
            return parsed
        return parse_decision(getattr(response, "text", ""))

    async def read_label(self, request: LabelReadRequest) -> LabelReadResponse:
        from google import genai
        from google.genai import types

        image_bytes = base64.b64decode(request.image_base64)
        client = genai.Client(
            vertexai=True,
            project=self.settings.project or None,
            location=self.settings.location,
            http_options=types.HttpOptions(timeout=self.settings.timeout_ms),
        )
        response = await self._generate_content_with_retry(
            client,
            types,
            model=self.settings.model,
            contents=[
                label_reader_prompt(request.application),
                types.Part.from_bytes(data=image_bytes, mime_type=request.mime_type),
            ],
            config=types.GenerateContentConfig(
                temperature=0,
                max_output_tokens=512,
                response_mime_type="application/json",
                response_schema=LabelReadResponse,
            ),
        )
        parsed = getattr(response, "parsed", None)
        if isinstance(parsed, LabelReadResponse):
            return parsed
        return parse_label_read(getattr(response, "text", ""))

    async def _generate_content_with_retry(self, client: Any, types: Any, **kwargs: Any) -> Any:
        delay = self.settings.retry_initial_ms / 1000
        for attempt in range(1, self.settings.retry_attempts + 1):
            try:
                return await client.aio.models.generate_content(**kwargs)
            except Exception as exc:
                if attempt >= self.settings.retry_attempts or not is_resource_exhausted(str(exc)):
                    raise
                await asyncio.sleep(delay)
                delay = max(delay * 2, 0.001)


settings = load_settings()
judge = build_judge(settings)
app = FastAPI(title="Label Verification Judge")


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {
        "status": "ok",
        "provider": settings.provider,
        "model": settings.model,
        "output_strategy": settings.output_strategy,
    }


@app.post("/adjudicate", response_model=JudgeDecision)
async def adjudicate(request: AdjudicateRequest) -> JudgeDecision:
    try:
        return await judge.adjudicate(request)
    except ValidationError as exc:
        raise HTTPException(status_code=502, detail=f"judge returned invalid JSON: {exc}") from exc
    except Exception as exc:
        raise judge_unavailable("judge model unavailable", exc) from exc


@app.post("/read-label", response_model=LabelReadResponse)
async def read_label(request: LabelReadRequest) -> LabelReadResponse:
    try:
        return await judge.read_label(request)
    except ValidationError as exc:
        raise HTTPException(
            status_code=502,
            detail="judge returned invalid label JSON; deterministic verification was used.",
        ) from exc
    except Exception as exc:
        raise judge_unavailable("judge label reader unavailable", exc) from exc


def judge_unavailable(prefix: str, exc: Exception) -> HTTPException:
    message = str(exc)
    if is_resource_exhausted(message):
        return HTTPException(
            status_code=429,
            detail=f"{prefix}: Vertex AI quota exhausted; try again later or disable AI Label Reader.",
        )
    return HTTPException(status_code=502, detail=f"{prefix}: {message}")


def is_resource_exhausted(message: str) -> bool:
    normalized = message.casefold()
    return "resource_exhausted" in normalized or "resource exhausted" in normalized or "429" in normalized


def user_prompt(request: AdjudicateRequest) -> str:
    evidence = request.model_dump()
    evidence["ocr_candidates"] = [candidate for candidate in request.ocr_candidates if candidate.strip()]
    return json.dumps(evidence, ensure_ascii=True, separators=(",", ":"))


def label_reader_prompt(application: ApplicationEvidence) -> str:
    evidence = application.model_dump()
    return (
        'Read this alcohol label image and return ONLY valid JSON with these exact keys: '
        '"brand", "class_type", "net_contents", "alcohol_contents", '
        '"name_and_address", "government_warning". '
        'For "government_warning", transcribe the GOVERNMENT WARNING header word-for-word exactly as it appears. '
        "Use null for any field not visible. Do not include text outside the JSON. "
        f"Application reference values: {json.dumps(evidence, ensure_ascii=True, separators=(',', ':'))}"
    )


def parse_decision(value: Any) -> JudgeDecision:
    if isinstance(value, JudgeDecision):
        return value
    if isinstance(value, dict):
        return JudgeDecision.model_validate(value)
    text = str(value).strip()
    if text.startswith("```"):
        text = text.strip("`")
        if text.lower().startswith("json"):
            text = text[4:].strip()
    return JudgeDecision.model_validate_json(text)


def parse_label_read(value: Any) -> LabelReadResponse:
    if isinstance(value, LabelReadResponse):
        return value
    if isinstance(value, dict):
        return LabelReadResponse.model_validate(value)
    text = str(value).strip()
    if text.startswith("```"):
        text = text.strip("`")
        if text.lower().startswith("json"):
            text = text[4:].strip()
    return LabelReadResponse.model_validate_json(text)
