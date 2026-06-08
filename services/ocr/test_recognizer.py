import cv2
import numpy as np

from recognizer import missing_model_paths, recognize, required_model_paths


def test_required_model_files_exist() -> None:
    assert len(required_model_paths()) == 3
    assert missing_model_paths() == []


def test_recognize_reads_synthetic_latin_text() -> None:
    img = np.full((240, 900, 3), 255, dtype=np.uint8)
    cv2.putText(
        img,
        "GOVERNMENT WARNING",
        (35, 105),
        cv2.FONT_HERSHEY_SIMPLEX,
        1.7,
        (0, 0, 0),
        4,
        cv2.LINE_AA,
    )
    cv2.putText(
        img,
        "CONTAINS SULFITES",
        (35, 180),
        cv2.FONT_HERSHEY_SIMPLEX,
        1.5,
        (0, 0, 0),
        3,
        cv2.LINE_AA,
    )

    regions, elapsed_ms = recognize(img, ["en", "fr"])

    text = " ".join(region.text for region in regions)
    assert "WARNING" in text
    assert "SULFITES" in text
    assert max(region.confidence for region in regions) > 0.9
    assert elapsed_ms > 0
