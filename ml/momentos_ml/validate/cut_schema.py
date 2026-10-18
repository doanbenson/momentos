"""Pydantic Cut / clips aligned with packages/contracts/openapi.yaml."""

from __future__ import annotations

from typing import Any, Literal
from uuid import uuid4

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

CutClipRole = Literal["reflection", "montage", "hero_outro"]
CutMarkerType = Literal["j_cut", "chorus_drop", "flash", "sfx"]


class CutClip(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    asset_id: str = Field(alias="assetId")
    order: int
    start_ms: int | None = Field(default=None, alias="startMs")
    end_ms: int | None = Field(default=None, alias="endMs")
    role: CutClipRole

    @model_validator(mode="after")
    def _window_ok(self) -> CutClip:
        if self.start_ms is not None and self.end_ms is not None:
            if self.start_ms < 0 or self.end_ms < 0:
                raise ValueError("startMs/endMs must be non-negative")
            if self.end_ms <= self.start_ms:
                raise ValueError("endMs must be > startMs")
        return self


class CutCaption(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    text: str
    start_ms: int = Field(alias="startMs")
    end_ms: int = Field(alias="endMs")

    @model_validator(mode="after")
    def _window_ok(self) -> CutCaption:
        if self.start_ms < 0 or self.end_ms < 0:
            raise ValueError("startMs/endMs must be non-negative")
        if self.end_ms <= self.start_ms:
            raise ValueError("endMs must be > startMs")
        return self


class CutMarker(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    type: CutMarkerType
    at_ms: int = Field(alias="atMs")
    payload: dict[str, Any] | None = None


class Cut(BaseModel):
    """OpenAPI Cut. id may be minted here if moment-service assigns on persist."""

    model_config = ConfigDict(populate_by_name=True)

    id: str
    moment_id: str = Field(alias="momentId")
    version: int = 1
    song_pick_id: str | None = Field(default=None, alias="songPickId")
    clips: list[CutClip]
    captions: list[CutCaption] = Field(default_factory=list)
    markers: list[CutMarker] = Field(default_factory=list)

    @field_validator("clips")
    @classmethod
    def _orders_unique(cls, clips: list[CutClip]) -> list[CutClip]:
        orders = [c.order for c in clips]
        if len(orders) != len(set(orders)):
            raise ValueError("clip orders must be unique")
        return clips

    @model_validator(mode="after")
    def _require_hero_outro(self) -> Cut:
        if not any(c.role == "hero_outro" for c in self.clips):
            raise ValueError("Cut must include a clip with role hero_outro")
        return self


def validate_cut(data: Cut | dict[str, Any]) -> Cut:
    """Validate and return a Cut. Accepts camelCase or snake_case keys."""
    if isinstance(data, Cut):
        return Cut.model_validate(data.model_dump(by_alias=True))
    return Cut.model_validate(data)


def build_cut(
    *,
    moment_id: str,
    song_pick_id: str,
    clips: list[CutClip],
    version: int = 1,
    cut_id: str | None = None,
    captions: list[CutCaption] | list[dict[str, Any]] | None = None,
    markers: list[CutMarker] | list[dict[str, Any]] | None = None,
) -> Cut:
    payload: dict[str, Any] = {
        "id": cut_id or str(uuid4()),
        "momentId": moment_id,
        "version": version,
        "songPickId": song_pick_id,
        "clips": [c.model_dump(by_alias=True) for c in clips],
        "captions": [],
        "markers": [],
    }
    if captions:
        payload["captions"] = [
            c.model_dump(by_alias=True) if isinstance(c, CutCaption) else c for c in captions
        ]
    if markers:
        payload["markers"] = [
            m.model_dump(by_alias=True) if isinstance(m, CutMarker) else m for m in markers
        ]
    return validate_cut(payload)
