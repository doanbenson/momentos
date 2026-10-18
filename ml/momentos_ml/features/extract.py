"""Metadata-only Asset → AssetFeature (vision scoring is a separate stage)."""

from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, Field

AssetType = Literal["photo", "video", "voice"]


class AssetFeature(BaseModel):
    asset_id: str
    type: AssetType
    intake_order: int
    duration_ms: int | None = None
    aspect: float | None = None  # width / height when both present


class JobAsset(BaseModel):
    """Asset slice from cut-draft job payload."""

    id: str
    type: AssetType
    order: int = 0
    duration_ms: int | None = Field(default=None, alias="durationMs")
    width: int | None = None
    height: int | None = None
    storage_url: str | None = Field(default=None, alias="storageUrl")

    model_config = {"populate_by_name": True}


def extract_features(assets: list[JobAsset] | list[dict[str, Any]]) -> list[AssetFeature]:
    """Derive AssetFeature[] from job Assets. storageUrl is ignored (no vision)."""
    features: list[AssetFeature] = []
    for raw in assets:
        a = raw if isinstance(raw, JobAsset) else JobAsset.model_validate(raw)
        aspect: float | None = None
        if a.width and a.height and a.height > 0:
            aspect = a.width / a.height
        duration = a.duration_ms
        if a.type == "photo":
            duration = None  # hold length comes from beat span after snap
        features.append(
            AssetFeature(
                asset_id=a.id,
                type=a.type,
                intake_order=a.order,
                duration_ms=duration,
                aspect=aspect,
            )
        )
    features.sort(key=lambda f: f.intake_order)
    return features
