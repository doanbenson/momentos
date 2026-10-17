"""Single-shot Gemini Cut planner → PlannerPlacement[] (ADR 0002)."""

from __future__ import annotations

import json
import logging
import re
from typing import Any

from momentos_ml.agents.prompts import REPAIR_SUFFIX, SYSTEM_PROMPT, build_user_prompt
from momentos_ml.beat.grid import BeatGrid
from momentos_ml.beat.snap import PlannerPlacement
from momentos_ml.config import Settings
from momentos_ml.features.extract import AssetFeature

logger = logging.getLogger(__name__)


class GeminiPlannerError(RuntimeError):
    pass


def plan_placements(
    features: list[AssetFeature],
    grid: BeatGrid,
    settings: Settings,
    *,
    prior_hint: str | None = None,
    repair_note: str | None = None,
) -> list[PlannerPlacement]:
    if not settings.gemini_api_key:
        raise GeminiPlannerError("GEMINI_API_KEY is required when CUT_PLANNER=gemini")

    # Downsample beat_ms for prompt size (first/last + every Nth).
    sample = _downsample_beats(grid.beat_ms, max_points=32)
    user = build_user_prompt(
        features=features,
        beat_count=len(grid.beat_ms),
        beat_ms_sample=sample,
        song_pick_id=grid.song_pick_id,
        prior_hint=prior_hint,
    )
    if repair_note:
        user = user + REPAIR_SUFFIX + f"\nDetail: {repair_note}"

    raw = _call_gemini(settings, system=SYSTEM_PROMPT, user=user)
    return parse_placements(raw, allowed_ids={f.asset_id for f in features})


def parse_placements(raw: str, *, allowed_ids: set[str]) -> list[PlannerPlacement]:
    data = _extract_json(raw)
    items = data.get("placements") if isinstance(data, dict) else data
    if not isinstance(items, list):
        raise GeminiPlannerError("JSON must contain a placements array")

    out: list[PlannerPlacement] = []
    for item in items:
        if not isinstance(item, dict):
            continue
        aid = item.get("asset_id") or item.get("assetId")
        if aid not in allowed_ids:
            logger.warning("dropping unknown asset_id from planner: %s", aid)
            continue
        out.append(
            PlannerPlacement(
                asset_id=str(aid),
                start_beat=int(item["start_beat"] if "start_beat" in item else item["startBeat"]),
                end_beat=int(item["end_beat"] if "end_beat" in item else item["endBeat"]),
            )
        )
    if not out:
        raise GeminiPlannerError("no valid placements in model output")
    return out


def _call_gemini(settings: Settings, *, system: str, user: str) -> str:
    try:
        from google import genai
        from google.genai import types
    except ImportError as e:
        raise GeminiPlannerError(
            "google-genai is not installed; uv sync with project deps"
        ) from e

    client = genai.Client(api_key=settings.gemini_api_key)
    response = client.models.generate_content(
        model=settings.gemini_model,
        contents=user,
        config=types.GenerateContentConfig(
            system_instruction=system,
            response_mime_type="application/json",
            temperature=0.2,
        ),
    )
    text = getattr(response, "text", None) or ""
    if not text and response.candidates:
        parts = response.candidates[0].content.parts
        text = "".join(getattr(p, "text", "") or "" for p in parts)
    if not text:
        raise GeminiPlannerError("empty Gemini response")
    return text


def _extract_json(raw: str) -> Any:
    raw = raw.strip()
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        match = re.search(r"\{[\s\S]*\}|\[[\s\S]*\]", raw)
        if not match:
            raise GeminiPlannerError(f"could not parse JSON: {raw[:200]}")
        return json.loads(match.group(0))


def _downsample_beats(beat_ms: list[int], *, max_points: int) -> list[int]:
    if len(beat_ms) <= max_points:
        return list(beat_ms)
    step = max(1, len(beat_ms) // max_points)
    sampled = beat_ms[::step]
    if sampled[-1] != beat_ms[-1]:
        sampled = sampled + [beat_ms[-1]]
    return sampled[:max_points]
