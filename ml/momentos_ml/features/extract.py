"""Metadata-only Asset → AssetFeature (vision scoring is a separate stage)."""

from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, Field

AssetType = Literal["photo", "video", "voice"]

# Still images / unknown-duration picture Assets can be held for any positive
# beat window. Downstream snap/validators need a positive source duration.
PHOTO_SYNTHETIC_DURATION_MS = 10_000


class AssetFeature(BaseModel):
    asset_id: str
    type: AssetType
    intake_order: int
    duration_ms: int | None = None
    aspect: float | None = None  # width / height when both present
    storage_url: str | None = None  # signed download URL when job provides it
    mime_type: str | None = None


class JobAsset(BaseModel):
    """Asset slice from cut-draft job payload."""

    id: str
    type: AssetType
    order: int = 0
    duration_ms: int | None = Field(default=None, alias="durationMs")
    width: int | None = None
    height: int | None = None
    storage_url: str | None = Field(default=None, alias="storageUrl")
    mime_type: str | None = Field(default=None, alias="mimeType")

    model_config = {"populate_by_name": True}


def _is_still_or_zero_duration(a: JobAsset) -> bool:
    if a.type == "photo":
        return True
    mime = (a.mime_type or "").lower()
    if mime.startswith("image/"):
        return True
    return a.duration_ms is not None and a.duration_ms == 0


def extract_features(assets: list[JobAsset] | list[dict[str, Any]]) -> list[AssetFeature]:
    """Derive AssetFeature[] from job Assets (keeps storageUrl for align/vision)."""
    features: list[AssetFeature] = []
    for raw in assets:
        a = raw if isinstance(raw, JobAsset) else JobAsset.model_validate(raw)
        aspect: float | None = None
        if a.width and a.height and a.height > 0:
            aspect = a.width / a.height
        duration = a.duration_ms
        if _is_still_or_zero_duration(a):
            duration = PHOTO_SYNTHETIC_DURATION_MS
        features.append(
            AssetFeature(
                asset_id=a.id,
                type=a.type,
                intake_order=a.order,
                duration_ms=duration,
                aspect=aspect,
                storage_url=a.storage_url,
                mime_type=a.mime_type,
            )
        )
    features.sort(key=lambda f: f.intake_order)
    return features
