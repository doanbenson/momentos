"""Gemini system/user templates for montage storytelling order."""

from __future__ import annotations

import json

from momentos_ml.features.extract import AssetFeature
from momentos_ml.styles import mvp


ORDER_SYSTEM_PROMPT = """You are the Momentos montage orderer for a Life On Film Moment.
You ONLY choose the visual storytelling order of photo/video Assets.

Rules:
- Output ONLY valid JSON matching the schema (no markdown).
- Return every montage asset_id exactly once — do not invent ids.
- Do NOT emit beat indices or milliseconds (Python owns timing).
- Prefer emotional arc: establish → build → peak → settle.
"""


def build_order_user_prompt(
    *,
    montage_features: list[AssetFeature],
    prior_hint: str | None = None,
) -> str:
    ids = [f.asset_id for f in montage_features]
    payload = {
        "styleProfile": mvp.MVP_STYLE,
        "vibe": mvp.MVP_VIBE,
        "montageAssetIds": ids,
        "assets": [f.model_dump() for f in montage_features],
        "outputSchema": {"asset_ids": ["uuid", "..."]},
    }
    text = (
        "Order these montage Assets for the best visual story.\n"
        f"Include every id in montageAssetIds exactly once: {json.dumps(ids)}\n"
        f"{json.dumps(payload, indent=2)}\n"
    )
    if prior_hint:
        text += f"\nConstraints from Editor revise:\n{prior_hint}\n"
    return text


# Legacy aliases kept so older imports/tests keep resolving.
SYSTEM_PROMPT = ORDER_SYSTEM_PROMPT
REPAIR_SUFFIX = "\nReturn corrected asset_ids JSON only.\n"


def build_user_prompt(
    *,
    features: list[AssetFeature],
    beat_count: int,
    beat_ms_sample: list[int],
    song_pick_id: str,
    prior_hint: str | None = None,
) -> str:
    del beat_count, beat_ms_sample, song_pick_id
    return build_order_user_prompt(
        montage_features=features, prior_hint=prior_hint
    )
