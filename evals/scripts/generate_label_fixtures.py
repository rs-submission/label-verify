from __future__ import annotations

import json
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parents[2]
IMAGES_DIR = ROOT / "evals" / "images"
CASES_DIR = ROOT / "evals" / "cases"
DATA_DIR = ROOT / "evals" / "label_data"

FONT_DIR = Path("/System/Library/Fonts/Supplemental")
SERIF_BOLD = FONT_DIR / "Georgia Bold.ttf"
SERIF = FONT_DIR / "Georgia.ttf"
SANS_BOLD = FONT_DIR / "Arial Bold.ttf"
SANS = FONT_DIR / "Arial.ttf"


LABELS = [
    {
        "id": "generated_rye_whiskey",
        "filename": "generated_rye_whiskey.png",
        "product": "POM RYE WHISKEY",
        "subtitle": "STRAIGHT RYE WHISKEY",
        "production": "DISTILLED FROM RYE MASH",
        "location": "PURCELLVILLE, VA",
        "application": {
            "ID": "generated-rye-whiskey",
            "Brand": "POM CREEK DISTILLING COMPANY, LLC",
            "ClassType": "Rye Whiskey",
            "NetContents": "750 mL",
            "ABV": "45% ALC/VOL (90 Proof)",
            "GovernmentWarning": "GOVERNMENT WARNING",
            "ForeignBlocks": [],
            "DeclaredLanguages": ["en"],
        },
    },
    {
        "id": "generated_blueberry_liqueur",
        "filename": "generated_blueberry_liqueur.png",
        "product": "POM BLUEBERRY LIQUEUR",
        "subtitle": "BLUEBERRY LIQUEUR / CORDIAL",
        "production": "DISTILLED WITH BLUEBERRIES",
        "location": "LOUDOUN COUNTY, VA",
        "application": {
            "ID": "generated-blueberry-liqueur",
            "Brand": "POM CREEK DISTILLING COMPANY, LLC",
            "ClassType": "Blueberry Liqueur/Cordial",
            "NetContents": "375 mL",
            "ABV": "25% ALC/VOL (50 Proof)",
            "GovernmentWarning": "GOVERNMENT WARNING",
            "ForeignBlocks": [],
            "DeclaredLanguages": ["en"],
        },
    },
    {
        "id": "generated_apple_brandy",
        "filename": "generated_apple_brandy.png",
        "product": "POM APPLE BRANDY",
        "subtitle": "APPLE BRANDY",
        "production": "DISTILLED WITH APPLES",
        "location": "LOUDOUN COUNTY, VA",
        "application": {
            "ID": "generated-apple-brandy",
            "Brand": "POM CREEK DISTILLING COMPANY, LLC",
            "ClassType": "Apple Brandy",
            "NetContents": "750 mL",
            "ABV": "40% ALC/VOL (80 Proof)",
            "GovernmentWarning": "GOVERNMENT WARNING",
            "ForeignBlocks": [],
            "DeclaredLanguages": ["en"],
        },
    },
]


def font(path: Path, size: int) -> ImageFont.FreeTypeFont:
    return ImageFont.truetype(str(path), size=size)


def fit_font(draw: ImageDraw.ImageDraw, path: Path, text: str, size: int, max_width: int) -> ImageFont.FreeTypeFont:
    while size > 18:
        font_obj = font(path, size)
        bbox = draw.textbbox((0, 0), text, font=font_obj)
        if bbox[2] - bbox[0] <= max_width:
            return font_obj
        size -= 2
    return font(path, size)


def center(draw: ImageDraw.ImageDraw, xy: tuple[int, int], text: str, font_obj: ImageFont.FreeTypeFont, fill: str) -> None:
    x, y = xy
    bbox = draw.textbbox((0, 0), text, font=font_obj)
    width = bbox[2] - bbox[0]
    draw.text((x - width / 2, y), text, font=font_obj, fill=fill)


def render_label(label: dict) -> Image.Image:
    img = Image.new("RGB", (1400, 1900), "#f7f2e8")
    draw = ImageDraw.Draw(img)

    ink = "#1e1a17"
    accent = "#8f2732"
    gold = "#b88a34"

    draw.rounded_rectangle((90, 90, 1310, 1810), radius=34, outline=ink, width=8)
    draw.rounded_rectangle((130, 130, 1270, 1770), radius=24, outline=gold, width=5)
    draw.line((210, 430, 1190, 430), fill=gold, width=4)
    draw.line((210, 1180, 1190, 1180), fill=gold, width=4)

    center(draw, (700, 190), "POM CREEK", font(SERIF_BOLD, 88), ink)
    center(draw, (700, 295), "DISTILLING COMPANY, LLC", font(SANS_BOLD, 42), ink)
    center(draw, (700, 500), label["product"], fit_font(draw, SERIF_BOLD, label["product"], 76, 1080), accent)
    center(draw, (700, 610), label["subtitle"], fit_font(draw, SANS_BOLD, label["subtitle"], 46, 980), ink)

    center(draw, (700, 820), label["production"], font(SANS_BOLD, 38), ink)
    center(draw, (700, 890), "NATURALLY DISTILLED AND BOTTLED", font(SANS, 34), ink)
    center(draw, (700, 960), label["location"], font(SANS_BOLD, 36), ink)

    center(draw, (700, 1250), label["application"]["Brand"], font(SANS_BOLD, 38), ink)
    center(draw, (700, 1330), label["application"]["NetContents"], font(SANS_BOLD, 44), ink)
    center(draw, (700, 1410), label["application"]["ABV"], font(SANS_BOLD, 44), ink)

    warning = label["application"]["GovernmentWarning"]
    center(draw, (700, 1550), warning, font(SANS_BOLD, 42), ink)
    center(
        draw,
        (700, 1620),
        "ACCORDING TO THE SURGEON GENERAL, WOMEN SHOULD NOT DRINK ALCOHOLIC",
        font(SANS, 25),
        ink,
    )
    center(draw, (700, 1660), "BEVERAGES DURING PREGNANCY.", font(SANS, 25), ink)

    return img


def ocr_regions(label: dict) -> list[dict]:
    app = label["application"]
    return [
        {"text": label["product"], "confidence": 0.99, "bbox": [240.0, 500.0, 880.0, 82.0]},
        {"text": label["subtitle"], "confidence": 0.99, "bbox": [280.0, 610.0, 840.0, 48.0]},
        {"text": label["production"], "confidence": 0.99, "bbox": [320.0, 820.0, 760.0, 42.0]},
        {"text": label["location"], "confidence": 0.99, "bbox": [420.0, 960.0, 560.0, 40.0]},
        {"text": app["Brand"], "confidence": 0.99, "bbox": [250.0, 1250.0, 900.0, 42.0]},
        {"text": app["NetContents"], "confidence": 0.99, "bbox": [620.0, 1330.0, 160.0, 46.0]},
        {"text": app["ABV"], "confidence": 0.99, "bbox": [420.0, 1410.0, 560.0, 46.0]},
        {"text": app["GovernmentWarning"], "confidence": 0.99, "bbox": [430.0, 1550.0, 540.0, 44.0]},
    ]


def eval_case(label: dict) -> dict:
    return {
        "id": label["id"],
        "description": f"Generated complete label fixture for {label['product']}.",
        "image": f"evals/images/{label['filename']}",
        "application": label["application"],
        "ocr": {"regions": ocr_regions(label)},
        "expected": {
            "status": "consistent",
            "fields": {
                "brand": "pass",
                "class_type": "pass",
                "net_contents": "pass",
                "abv": "pass",
                "government_warning": "pass",
            },
        },
    }


def main() -> None:
    IMAGES_DIR.mkdir(parents=True, exist_ok=True)
    CASES_DIR.mkdir(parents=True, exist_ok=True)
    DATA_DIR.mkdir(parents=True, exist_ok=True)

    label_data = []
    for label in LABELS:
        img = render_label(label)
        img.save(IMAGES_DIR / label["filename"])

        case = eval_case(label)
        (CASES_DIR / f"{label['id']}.json").write_text(json.dumps(case, indent=2) + "\n")
        label_data.append(
            {
                "id": label["id"],
                "image": f"evals/images/{label['filename']}",
                "application": label["application"],
            }
        )

    (DATA_DIR / "generated_labels.json").write_text(json.dumps(label_data, indent=2) + "\n")


if __name__ == "__main__":
    main()
