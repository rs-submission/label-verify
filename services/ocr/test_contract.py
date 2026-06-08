from pathlib import Path

from app import RecognizeResponse, Region


def test_golden_response_contract() -> None:
    golden_path = Path(__file__).parents[2] / "contracts" / "ocr_response.golden.json"
    golden = golden_path.read_text()

    parsed = RecognizeResponse.model_validate_json(golden)
    assert parsed.regions[0].text == "GOVERNMENT WARNING"
    assert parsed.elapsed_ms == 740

    example = RecognizeResponse(
        regions=[Region(text="GOVERNMENT WARNING", confidence=0.97, bbox=(0.1, 0.62, 0.55, 0.08))],
        elapsed_ms=740,
    )
    assert example.model_dump(mode="json") == parsed.model_dump(mode="json")

