"""Greenfield Cut draft: features → BeatGrid → planner → snap → validate."""

from __future__ import annotations

import logging
from typing import Any

from momentos_ml.agents import gemini_planner, stub_planner
from momentos_ml.beat.grid import BeatGrid, build_beat_grid, load_edit_audio_bytes
from momentos_ml.beat.snap import PlannerPlacement, SnapError, snap_to_grid
from momentos_ml.config import (
    MAX_REPAIR_ATTEMPTS,
    MVP_SONG_PICK_ID,
    Settings,
    load_settings,
)
from momentos_ml.features.extract import AssetFeature, JobAsset, extract_features
from momentos_ml.validate.cut_schema import Cut, build_cut

logger = logging.getLogger(__name__)


def draft_cut(
    assets: list[dict[str, Any]] | list[JobAsset],
    song_pick_id: str,
    *,
    moment_id: str,
    edit_audio_url: str | None = None,
    edit_audio_path: str | None = None,
    settings: Settings | None = None,
    version: int = 1,
    prior_hint: str | None = None,
    asset_order: list[str] | None = None,
) -> Cut:
    """
    Draft a schema-valid Cut.

    Retries planner ≤ MAX_REPAIR_ATTEMPTS on snap/validate failure, then raises.
    """
    settings = settings or load_settings()
    song_pick_id = song_pick_id or MVP_SONG_PICK_ID
    features = extract_features(assets)
    if not features:
        raise ValueError("no Assets to draft")

    audio = load_edit_audio_bytes(
        edit_audio_url=edit_audio_url,
        edit_audio_path=edit_audio_path or settings.edit_audio_path,
    )
    grid = build_beat_grid(
        audio,
        song_pick_id=song_pick_id,
        cache_dir=settings.beat_cache_dir,
    )

    return _plan_snap_validate(
        features=features,
        grid=grid,
        moment_id=moment_id,
        song_pick_id=song_pick_id,
        settings=settings,
        version=version,
        prior_hint=prior_hint,
        asset_order=asset_order,
    )


def _plan_snap_validate(
    *,
    features: list[AssetFeature],
    grid: BeatGrid,
    moment_id: str,
    song_pick_id: str,
    settings: Settings,
    version: int,
    prior_hint: str | None,
    asset_order: list[str] | None,
) -> Cut:
    last_err: Exception | None = None
    repair_note: str | None = None
    # First attempt + ≤ MAX_REPAIR_ATTEMPTS repairs
    for attempt in range(MAX_REPAIR_ATTEMPTS + 1):
        try:
            placements = _run_planner(
                features,
                grid,
                settings,
                prior_hint=prior_hint,
                asset_order=asset_order,
                repair_note=repair_note,
            )
            clips = snap_to_grid(placements, grid, features)
            return build_cut(
                moment_id=moment_id,
                song_pick_id=song_pick_id,
                clips=clips,
                version=version,
            )
        except (SnapError, ValueError, gemini_planner.GeminiPlannerError) as e:
            last_err = e
            repair_note = str(e)
            logger.warning(
                "cut plan attempt %s failed: %s", attempt + 1, e
            )
    raise RuntimeError(
        f"Cut draft failed after {MAX_REPAIR_ATTEMPTS + 1} attempts: {last_err}"
    ) from last_err


def _run_planner(
    features: list[AssetFeature],
    grid: BeatGrid,
    settings: Settings,
    *,
    prior_hint: str | None,
    asset_order: list[str] | None,
    repair_note: str | None,
) -> list[PlannerPlacement]:
    if settings.cut_planner == "stub":
        return stub_planner.plan_placements(
            features, grid, asset_order=asset_order
        )

    # gemini default; fall back to stub only if key missing would fail — let it raise
    return gemini_planner.plan_placements(
        features,
        grid,
        settings,
        prior_hint=prior_hint,
        repair_note=repair_note,
    )


# Back-compat shim for early scaffold callers
def draft_cut_legacy(assets: list[dict[str, Any]], song_pick_id: str) -> dict[str, Any]:
    """Minimal dict shape without BeatGrid — tests/scaffold only."""
    return {
        "version": 1,
        "songPickId": song_pick_id,
        "clips": [{"assetId": a["id"], "order": i} for i, a in enumerate(assets)],
    }
