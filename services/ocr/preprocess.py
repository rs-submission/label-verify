import cv2
import numpy as np


# 1600 downscaled tall labels (e.g. 2362px) enough to erase fine mandated print
# such as the vertically-set GOVERNMENT WARNING; 2600 keeps such labels at full
# resolution while still bounding very large phone photos. See preprocessing eval.
def preprocess(img: np.ndarray, max_dim: int = 2600, denoise: bool = True) -> np.ndarray:
    if img is None or img.size == 0:
        raise ValueError("empty image")

    gray = _to_gray(img)
    gray = _resize_to_max(gray, max_dim)
    if denoise:
        gray = cv2.fastNlMeansDenoising(gray, None, 10, 7, 21)
    enhanced = _clahe(gray)
    return _deskew(enhanced)


def _to_gray(img: np.ndarray) -> np.ndarray:
    if len(img.shape) == 2:
        return img.astype(np.uint8, copy=False)
    return cv2.cvtColor(img, cv2.COLOR_BGR2GRAY)


def _resize_to_max(img: np.ndarray, max_dim: int) -> np.ndarray:
    h, w = img.shape[:2]
    longest = max(h, w)
    if longest <= max_dim:
        return img
    scale = max_dim / longest
    return cv2.resize(img, (int(w * scale), int(h * scale)), interpolation=cv2.INTER_AREA)


def _clahe(img: np.ndarray) -> np.ndarray:
    clahe = cv2.createCLAHE(clipLimit=2.0, tileGridSize=(8, 8))
    return clahe.apply(img)


def _deskew(img: np.ndarray) -> np.ndarray:
    coords = np.column_stack(np.where(img < 250))
    if len(coords) < 10:
        return img

    angle = cv2.minAreaRect(coords)[-1]
    if angle < -45:
        angle = -(90 + angle)
    else:
        angle = -angle
    if abs(angle) < 0.5 or abs(angle) > 15:
        return img

    h, w = img.shape[:2]
    matrix = cv2.getRotationMatrix2D((w / 2, h / 2), angle, 1.0)
    return cv2.warpAffine(
        img,
        matrix,
        (w, h),
        flags=cv2.INTER_CUBIC,
        borderMode=cv2.BORDER_REPLICATE,
    )
