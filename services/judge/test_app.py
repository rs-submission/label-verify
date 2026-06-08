from fastapi.testclient import TestClient
from pydantic import ValidationError

from app import (
    AdjudicateRequest,
    ApplicationEvidence,
    JudgeDecision,
    LabelReadResponse,
    app,
    is_resource_exhausted,
    label_reader_prompt,
    load_settings,
    normalize_base_url,
    parse_decision,
    parse_label_read,
    user_prompt,
)


class FakeJudge:
    def __init__(
        self,
        output: JudgeDecision,
        label_output: LabelReadResponse | None = None,
        label_error: Exception | None = None,
    ):
        self.output = output
        self.label_output = label_output or LabelReadResponse(brand="POM BOURBON")
        self.label_error = label_error
        self.requests: list[AdjudicateRequest] = []

    async def adjudicate(self, request: AdjudicateRequest) -> JudgeDecision:
        self.requests.append(request)
        return self.output

    async def read_label(self, request):
        if self.label_error is not None:
            raise self.label_error
        return self.label_output


def test_healthz() -> None:
    client = TestClient(app)

    response = client.get("/healthz")

    assert response.status_code == 200
    assert response.json()["status"] == "ok"
    assert response.json()["provider"] == "ollama"
    assert response.json()["output_strategy"] == "json_schema"


def test_load_settings_defaults_to_local_ollama(monkeypatch) -> None:
    monkeypatch.delenv("JUDGE_PROVIDER", raising=False)
    monkeypatch.delenv("JUDGE_MODEL", raising=False)

    settings = load_settings()

    assert settings.provider == "ollama"
    assert settings.model == "gemma4:latest"
    assert settings.base_url == "http://localhost:11434"


def test_load_settings_treats_openai_as_legacy_ollama_alias(monkeypatch) -> None:
    monkeypatch.setenv("JUDGE_PROVIDER", "openai")
    monkeypatch.setenv("JUDGE_BASE_URL", "http://localhost:11434/v1")

    settings = load_settings()

    assert settings.provider == "ollama"
    assert settings.base_url == "http://localhost:11434"


def test_load_settings_supports_google_cloud(monkeypatch) -> None:
    monkeypatch.setenv("JUDGE_PROVIDER", "google-cloud")
    monkeypatch.delenv("JUDGE_MODEL", raising=False)
    monkeypatch.delenv("JUDGE_LOCATION", raising=False)
    monkeypatch.setenv("JUDGE_PROJECT", "demo-project")
    monkeypatch.setenv("JUDGE_RETRY_ATTEMPTS", "3")
    monkeypatch.setenv("JUDGE_RETRY_INITIAL_MS", "125")

    settings = load_settings()

    assert settings.provider == "google-cloud"
    assert settings.model == "gemini-2.5-flash"
    assert settings.project == "demo-project"
    assert settings.location == "global"
    assert settings.retry_attempts == 3
    assert settings.retry_initial_ms == 125


def test_load_settings_allows_explicit_google_cloud_location(monkeypatch) -> None:
    monkeypatch.setenv("JUDGE_PROVIDER", "google-cloud")
    monkeypatch.setenv("JUDGE_LOCATION", "us-central1")

    settings = load_settings()

    assert settings.location == "us-central1"


def test_adjudicate_returns_typed_decision(monkeypatch) -> None:
    fake = FakeJudge(
        JudgeDecision(
            decision="equivalent",
            confidence=0.88,
            matched_text="POM CHERRY CORDIAL",
            reason="Cordial is equivalent to liqueur/cordial class text.",
        )
    )
    monkeypatch.setattr("app.judge", fake)
    client = TestClient(app)

    response = client.post(
        "/adjudicate",
        json={
            "field": "class_type",
            "expected": "Cherry Liqueur/Cordial",
            "extracted": "POM CHERRY CORDIAL",
            "deterministic_score": 0.67,
            "ocr_candidates": ["POM CHERRY CORDIAL"],
        },
    )

    assert response.status_code == 200
    assert response.json()["decision"] == "equivalent"
    assert response.json()["confidence"] == 0.88
    assert fake.requests[0].expected == "Cherry Liqueur/Cordial"


def test_invalid_decision_payload_fails_validation() -> None:
    client = TestClient(app)

    response = client.post(
        "/adjudicate",
        json={
            "field": "class_type",
            "expected": "Vodka",
            "extracted": "POM VODKA",
            "deterministic_score": 1.5,
        },
    )

    assert response.status_code == 422


def test_read_label_returns_typed_fields(monkeypatch) -> None:
    fake = FakeJudge(
        JudgeDecision(decision="uncertain", confidence=0.1),
        LabelReadResponse(
            brand="POM BOURBON",
            class_type="Bourbon",
            net_contents="750 mL",
            alcohol_contents="45% ALC/VOL (90 Proof)",
            name_and_address="POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
            government_warning="GOVERNMENT WARNING",
        ),
    )
    monkeypatch.setattr("app.judge", fake)
    client = TestClient(app)

    response = client.post(
        "/read-label",
        json={
            "image_base64": "aW1hZ2U=",
            "mime_type": "image/png",
            "application": {"brand": "POM BOURBON", "class_type": "Bourbon"},
        },
    )

    assert response.status_code == 200
    assert response.json()["brand"] == "POM BOURBON"
    assert response.json()["government_warning"] == "GOVERNMENT WARNING"


def test_read_label_reports_vertex_quota_as_rate_limit(monkeypatch) -> None:
    fake = FakeJudge(
        JudgeDecision(decision="uncertain", confidence=0.1),
        label_error=RuntimeError("429 RESOURCE_EXHAUSTED. Resource exhausted."),
    )
    monkeypatch.setattr("app.judge", fake)
    client = TestClient(app)

    response = client.post(
        "/read-label",
        json={"image_base64": "aW1hZ2U=", "mime_type": "image/png"},
    )

    assert response.status_code == 429
    assert "Vertex AI quota exhausted" in response.json()["detail"]


def test_read_label_hides_raw_validation_error(monkeypatch) -> None:
    fake = FakeJudge(
        JudgeDecision(decision="uncertain", confidence=0.1),
        label_error=ValidationError.from_exception_data(
            "LabelReadResponse",
            [
                {
                    "type": "json_invalid",
                    "loc": (),
                    "input": '{"brand":"POM BOURBON",',
                    "ctx": {"error": "EOF while parsing a value"},
                }
            ],
        ),
    )
    monkeypatch.setattr("app.judge", fake)
    client = TestClient(app)

    response = client.post(
        "/read-label",
        json={"image_base64": "aW1hZ2U=", "mime_type": "image/png"},
    )

    assert response.status_code == 502
    assert response.json()["detail"] == "judge returned invalid label JSON; deterministic verification was used."


def test_resource_exhausted_detection() -> None:
    assert is_resource_exhausted("429 RESOURCE_EXHAUSTED. Resource exhausted.")
    assert is_resource_exhausted("resource exhausted, please try again later")
    assert not is_resource_exhausted("permission denied")


def test_user_prompt_strips_blank_candidates() -> None:
    prompt = user_prompt(
        request=AdjudicateRequest(
            field="brand",
            expected="POM BOURBON",
            extracted="POM B0URBON",
            deterministic_score=0.86,
            ocr_candidates=["", "POM B0URBON"],
        )
    )

    assert '""' not in prompt
    assert "POM B0URBON" in prompt


def test_parse_decision_validates_model_json() -> None:
    decision = parse_decision(
        '{"decision":"uncertain","confidence":0.4,"matched_text":"","reason":"not enough evidence"}'
    )

    assert decision.decision == "uncertain"
    assert decision.confidence == 0.4


def test_parse_label_read_validates_model_json() -> None:
    read = parse_label_read(
        '{"brand":"POM BOURBON","class_type":"Bourbon","net_contents":null,'
        '"alcohol_contents":"45% ALC/VOL (90 Proof)","name_and_address":null,'
        '"government_warning":"GOVERNMENT WARNING"}'
    )

    assert read.class_type == "Bourbon"
    assert read.net_contents is None


def test_label_reader_prompt_names_required_keys() -> None:
    prompt = label_reader_prompt(application=ApplicationEvidence())

    assert '"alcohol_contents"' in prompt
    assert "Use null" in prompt


def test_normalize_base_url_removes_openai_suffix() -> None:
    assert normalize_base_url("http://host.docker.internal:11434/v1") == "http://host.docker.internal:11434"
