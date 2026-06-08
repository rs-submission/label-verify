from typing import Annotated

import cv2
import numpy as np
import os
import re
from fastapi import FastAPI, HTTPException, Query, Request
from pydantic import BaseModel, Field
from starlette.concurrency import run_in_threadpool

from preprocess import preprocess
from recognizer import OCRRegion, recognize


class Region(BaseModel):
    text: str
    confidence: float = Field(ge=0.0, le=1.0)
    bbox: tuple[float, float, float, float]


class RecognizeResponse(BaseModel):
    regions: list[Region]
    elapsed_ms: int


app = FastAPI(title="Label Verification OCR")

FIXED_HEADER_BOUNDARIES = {
    re.compile(r"(?<![A-Za-z0-9])GOVERNMENTWARNING(?![A-Za-z0-9])"): "GOVERNMENT WARNING",
}


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/recognize", response_model=RecognizeResponse)
async def recognize_route(
    request: Request,
    langs: Annotated[str, Query(pattern=r"^[a-z]{2}(,[a-z]{2})*$")] = "en",
) -> RecognizeResponse:
    if request.headers.get("content-type", "").split(";")[0] not in {
        "image/jpeg",
        "image/png",
        "image/webp",
    }:
        raise HTTPException(status_code=400, detail="unsupported content-type")

    image = _decode_image(await request.body())
    if image is None:
        raise HTTPException(status_code=400, detail="malformed image")

    regions, elapsed_ms = await run_in_threadpool(_recognize_image, image, langs)
    return RecognizeResponse(regions=regions, elapsed_ms=elapsed_ms)


def _decode_image(body: bytes) -> np.ndarray | None:
    if not body:
        return None

    encoded = np.frombuffer(body, dtype=np.uint8)
    return cv2.imdecode(encoded, cv2.IMREAD_COLOR)


def _recognize_image(image: np.ndarray, langs: str) -> tuple[list[Region], int]:
    requested_langs = langs.split(",")
    processed = _preprocess(image)
    regions, elapsed_ms = recognize(processed, requested_langs)
    bottom_regions: list[OCRRegion] = []
    bottom_elapsed_ms = 0
    if _env_bool("OCR_BOTTOM_PASS", True):
        bottom_regions, bottom_elapsed_ms = _recognize_bottom_enlarged(image, requested_langs)
    rotated_regions: list[OCRRegion] = []
    rotated_elapsed_ms = 0
    if _env_bool("OCR_RIGHT_EDGE_PASS", True):
        rotated_regions, rotated_elapsed_ms = _recognize_right_edge_rotated(image, requested_langs)
    regions = _recover_fixed_header_boundaries(regions)
    bottom_regions = _recover_fixed_header_boundaries(bottom_regions)
    rotated_regions = _recover_fixed_header_boundaries(rotated_regions)
    regions = _merge_regions(_merge_regions(regions, bottom_regions), rotated_regions)
    elapsed_ms += bottom_elapsed_ms + rotated_elapsed_ms
    return [
        Region(text=region.text, confidence=region.confidence, bbox=region.bbox)
        for region in regions
    ], elapsed_ms


def _preprocess(image: np.ndarray) -> np.ndarray:
    return preprocess(
        image,
        max_dim=_env_int("OCR_MAX_DIM", 2600),
        denoise=_env_bool("OCR_DENOISE", True),
    )


def _recognize_bottom_enlarged(image: np.ndarray, langs: list[str]) -> tuple[list[OCRRegion], int]:
    height, width = image.shape[:2]
    if height <= 0 or width <= 0:
        return [], 0

    x_offset = int(width * 0.10)
    y_offset = int(height * 0.80)
    x1 = int(width * 0.90)
    y1 = int(height * 0.98)
    crop = image[y_offset:y1, x_offset:x1]
    if crop.size == 0:
        return [], 0

    enlarged = cv2.resize(crop, None, fx=2.0, fy=2.0, interpolation=cv2.INTER_CUBIC)
    processed = _preprocess(enlarged)
    regions, elapsed_ms = recognize(processed, langs)
    mapped = [
        _map_scaled_crop_region(
            region,
            x_offset=x_offset,
            y_offset=y_offset,
            crop_shape=crop.shape,
            processed_shape=processed.shape,
        )
        for region in regions
    ]
    return [region for region in mapped if region is not None and _keep_bottom_region(region)], elapsed_ms


def _recognize_right_edge_rotated(image: np.ndarray, langs: list[str]) -> tuple[list[OCRRegion], int]:
    height, width = image.shape[:2]
    if height <= 0 or width <= 0:
        return [], 0

    x_offset = int(width * 0.82)
    crop = image[:, x_offset:]
    if crop.size == 0:
        return [], 0

    rotated = cv2.rotate(crop, cv2.ROTATE_90_CLOCKWISE)
    processed = _preprocess(rotated)
    regions, elapsed_ms = recognize(processed, langs)
    mapped = [_map_rotated_right_edge_region(region, width, height, x_offset, processed.shape) for region in regions]
    return [region for region in mapped if region is not None], elapsed_ms


def _map_scaled_crop_region(
    region: OCRRegion,
    x_offset: int,
    y_offset: int,
    crop_shape: tuple[int, ...],
    processed_shape: tuple[int, ...],
) -> OCRRegion | None:
    crop_h, crop_w = crop_shape[:2]
    processed_h, processed_w = processed_shape[:2]
    if crop_w <= 0 or crop_h <= 0 or processed_w <= 0 or processed_h <= 0:
        return None

    scale_x = crop_w / processed_w
    scale_y = crop_h / processed_h
    rx, ry, rw, rh = region.bbox
    w = rw * scale_x
    h = rh * scale_y
    if w <= 0 or h <= 0:
        return None
    return OCRRegion(
        text=region.text,
        confidence=region.confidence,
        bbox=(x_offset + rx * scale_x, y_offset + ry * scale_y, w, h),
    )


def _recover_fixed_header_boundaries(regions: list[OCRRegion]) -> list[OCRRegion]:
    return [_recover_fixed_header_boundary(region) for region in regions]


def _recover_fixed_header_boundary(region: OCRRegion) -> OCRRegion:
    text = region.text
    for pattern, replacement in FIXED_HEADER_BOUNDARIES.items():
        text = pattern.sub(replacement, text)
    if text == region.text:
        return region
    return OCRRegion(text=text, confidence=region.confidence, bbox=region.bbox)


def _keep_bottom_region(region: OCRRegion) -> bool:
    text = _region_key(region)
    compact = "".join(text.split())
    if len(compact) >= 16:
        return True
    return any(
        term in text
        for term in (
            "government",
            "warning",
            "surgeon",
            "pregnancy",
            "alc",
            "vol",
            "proof",
            "content",
            "ml",
        )
    )


def _map_rotated_right_edge_region(
    region: OCRRegion,
    image_width: int,
    image_height: int,
    x_offset: int,
    processed_shape: tuple[int, ...],
) -> OCRRegion | None:
    rotated_h, rotated_w = processed_shape[:2]
    crop_w = image_width - x_offset
    if rotated_w <= 0 or rotated_h <= 0 or crop_w <= 0:
        return None

    scale_x = image_height / rotated_w
    scale_y = crop_w / rotated_h
    rx, ry, rw, rh = region.bbox
    x0 = x_offset + (rotated_h - (ry + rh)) * scale_y
    y0 = rx * scale_x
    w = rh * scale_y
    h = rw * scale_x
    if w <= 0 or h <= 0:
        return None
    return OCRRegion(
        text=region.text,
        confidence=region.confidence,
        bbox=(max(0.0, x0), max(0.0, y0), w, h),
    )


def _merge_regions(primary: list[OCRRegion], extra: list[OCRRegion]) -> list[OCRRegion]:
    merged = list(primary)
    seen = {_region_key(region) for region in merged}
    for region in extra:
        key = _region_key(region)
        if key in seen:
            continue
        seen.add(key)
        merged.append(region)
    return merged


def _region_key(region: OCRRegion) -> str:
    return " ".join(region.text.casefold().split())


def _env_bool(name: str, default: bool) -> bool:
    value = os.environ.get(name)
    if value is None:
        return default
    return value.strip().casefold() in {"1", "true", "yes", "on"}


def _env_int(name: str, default: int) -> int:
    try:
        value = int(os.environ.get(name, str(default)))
    except ValueError:
        return default
    return max(1, value)
