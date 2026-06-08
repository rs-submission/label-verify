from __future__ import annotations

import os
import time
from dataclasses import dataclass
from functools import lru_cache
from pathlib import Path
from typing import Any

import numpy as np
from rapidocr_onnxruntime import RapidOCR


BASE_DIR = Path(__file__).resolve().parent
MODELS_DIR = BASE_DIR / "models"
DET_MODEL = MODELS_DIR / "ch_PP-OCRv4_det_infer.onnx"
CLS_MODEL = MODELS_DIR / "ch_ppocr_mobile_v2.0_cls_infer.onnx"
REC_MODEL = MODELS_DIR / "ch_PP-OCRv4_rec_infer.onnx"


@dataclass(frozen=True)
class OCRRegion:
    text: str
    confidence: float
    bbox: tuple[float, float, float, float]


def required_model_paths() -> tuple[Path, Path, Path]:
    return (DET_MODEL, CLS_MODEL, REC_MODEL)


def missing_model_paths() -> list[Path]:
    return [path for path in required_model_paths() if not path.is_file()]


def recognize(img: np.ndarray, langs: list[str] | None = None) -> tuple[list[OCRRegion], int]:
    _ = langs  # Advisory in the Latin-only prototype.
    started = time.perf_counter()
    raw_result = _engine()(img)
    elapsed_ms = int((time.perf_counter() - started) * 1000)
    return _parse_result(raw_result), elapsed_ms


@lru_cache(maxsize=1)
def _engine() -> RapidOCR:
    missing = missing_model_paths()
    if missing:
        rendered = ", ".join(str(path) for path in missing)
        raise RuntimeError(f"missing OCR model files: {rendered}")

    return RapidOCR(
        det_model_path=str(DET_MODEL),
        cls_model_path=str(CLS_MODEL),
        rec_model_path=str(REC_MODEL),
        print_verbose=False,
        use_cuda=False,
        intra_op_num_threads=_env_int("OCR_INTRA_OP_THREADS", 1),
        inter_op_num_threads=_env_int("OCR_INTER_OP_THREADS", 1),
    )


def _env_int(name: str, default: int) -> int:
    try:
        value = int(os.environ.get(name, str(default)))
    except ValueError:
        return default
    return max(1, value)


def _parse_result(raw_result: Any) -> list[OCRRegion]:
    raw_regions = raw_result[0] if isinstance(raw_result, tuple) else raw_result
    if not raw_regions:
        return []

    parsed: list[OCRRegion] = []
    for raw_region in raw_regions:
        region = _parse_region(raw_region)
        if region is not None:
            parsed.append(region)
    return parsed


def _parse_region(raw_region: Any) -> OCRRegion | None:
    if not isinstance(raw_region, (list, tuple)) or len(raw_region) < 2:
        return None

    box = raw_region[0]
    text: Any
    confidence: Any
    if len(raw_region) >= 3:
        text = raw_region[1]
        confidence = raw_region[2]
    elif isinstance(raw_region[1], (list, tuple)) and len(raw_region[1]) >= 2:
        text = raw_region[1][0]
        confidence = raw_region[1][1]
    else:
        return None

    if not isinstance(text, str):
        return None

    bbox = _bbox_from_polygon(box)
    if bbox is None:
        return None

    return OCRRegion(
        text=text,
        confidence=_clamp_confidence(confidence),
        bbox=bbox,
    )


def _bbox_from_polygon(box: Any) -> tuple[float, float, float, float] | None:
    points = np.asarray(box, dtype=np.float32)
    if points.ndim != 2 or points.shape[0] < 2 or points.shape[1] < 2:
        return None

    xs = points[:, 0]
    ys = points[:, 1]
    x0 = max(0.0, float(xs.min()))
    y0 = max(0.0, float(ys.min()))
    x1 = max(x0, float(xs.max()))
    y1 = max(y0, float(ys.max()))
    return (x0, y0, x1 - x0, y1 - y0)


def _clamp_confidence(value: Any) -> float:
    try:
        confidence = float(value)
    except (TypeError, ValueError):
        return 0.0
    return min(1.0, max(0.0, confidence))
