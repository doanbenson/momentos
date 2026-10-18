"""Score montage candidate Assets before director placement."""

from __future__ import annotations

from pydantic import BaseModel, Field

from momentos_ml.features.extract import AssetFeature


class VisionScore(BaseModel):
    asset_id: str
    score: float = Field(ge=0.0)


def score_montage_candidates(
    features: list[AssetFeature],
    *,
    reflection_asset_id: str | None = None,
) -> list[VisionScore]:
    """
    Rank montage candidates (excludes Reflection).

    Stub is deterministic metadata-only scoring for CI. A future vision model
    can replace this without changing the director seam.
    """
    scores: list[VisionScore] = []
    for feat in features:
        if reflection_asset_id and feat.asset_id == reflection_asset_id:
            continue
        if feat.type == "voice" and (
            reflection_asset_id is None or feat.asset_id == reflection_asset_id
        ):
            # Voice without being the Reflection is rare; deprioritize.
            if feat.asset_id == reflection_asset_id:
                continue
        base = 0.5
        if feat.type == "video":
            base = 0.8
        elif feat.type == "photo":
            base = 0.7
        elif feat.type == "voice":
            base = 0.3
        # Prefer earlier intake slightly (stable tie-break via order).
        order_boost = max(0.0, 0.2 - 0.01 * feat.intake_order)
        scores.append(VisionScore(asset_id=feat.asset_id, score=base + order_boost))

    scores.sort(key=lambda s: (-s.score, s.asset_id))
    return scores
