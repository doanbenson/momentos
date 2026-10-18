"""Score montage candidate Assets before director placement."""

from __future__ import annotations

import json
import logging
import re
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

from pydantic import BaseModel, Field

from momentos_ml.config import Settings
from momentos_ml.features.extract import AssetFeature
from momentos_ml.media.download import materialize_asset

logger = logging.getLogger(__name__)

_VISION_WORKERS = 3
_VISION_RETRIES = 2


class VisionScore(BaseModel):
    asset_id: str
    score: float = Field(ge=0.0)
    shot_category: str | None = None
    semantic_tags: list[str] = Field(default_factory=list)


def score_montage_candidates(
    features: list[AssetFeature],
    *,
    reflection_asset_id: str | None = None,
    settings: Settings | None = None,
) -> list[VisionScore]:
    """
    Rank montage candidates (excludes Reflection).

    VISION_SCORING=metadata (default) → deterministic scores (fast demos).
    VISION_SCORING=gemini → multimodal Gemini when API key + URLs exist.
    CUT_PLANNER=stub always uses metadata.
    """
    candidates = [
        f
        for f in features
        if not (reflection_asset_id and f.asset_id == reflection_asset_id)
        and f.type != "voice"
    ]
    if not candidates:
        return []

    if settings is None or settings.cut_planner == "stub":
        return _score_metadata(candidates)

    mode = (settings.vision_scoring or "metadata").strip().lower()
    if mode != "gemini" or not settings.gemini_api_key:
        return _score_metadata(candidates)

    try:
        return _score_gemini(candidates, settings=settings)
    except Exception as e:
        logger.warning("Gemini vision scoring failed; metadata fallback: %s", e)
        return _score_metadata(candidates)


def _score_metadata(features: list[AssetFeature]) -> list[VisionScore]:
    scores: list[VisionScore] = []
    for feat in features:
        base = 0.5
        if feat.type == "video":
            base = 0.8
        elif feat.type == "photo":
            base = 0.7
        order_boost = max(0.0, 0.2 - 0.01 * feat.intake_order)
        scores.append(VisionScore(asset_id=feat.asset_id, score=base + order_boost))
    scores.sort(key=lambda s: (-s.score, s.asset_id))
    return scores


def _mime_for_path(path: Path, asset_type: str) -> str:
    ext = path.suffix.lower()
    by_ext = {
        ".jpg": "image/jpeg",
        ".jpeg": "image/jpeg",
        ".png": "image/png",
        ".heic": "image/heic",
        ".heif": "image/heif",
        ".mov": "video/quicktime",
        ".mp4": "video/mp4",
        ".m4a": "audio/mp4",
        ".webm": "video/webm",
    }
    if ext in by_ext:
        return by_ext[ext]
    return "image/jpeg" if asset_type == "photo" else "video/mp4"


def _score_gemini(
    features: list[AssetFeature],
    *,
    settings: Settings,
) -> list[VisionScore]:
    from google import genai
    from google.genai import types

    client = genai.Client(api_key=settings.gemini_api_key)
    out: list[VisionScore] = []
    temps: list[Path] = []

    def one(feat: AssetFeature) -> VisionScore:
        local = materialize_asset(storage_url=feat.storage_url, asset_type=feat.type)
        if local is None:
            return _score_metadata([feat])[0]
        temps.append(local)
        mime = _mime_for_path(local, feat.type)
        prompt = (
            "Score this clip for a bright, high-energy nostalgia montage "
            "(1.0=perfect: smiles, motion, scenic energy; 0.0=unusable). "
            'Reply JSON only: {"score":0.0-1.0,"shot_category":'
            '"action_movement"|"social_joy"|"scenic_wide"|"hero_outro",'
            '"semantic_tags":["tag",...]}'
        )
        data = local.read_bytes()
        last_err: Exception | None = None
        for attempt in range(_VISION_RETRIES + 1):
            try:
                response = client.models.generate_content(
                    model=settings.gemini_model,
                    contents=[
                        types.Content(
                            role="user",
                            parts=[
                                types.Part.from_bytes(data=data, mime_type=mime),
                                types.Part.from_text(text=prompt),
                            ],
                        )
                    ],
                    config=types.GenerateContentConfig(
                        response_mime_type="application/json",
                        temperature=0.2,
                    ),
                )
                text = getattr(response, "text", None) or "{}"
                return _parse_score_json(text, feat.asset_id)
            except Exception as e:
                last_err = e
                msg = str(e)
                retryable = "503" in msg or "UNAVAILABLE" in msg or "429" in msg
                if retryable and attempt < _VISION_RETRIES:
                    time.sleep(0.8 * (attempt + 1))
                    continue
                break
        logger.warning("vision score failed for %s: %s", feat.asset_id, last_err)
        return _score_metadata([feat])[0]

    try:
        with ThreadPoolExecutor(max_workers=_VISION_WORKERS) as pool:
            futures = {pool.submit(one, f): f for f in features}
            for fut in as_completed(futures):
                out.append(fut.result())
    finally:
        for p in temps:
            p.unlink(missing_ok=True)

    out.sort(key=lambda s: (-s.score, s.asset_id))
    return out


def _parse_score_json(raw: str, asset_id: str) -> VisionScore:
    m = re.search(r"\{.*\}", raw, flags=re.DOTALL)
    data = json.loads(m.group(0) if m else raw)
    score = float(data.get("score", 0.5))
    score = max(0.0, min(1.0, score))
    tags = data.get("semantic_tags") or data.get("semanticTags") or []
    if not isinstance(tags, list):
        tags = []
    return VisionScore(
        asset_id=asset_id,
        score=score,
        shot_category=data.get("shot_category") or data.get("shotCategory"),
        semantic_tags=[str(t) for t in tags],
    )
