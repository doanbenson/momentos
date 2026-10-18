"""Director: turn alignment + vision scores into beat-index placements + markers.

Agents emit beat indices; snap owns milliseconds (ADR 0002).
"""

from __future__ import annotations

from pydantic import BaseModel, Field

from momentos_ml.align.aligner import AlignmentResult
from momentos_ml.beat.grid import BeatGrid
from momentos_ml.beat.snap import PlannerPlacement
from momentos_ml.features.extract import AssetFeature
from momentos_ml.styles.song_pick import (
    CHORUS_DROP_BEAT_INDEX,
    HERO_OUTRO_SPAN_BEATS,
    PHOTO_HOLD_BEATS,
    is_legal_span_beats,
)
from momentos_ml.validate.cut_schema import CutCaption, CutMarkerType
from momentos_ml.vision.score import VisionScore


class PlannerMarker(BaseModel):
    """Marker on the Beat Grid; snap maps at_beat → atMs."""

    type: CutMarkerType
    at_beat: int = Field(ge=0)
    payload: dict | None = None


class DirectorPlan(BaseModel):
    placements: list[PlannerPlacement]
    markers: list[PlannerMarker]
    # Asset-local caption cues; draft_cut offsets onto Reflection timeline start.
    captions: list[CutCaption]


def plan_cut(
    features: list[AssetFeature],
    grid: BeatGrid,
    *,
    alignment: AlignmentResult,
    scores: list[VisionScore],
    reflection_asset_id: str | None = None,
    asset_order: list[str] | None = None,
) -> DirectorPlan:
    """
    Stub director for CI / CUT_PLANNER=stub.

    Places Reflection (Act 1) → scored montage (Act 2) → hero_outro (Act 3).
    Emits j_cut / chorus_drop / flash markers as beat indices.
    """
    n = len(grid.beat_ms)
    if n < 2:
        raise ValueError("BeatGrid needs at least 2 beats for director")

    by_id = {f.asset_id: f for f in features}
    reflection_id = reflection_asset_id or _infer_reflection_id(features)

    montage_ids = _montage_order(
        features=features,
        scores=scores,
        reflection_id=reflection_id,
        asset_order=asset_order,
    )

    placements: list[PlannerPlacement] = []
    cursor = 0
    reflection_end_beat = 0

    if reflection_id and reflection_id in by_id:
        feat = by_id[reflection_id]
        keep_ms = _keep_duration_ms(alignment, feat.duration_ms)
        span = _span_for_ms(grid, cursor, keep_ms)
        if span is None:
            span = (0, min(PHOTO_HOLD_BEATS, n))
        start, end = span
        placements.append(
            PlannerPlacement(
                asset_id=reflection_id,
                start_beat=start,
                end_beat=end,
                role="reflection",
            )
        )
        cursor = end
        reflection_end_beat = end

    # Reserve hero_outro span at the end of the packing cursor.
    hero_id = montage_ids[-1] if montage_ids else None
    body_ids = montage_ids[:-1] if hero_id else []

    for aid in body_ids:
        feat = by_id.get(aid)
        if feat is None:
            continue
        if cursor + HERO_OUTRO_SPAN_BEATS >= n:
            break
        span = _montage_span(feat, grid, cursor, reserve=HERO_OUTRO_SPAN_BEATS)
        if span is None:
            continue
        start, end = span
        placements.append(
            PlannerPlacement(
                asset_id=aid, start_beat=start, end_beat=end, role="montage"
            )
        )
        cursor = end

    if hero_id and hero_id in by_id:
        start = cursor
        end = start + HERO_OUTRO_SPAN_BEATS
        if end > n:
            # Prefer exclusive end == n when near the end of the grid.
            if start < n and (n - start) >= 1:
                # Still require legal 4-beat when possible; else clamp.
                if n - start >= HERO_OUTRO_SPAN_BEATS:
                    end = start + HERO_OUTRO_SPAN_BEATS
                else:
                    end = n
            else:
                start = max(0, n - HERO_OUTRO_SPAN_BEATS)
                end = n
        if end > start:
            span_beats = end - start
            if span_beats != HERO_OUTRO_SPAN_BEATS and start + HERO_OUTRO_SPAN_BEATS <= n:
                end = start + HERO_OUTRO_SPAN_BEATS
            placements.append(
                PlannerPlacement(
                    asset_id=hero_id,
                    start_beat=start,
                    end_beat=end,
                    role="hero_outro",
                )
            )
    elif placements and not any(p.role == "hero_outro" for p in placements):
        # No montage left — promote last placement to hero_outro and stretch to 4 beats.
        last = placements[-1]
        end = min(last.start_beat + HERO_OUTRO_SPAN_BEATS, n)
        if end <= last.start_beat:
            end = min(last.start_beat + 1, n) if last.start_beat < n else last.end_beat
        placements[-1] = PlannerPlacement(
            asset_id=last.asset_id,
            start_beat=last.start_beat,
            end_beat=end,
            role="hero_outro",
        )

    if not placements and features:
        feat = features[0]
        end = min(HERO_OUTRO_SPAN_BEATS, n)
        placements.append(
            PlannerPlacement(
                asset_id=feat.asset_id,
                start_beat=0,
                end_beat=max(end, 1),
                role="hero_outro",
            )
        )

    markers = _plan_markers(
        n=n,
        reflection_end_beat=reflection_end_beat,
        placements=placements,
    )
    return DirectorPlan(
        placements=placements,
        markers=markers,
        captions=list(alignment.captions),
    )


def _infer_reflection_id(features: list[AssetFeature]) -> str | None:
    voices = [f for f in features if f.type == "voice"]
    if voices:
        return sorted(voices, key=lambda f: f.intake_order)[0].asset_id
    videos = [f for f in features if f.type == "video"]
    if videos:
        return sorted(videos, key=lambda f: f.intake_order)[0].asset_id
    return None


def _montage_order(
    *,
    features: list[AssetFeature],
    scores: list[VisionScore],
    reflection_id: str | None,
    asset_order: list[str] | None,
) -> list[str]:
    by_id = {f.asset_id: f for f in features}
    if asset_order is not None:
        return [
            aid
            for aid in asset_order
            if aid in by_id and aid != reflection_id
        ]
    scored = [s.asset_id for s in scores if s.asset_id in by_id and s.asset_id != reflection_id]
    if scored:
        return scored
    return [
        f.asset_id
        for f in sorted(features, key=lambda x: x.intake_order)
        if f.asset_id != reflection_id
    ]


def _keep_duration_ms(alignment: AlignmentResult, fallback: int | None) -> int:
    if alignment.keep_segments:
        return max(seg.end_ms - seg.start_ms for seg in alignment.keep_segments)
    if fallback and fallback > 0:
        return fallback
    return 2000


def _span_for_ms(
    grid: BeatGrid, cursor: int, duration_ms: int
) -> tuple[int, int] | None:
    n = len(grid.beat_ms)
    if cursor >= n:
        return None
    start_ms = grid.beat_ms[cursor]
    end = cursor + 1
    while end < n and (grid.beat_ms[end] - start_ms) < duration_ms:
        end += 1
    if end <= cursor:
        return None
    # Prefer legal spans when lengthening by a beat still fits.
    span = end - cursor
    if not is_legal_span_beats(span):
        for legal in sorted(x for x in (1, 2, 4) if x >= span):
            if cursor + legal <= n:
                end = cursor + legal
                break
    return cursor, end


def _montage_span(
    feat: AssetFeature,
    grid: BeatGrid,
    cursor: int,
    *,
    reserve: int,
) -> tuple[int, int] | None:
    n = len(grid.beat_ms)
    if cursor >= n or cursor + reserve >= n:
        return None
    if feat.type == "photo":
        need = PHOTO_HOLD_BEATS
        end = cursor + need
        if end + reserve > n:
            return None
        return cursor, end
    if feat.duration_ms is not None and feat.duration_ms > 0:
        start_ms = grid.beat_ms[cursor]
        end = cursor + 1
        while (
            end < n - reserve
            and (grid.beat_ms[end] - start_ms) <= feat.duration_ms
        ):
            end += 1
        if end <= cursor:
            return None
        return cursor, end
    end = min(cursor + PHOTO_HOLD_BEATS, n - reserve)
    if end <= cursor:
        return None
    return cursor, end


def _plan_markers(
    *,
    n: int,
    reflection_end_beat: int,
    placements: list[PlannerPlacement],
) -> list[PlannerMarker]:
    markers: list[PlannerMarker] = []
    if reflection_end_beat > 0:
        markers.append(
            PlannerMarker(type="j_cut", at_beat=max(0, reflection_end_beat - 1))
        )

    drop = CHORUS_DROP_BEAT_INDEX
    if drop < n:
        markers.append(PlannerMarker(type="chorus_drop", at_beat=drop))
    elif n > 0:
        # Short fixture grids: clamp so CI still emits the marker type.
        markers.append(PlannerMarker(type="chorus_drop", at_beat=n - 1))

    # Flash on first montage placement start when present.
    for p in placements:
        if p.role == "montage":
            markers.append(
                PlannerMarker(
                    type="flash",
                    at_beat=p.start_beat,
                    payload={"systemMediaId": "flash"},
                )
            )
            break

    return markers
