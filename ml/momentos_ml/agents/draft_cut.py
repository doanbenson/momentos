"""Cut draft pipeline: align → vision → director → snap → validate."""

from __future__ import annotations

import logging
from typing import Any

from momentos_ml.agents import gemini_planner
from momentos_ml.agents.director import (
    DirectorPlan,
    PlacementImpossibleError,
    plan_cut as director_plan_cut,
)
from momentos_ml.align.aligner import AlignmentResult, align_reflection
from momentos_ml.beat.grid import BeatGrid, build_beat_grid, load_edit_audio_bytes
from momentos_ml.beat.snap import (
    PlannerPlacement,
    SnapError,
    offset_captions,
    snap_markers,
    snap_to_grid,
)
from momentos_ml.config import (
    MAX_REPAIR_ATTEMPTS,
    MVP_SONG_PICK_ID,
    Settings,
    load_settings,
)
from momentos_ml.agents.director import infer_reflection_id
from momentos_ml.features.extract import AssetFeature, JobAsset, extract_features
from momentos_ml.media.download import materialize_asset
from momentos_ml.styles.song_pick import HERO_OUTRO_SPAN_BEATS
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

    Stage order (ADR 0013 / hackathon spec):
      aligner → vision scoring → director/planner → snap → validate.

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

    last_err: Exception | None = None
    repair_note: str | None = None
    for attempt in range(MAX_REPAIR_ATTEMPTS + 1):
        try:
            plan = _run_director(
                features,
                grid,
                settings,
                alignment=alignment,
                scores=scores,
                reflection_asset_id=reflection_id,
                prior_hint=prior_hint,
                asset_order=asset_order,
                repair_note=repair_note,
            )
            _assert_place_all_once(plan.placements, features)
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
        except PlacementImpossibleError:
            # Place-all-once failures are not repairable by re-prompting (ADR 0017).
            raise
        except (SnapError, ValueError, gemini_planner.GeminiPlannerError) as e:
            last_err = e
            repair_note = str(e)
            logger.warning("cut plan attempt %s failed: %s", attempt + 1, e)
    raise RuntimeError(
        f"Cut draft failed after {MAX_REPAIR_ATTEMPTS + 1} attempts: {last_err}"
    ) from last_err


def _assert_place_all_once(
    placements: list[PlannerPlacement], features: list[AssetFeature]
) -> None:
    """Every Moment Asset appears exactly once (ADR 0017)."""
    placed = [p.asset_id for p in placements]
    expected = {f.asset_id for f in features}
    if set(placed) != expected:
        missing = sorted(expected - set(placed))
        raise PlacementImpossibleError(
            f"cannot place all Assets once; missing={missing} (place-all-once)"
        )
    if len(placed) != len(set(placed)):
        raise PlacementImpossibleError(
            "multi-cycle Asset reuse is not allowed in MVP (place-all-once)"
        )


def _timeline_captions(plan: DirectorPlan, clips: list) -> list:
    reflection = next((c for c in clips if c.role == "reflection"), None)
    start_ms = reflection.start_ms if reflection and reflection.start_ms is not None else 0
    return offset_captions(plan.captions, start_ms)


def _run_director(
    features: list[AssetFeature],
    grid: BeatGrid,
    settings: Settings,
    *,
    alignment: AlignmentResult,
    scores: list,
    reflection_asset_id: str | None,
    prior_hint: str | None,
    asset_order: list[str] | None,
    repair_note: str | None,
) -> DirectorPlan:
    if settings.cut_planner == "stub":
        return director_plan_cut(
            features,
            grid,
            alignment=alignment,
            scores=scores,
            reflection_asset_id=reflection_asset_id,
            asset_order=asset_order,
        )

    # Gemini emits placements; wrap with stub markers/captions from aligner.
    placements = gemini_planner.plan_placements(
        features,
        grid,
        settings,
        prior_hint=prior_hint,
        repair_note=repair_note,
    )
    # Infer roles + force legal 4-beat hero_outro (Gemini often emits 2-beat finale).
    enriched = _enrich_gemini_placements(
        placements,
        reflection_asset_id=reflection_asset_id,
        beat_count=len(grid.beat_ms),
    )
    stub = director_plan_cut(
        features,
        grid,
        alignment=alignment,
        scores=scores,
        reflection_asset_id=reflection_asset_id,
        asset_order=asset_order,
    )
    return DirectorPlan(
        placements=enriched,
        markers=stub.markers,
        captions=alignment.captions,
    )


def _enrich_gemini_placements(
    placements: list[PlannerPlacement],
    *,
    reflection_asset_id: str | None,
    beat_count: int,
) -> list[PlannerPlacement]:
    if not placements:
        return placements
    out: list[PlannerPlacement] = []
    last = len(placements) - 1
    for i, p in enumerate(placements):
        role = p.role
        start_beat = p.start_beat
        end_beat = p.end_beat
        if reflection_asset_id and p.asset_id == reflection_asset_id and i == 0:
            role = "reflection"
        elif i == last:
            role = "hero_outro"
            # Deterministic 4-beat exclusive span (mirrors stub director).
            max_start = max(0, beat_count - HERO_OUTRO_SPAN_BEATS)
            if start_beat > max_start:
                start_beat = max_start
            end_beat = min(start_beat + HERO_OUTRO_SPAN_BEATS, beat_count)
            if end_beat <= start_beat:
                end_beat = min(start_beat + HERO_OUTRO_SPAN_BEATS, beat_count)
        else:
            role = "montage"
        out.append(
            PlannerPlacement(
                asset_id=p.asset_id,
                start_beat=start_beat,
                end_beat=end_beat,
                role=role,
            )
        )
    return out


def draft_cut_legacy(assets: list[dict[str, Any]], song_pick_id: str) -> dict[str, Any]:
    """Minimal dict shape without BeatGrid — tests/scaffold only."""
    return {
        "version": 1,
        "songPickId": song_pick_id,
        "clips": [{"assetId": a["id"], "order": i} for i, a in enumerate(assets)],
    }
