import cv2
import numpy as np

from preprocess import preprocess


def test_preprocess_returns_grayscale_uint8_with_bounded_size() -> None:
    img = np.full((2400, 1800, 3), 210, dtype=np.uint8)
    cv2.putText(img, "GOVERNMENT WARNING", (100, 900), cv2.FONT_HERSHEY_SIMPLEX, 3, (40, 40, 40), 5)

    out = preprocess(img, max_dim=800)

    assert len(out.shape) == 2
    assert out.dtype == np.uint8
    assert max(out.shape) <= 800


def test_preprocess_rejects_empty_image() -> None:
    try:
        preprocess(np.array([], dtype=np.uint8))
    except ValueError as exc:
        assert "empty image" in str(exc)
    else:
        raise AssertionError("expected ValueError")

