from fastapi.testclient import TestClient
import cv2
import numpy as np

from app import (
    _keep_bottom_region,
    _map_rotated_right_edge_region,
    _map_scaled_crop_region,
    _merge_regions,
    _recover_fixed_header_boundaries,
    app,
)
from recognizer import OCRRegion


def test_healthz() -> None:
    client = TestClient(app)
    response = client.get("/healthz")
    assert response.status_code == 200
    assert response.json() == {"status": "ok"}


def test_recognize_reads_image_body() -> None:
    client = TestClient(app)
    response = client.post(
        "/recognize?langs=en,fr",
        content=_encoded_label(),
        headers={"Content-Type": "image/jpeg"},
    )
    assert response.status_code == 200
    data = response.json()
    text = " ".join(region["text"] for region in data["regions"])
    assert "WARNING" in text
    assert data["elapsed_ms"] > 0


def test_recognize_reads_vertical_right_edge_warning() -> None:
    client = TestClient(app)
    response = client.post(
        "/recognize?langs=en",
        content=_encoded_vertical_warning_label(),
        headers={"Content-Type": "image/jpeg"},
    )
    assert response.status_code == 200
    data = response.json()
    text = " ".join(region["text"] for region in data["regions"])
    assert "GOVERNMENT" in text
    assert "WARNING" in text


def test_recognize_reads_bottom_warning_box() -> None:
    client = TestClient(app)
    response = client.post(
        "/recognize?langs=en",
        content=_encoded_bottom_warning_label(),
        headers={"Content-Type": "image/jpeg"},
    )
    assert response.status_code == 200
    data = response.json()
    text = " ".join(region["text"] for region in data["regions"])
    assert "GOVERNMENT" in text
    assert "WARNING" in text


def test_recognize_rejects_malformed_image() -> None:
    client = TestClient(app)
    response = client.post(
        "/recognize?langs=en",
        content=b"not-an-image",
        headers={"Content-Type": "image/jpeg"},
    )
    assert response.status_code == 400


def test_recognize_rejects_unsupported_content_type() -> None:
    client = TestClient(app)
    response = client.post(
        "/recognize?langs=en",
        content=b"hello",
        headers={"Content-Type": "text/plain"},
    )
    assert response.status_code == 400


def test_map_rotated_right_edge_region_to_original_coordinates() -> None:
    region = OCRRegion(text="GOVERNMENT WARNING", confidence=0.9, bbox=(10, 20, 100, 30))

    mapped = _map_rotated_right_edge_region(
        region,
        image_width=1000,
        image_height=1600,
        x_offset=760,
        processed_shape=(240, 1600),
    )

    assert mapped is not None
    assert mapped.text == "GOVERNMENT WARNING"
    assert mapped.bbox == (950.0, 10.0, 30.0, 100.0)


def test_map_scaled_crop_region_to_original_coordinates() -> None:
    region = OCRRegion(text="GOVERNMENT WARNING", confidence=0.9, bbox=(20, 30, 100, 40))

    mapped = _map_scaled_crop_region(
        region,
        x_offset=100,
        y_offset=1200,
        crop_shape=(400, 800, 3),
        processed_shape=(800, 1600),
    )

    assert mapped is not None
    assert mapped.text == "GOVERNMENT WARNING"
    assert mapped.bbox == (110.0, 1215.0, 50.0, 20.0)


def test_merge_regions_deduplicates_text_case_insensitively() -> None:
    primary = [OCRRegion(text="Government Warning", confidence=0.8, bbox=(0, 0, 10, 10))]
    extra = [
        OCRRegion(text="GOVERNMENT  WARNING", confidence=0.9, bbox=(1, 1, 10, 10)),
        OCRRegion(text="POM VODKA", confidence=0.9, bbox=(2, 2, 10, 10)),
    ]

    merged = _merge_regions(primary, extra)

    assert [region.text for region in merged] == ["Government Warning", "POM VODKA"]


def test_recover_fixed_header_boundaries_splits_known_compact_header() -> None:
    regions = [
        OCRRegion(text="GOVERNMENTWARNING", confidence=0.99, bbox=(0, 0, 10, 10)),
        OCRRegion(text="GOVERNMENTWARNING:", confidence=0.99, bbox=(0, 0, 10, 10)),
    ]

    recovered = _recover_fixed_header_boundaries(regions)

    assert [region.text for region in recovered] == [
        "GOVERNMENT WARNING",
        "GOVERNMENT WARNING:",
    ]
    assert recovered[0].confidence == regions[0].confidence
    assert recovered[0].bbox == regions[0].bbox


def test_recover_fixed_header_boundaries_does_not_fix_wrong_header_text() -> None:
    regions = [
        OCRRegion(text="GovernmentWarning", confidence=0.99, bbox=(0, 0, 10, 10)),
        OCRRegion(text="GOVERNMENTWARNNG", confidence=0.99, bbox=(0, 0, 10, 10)),
        OCRRegion(text="PREFIXGOVERNMENTWARNING", confidence=0.99, bbox=(0, 0, 10, 10)),
    ]

    recovered = _recover_fixed_header_boundaries(regions)

    assert [region.text for region in recovered] == [
        "GovernmentWarning",
        "GOVERNMENTWARNNG",
        "PREFIXGOVERNMENTWARNING",
    ]


def test_keep_bottom_region_drops_short_orphan_fragments() -> None:
    assert not _keep_bottom_region(OCRRegion(text="SDURING", confidence=0.99, bbox=(0, 0, 10, 10)))
    assert _keep_bottom_region(OCRRegion(text="GOVERNMENT WARNING:", confidence=0.99, bbox=(0, 0, 10, 10)))
    assert _keep_bottom_region(OCRRegion(text="NET CONTENTS 750 mL", confidence=0.99, bbox=(0, 0, 10, 10)))
    assert _keep_bottom_region(
        OCRRegion(text="WOMEN SHOULD NOT DRINK ALCOHOLIC BEVERAGES", confidence=0.99, bbox=(0, 0, 10, 10))
    )


def _encoded_label() -> bytes:
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
    ok, encoded = cv2.imencode(".jpg", img)
    assert ok
    return encoded.tobytes()


def _encoded_vertical_warning_label() -> bytes:
    img = np.full((900, 700, 3), 255, dtype=np.uint8)
    cv2.putText(
        img,
        "POM VODKA",
        (35, 120),
        cv2.FONT_HERSHEY_SIMPLEX,
        1.7,
        (0, 0, 0),
        4,
        cv2.LINE_AA,
    )

    warning_band = np.full((120, 760, 3), 255, dtype=np.uint8)
    cv2.putText(
        warning_band,
        "GOVERNMENT WARNING",
        (20, 78),
        cv2.FONT_HERSHEY_SIMPLEX,
        1.6,
        (0, 0, 0),
        4,
        cv2.LINE_AA,
    )
    vertical = cv2.rotate(warning_band, cv2.ROTATE_90_CLOCKWISE)
    img[120 : 120 + vertical.shape[0], 570 : 570 + vertical.shape[1]] = vertical

    ok, encoded = cv2.imencode(".jpg", img)
    assert ok
    return encoded.tobytes()


def _encoded_bottom_warning_label() -> bytes:
    img = np.full((1200, 900, 3), 255, dtype=np.uint8)
    cv2.putText(
        img,
        "POM BOURBON",
        (90, 300),
        cv2.FONT_HERSHEY_SIMPLEX,
        2.0,
        (0, 0, 0),
        5,
        cv2.LINE_AA,
    )
    cv2.rectangle(img, (110, 970), (790, 1135), (0, 0, 0), 2)
    cv2.putText(
        img,
        "GOVERNMENT WARNING:",
        (130, 1025),
        cv2.FONT_HERSHEY_SIMPLEX,
        0.9,
        (0, 0, 0),
        2,
        cv2.LINE_AA,
    )
    cv2.putText(
        img,
        "WOMEN SHOULD NOT DRINK ALCOHOLIC BEVERAGES",
        (130, 1075),
        cv2.FONT_HERSHEY_SIMPLEX,
        0.58,
        (0, 0, 0),
        2,
        cv2.LINE_AA,
    )

    ok, encoded = cv2.imencode(".jpg", img)
    assert ok
    return encoded.tobytes()
