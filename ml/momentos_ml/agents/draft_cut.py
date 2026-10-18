"""Cut draft pipeline: align → vision → director → snap → validate."""

from __future__ import annotations

import logging
from typing import Any

from momentos_ml.agents import gemini_planner
from momentos_ml.agents.director import (
    DirectorPlan,
    PlacementImpossibleError,
    infer_reflection_id,
    plan_cut as director_plan_cut,
)
from momentos_ml.align.aligner import align_reflection
from momentos_ml.beat.grid import BeatGrid, build_beat_grid, load_edit_audio_bytes
from momentos_ml.beat.snap import (
    PlannerPlacement,
    offset_captions,
    snap_markers,
    snap_to_grid,
)
from momentos_ml.config import (
    MVP_SONG_PICK_ID,
    Settings,
    load_settings,
)
from momentos_ml.features.extract import AssetFeature, JobAsset, extract_features
from momentos_ml.media.download import materialize_asset
from momentos_ml.validate.cut_schema import Cut, build_cut
from momentos_ml.vision.score import score_montage_candidates

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
    reflection_asset_id: str | None = None,
    reflection_question: str | None = None,
) -> Cut:
    """
    Draft a schema-valid Cut.

    Stage order:
      aligner → vision scoring → optional Gemini montage order →
      deterministic beat stepping → snap → validate.

    Beat indices are authored in Python (not by the LLM). Gemini may only
    reorder montage Asset ids; failures fall back to intake/chronological order.
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
        reflection_asset_id=reflection_asset_id,
        reflection_question=reflection_question,
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
    reflection_asset_id: str | None,
    reflection_question: str | None,
) -> Cut:
    reflection_id = reflection_asset_id or infer_reflection_id(features)
    reflection_feat = next(
        (f for f in features if f.asset_id == reflection_id), None
    )

    aligner_backend = "stub" if settings.cut_planner == "stub" else "faster_whisper"
    audio_path: str | None = None
    reflection_local = None
    if aligner_backend == "faster_whisper":
        if reflection_feat is None:
            raise ValueError(
                "faster_whisper requires a Reflection Asset in the job features; "
                f"reflection_asset_id={reflection_id!r}"
            )
        storage_url = (reflection_feat.storage_url or "").strip()
        reflection_local = materialize_asset(
            storage_url=storage_url or None,
            asset_type=reflection_feat.type,
        )
        if reflection_local is not None:
            audio_path = str(reflection_local)
        else:
            raise ValueError(
                "faster_whisper requires Reflection Asset download; "
                f"storageUrl_empty={not storage_url} reflection_id={reflection_id!r}. "
                "Ensure MEDIA_SKIP_VALIDATE=false and media sign-download returns downloadUrl."
            )

    try:
        alignment = align_reflection(
            reflection_asset_id=reflection_id or "unknown",
            reflection_question=reflection_question,
            duration_ms=reflection_feat.duration_ms if reflection_feat else None,
            audio_path=audio_path,
            backend=aligner_backend,  # type: ignore[arg-type]
        )
    finally:
        if reflection_local is not None:
            reflection_local.unlink(missing_ok=True)

    scores = score_montage_candidates(
        features,
        reflection_asset_id=reflection_id,
        settings=settings,
    )

    order = _resolve_montage_order(
        features=features,
        settings=settings,
        reflection_asset_id=reflection_id,
        asset_order=asset_order,
        prior_hint=prior_hint,
    )
    plan = director_plan_cut(
        features,
        grid,
        alignment=alignment,
        scores=scores,
        reflection_asset_id=reflection_id,
        asset_order=order,
    )
    _assert_every_asset_placed(plan.placements, features)
    clips = snap_to_grid(plan.placements, grid, features)
    markers = snap_markers(plan.markers, grid)
    captions = _timeline_captions(plan, clips)
    return build_cut(
        moment_id=moment_id,
        song_pick_id=song_pick_id,
        clips=clips,
        captions=captions,
        markers=markers,
        version=version,
    )


def _resolve_montage_order(
    *,
    features: list[AssetFeature],
    settings: Settings,
    reflection_asset_id: str | None,
    asset_order: list[str] | None,
    prior_hint: str | None,
) -> list[str] | None:
    """Optional Gemini storytelling order; always safe to fall back."""
    if asset_order is not None:
        return asset_order
    chrono = [
        f.asset_id
        for f in sorted(features, key=lambda x: x.intake_order)
        if f.asset_id != reflection_asset_id
    ]
    if settings.cut_planner != "gemini":
        return None  # director uses vision scores / intake order
    try:
        ordered = gemini_planner.plan_montage_order(
            features,
            settings,
            reflection_asset_id=reflection_asset_id,
            prior_hint=prior_hint,
        )
        if ordered:
            return ordered
    except gemini_planner.GeminiPlannerError as e:
        logger.warning("Gemini montage order failed; using chronological: %s", e)
    return chrono


def _assert_every_asset_placed(
    placements: list[PlannerPlacement], features: list[AssetFeature]
) -> None:
    """Every Moment Asset appears ≥ once (cycling may reuse montage ids)."""
    placed = [p.asset_id for p in placements]
    expected = {f.asset_id for f in features}
    if not expected.issubset(set(placed)):
        missing = sorted(expected - set(placed))
        raise PlacementImpossibleError(
            f"cannot place all Assets; missing={missing}"
        )


def _timeline_captions(plan: DirectorPlan, clips: list) -> list:
    reflection = next((c for c in clips if c.role == "reflection"), None)
    if reflection is None:
        # Voice-only Reflection is promoted to hero_outro but still carries sourceInMs.
        reflection = next(
            (c for c in clips if c.source_in_ms and c.source_in_ms > 0),
            None,
        )
    start_ms = (
        reflection.start_ms
        if reflection is not None and reflection.start_ms is not None
        else 0
    )
    source_in = (
        reflection.source_in_ms
        if reflection is not None and reflection.source_in_ms is not None
        else 0
    )
    return offset_captions(plan.captions, start_ms, source_in)


def draft_cut_legacy(assets: list[dict[str, Any]], song_pick_id: str) -> dict[str, Any]:
    """Minimal dict shape without BeatGrid — tests/scaffold only."""
    return {
        "version": 1,
        "songPickId": song_pick_id,
        "clips": [{"assetId": a["id"], "order": i} for i, a in enumerate(assets)],
    }
