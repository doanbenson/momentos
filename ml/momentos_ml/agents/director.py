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
    DROP_OFFSET_AFTER_SPEECH_SEC,
    HERO_OUTRO_SPAN_BEATS,
    J_CUT_PRE_ROLL_SEC,
    PHOTO_HOLD_BEATS,
    is_legal_span_beats,
)
from momentos_ml.validate.cut_schema import (
    CutCaption,
    CutClipAct,
    CutClipRole,
    CutMarkerType,
    CutMotionPreset,
    CutTransitionIn,
    CutVisualStyle,
)
from momentos_ml.vision.score import VisionScore

class PlacementImpossibleError(ValueError):
    """Every intake Asset cannot fit on the Beat Grid even at minimum legal spans."""


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

    Deterministic beat stepping (Life on Film):
      Act 1 Reflection → Act 3 photo/clip cycle at PHOTO_HOLD_BEATS each,
      final clip HERO_OUTRO_SPAN_BEATS. Every intake Asset appears ≥ once;
      when fewer than 10 montage Assets, cycle a second time to fill music.

    Authors Act / Motion Preset / Visual Style / transitionIn (ADR 0015 / 0020).
    Emits j_cut / chorus_drop / flash markers as beat indices from timing
    constants (ADR 0019 / 0021); snap owns ms.
    """
    n = len(grid.beat_ms)
    if n < 2:
        raise ValueError("BeatGrid needs at least 2 beats for director")

    by_id = {f.asset_id: f for f in features}
    reflection_id = reflection_asset_id or infer_reflection_id(features)

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
        # Leave room for montage schedule when montage Assets exist.
        if montage_ids:
            sequence = _montage_sequence(montage_ids)
            min_montage = _min_montage_beats(len(sequence))
            if end + min_montage > n:
                max_end = n - min_montage
                if max_end <= start:
                    raise PlacementImpossibleError(
                        f"cannot place all {len(features)} Assets on a "
                        f"{n}-beat grid: Reflection + montage need more beats "
                        f"than available"
                    )
                end = _clamp_legal_end(start, max_end, n)
        placements.append(
            _placement(
                asset_id=reflection_id,
                start_beat=start,
                end_beat=end,
                role="reflection",
                feat=feat,
                montage_index=None,
            )
        )
        cursor = end
        reflection_end_beat = end

    if montage_ids:
        placements.extend(
            _place_montage_cycle(
                montage_ids=montage_ids,
                by_id=by_id,
                grid=grid,
                cursor=cursor,
                total_assets=len(features),
            )
        )
    elif placements and not any(p.role == "hero_outro" for p in placements):
        # Voice-only / Reflection-only: promote to hero_outro, keep clean slate look.
        last = placements[-1]
        feat = by_id.get(last.asset_id)
        end = min(last.start_beat + HERO_OUTRO_SPAN_BEATS, n)
        if end <= last.start_beat:
            end = min(last.start_beat + 1, n) if last.start_beat < n else last.end_beat
        placements[-1] = _placement(
            asset_id=last.asset_id,
            start_beat=last.start_beat,
            end_beat=end,
            role="hero_outro",
            feat=feat,
            montage_index=None,
        )
    elif not placements and features:
        feat = features[0]
        end = min(HERO_OUTRO_SPAN_BEATS, n)
        placements.append(
            _placement(
                asset_id=feat.asset_id,
                start_beat=0,
                end_beat=max(end, 1),
                role="hero_outro",
                feat=feat,
                montage_index=None,
            )
        )

    # Every intake Asset must appear ≥ once (cycling may reuse montage ids).
    placed_ids = [p.asset_id for p in placements]
    expected_ids = {f.asset_id for f in features}
    if not expected_ids.issubset(set(placed_ids)):
        missing = sorted(expected_ids - set(placed_ids))
        raise PlacementImpossibleError(
            f"cannot place all Assets; missing={missing} "
            f"on a {n}-beat grid"
        )

    markers = _plan_markers(
        grid=grid,
        reflection_end_beat=reflection_end_beat,
        placements=placements,
    )
    return DirectorPlan(
        placements=placements,
        markers=markers,
        captions=list(alignment.captions),
    )


def infer_reflection_id(features: list[AssetFeature]) -> str | None:
    """Prefer explicit job id; fallback: first voice, else first video by intake order."""
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


# Cycle fill threshold: fewer unique montage Assets → second pass for music fill.
_MONTAGE_CYCLE_THRESHOLD = 10


def _montage_sequence(montage_ids: list[str]) -> list[str]:
    """Deterministic schedule: unique order, optionally doubled when short."""
    if not montage_ids:
        return []
    if len(montage_ids) < _MONTAGE_CYCLE_THRESHOLD:
        return list(montage_ids) + list(montage_ids)
    return list(montage_ids)


def _min_montage_beats(sequence_count: int) -> int:
    """Beats for a montage sequence (body @ PHOTO_HOLD + 4-beat hero)."""
    if sequence_count <= 0:
        return 0
    if sequence_count == 1:
        return HERO_OUTRO_SPAN_BEATS
    return (sequence_count - 1) * PHOTO_HOLD_BEATS + HERO_OUTRO_SPAN_BEATS


def _place_montage_cycle(
    *,
    montage_ids: list[str],
    by_id: dict[str, AssetFeature],
    grid: BeatGrid,
    cursor: int,
    total_assets: int,
) -> list[PlannerPlacement]:
    """Place montage Assets on fixed 2-beat holds; last clip is 4-beat hero_outro."""
    n = len(grid.beat_ms)
    sequence = _montage_sequence(montage_ids)
    available = n - cursor
    need = _min_montage_beats(len(sequence))
    # Prefer 2-beat holds; shrink body to 1-beat only when the grid is tight.
    body_hold = PHOTO_HOLD_BEATS
    if need > available:
        need_min = (
            HERO_OUTRO_SPAN_BEATS
            if len(sequence) == 1
            else (len(sequence) - 1) + HERO_OUTRO_SPAN_BEATS
        )
        if available < need_min:
            raise PlacementImpossibleError(
                f"cannot place all {total_assets} Assets on a {n}-beat grid: "
                f"need ≥{need_min} beats after Reflection for {len(sequence)} "
                f"montage slots (have {available})"
            )
        body_hold = 1

    placements: list[PlannerPlacement] = []
    cur = cursor
    last = len(sequence) - 1
    for i, aid in enumerate(sequence):
        feat = by_id.get(aid)
        if feat is None:
            raise PlacementImpossibleError(
                f"montage Asset {aid} missing from features"
            )
        hold = HERO_OUTRO_SPAN_BEATS if i == last else body_hold
        end = cur + hold
        if end > n:
            raise PlacementImpossibleError(
                f"cannot place montage schedule: ran out of Beat Grid room "
                f"at slot {i}/{len(sequence)}"
            )
        role: CutClipRole = "hero_outro" if i == last else "montage"
        placements.append(
            _placement(
                asset_id=aid,
                start_beat=cur,
                end_beat=end,
                role=role,
                feat=feat,
                montage_index=i,
            )
        )
        cur = end
    return placements


def _placement(
    *,
    asset_id: str,
    start_beat: int,
    end_beat: int,
    role: CutClipRole,
    feat: AssetFeature | None,
    montage_index: int | None,
) -> PlannerPlacement:
    act, motion, visual, transition = _author_look(
        role=role, feat=feat, montage_index=montage_index
    )
    return PlannerPlacement(
        asset_id=asset_id,
        start_beat=start_beat,
        end_beat=end_beat,
        role=role,
        act=act,
        motionPreset=motion,
        visualStyle=visual,
        transitionIn=transition,
    )


def _author_look(
    *,
    role: CutClipRole,
    feat: AssetFeature | None,
    montage_index: int | None,
) -> tuple[CutClipAct, CutMotionPreset | None, CutVisualStyle, CutTransitionIn]:
    """Stub Act / Motion / Visual / transitionIn (ADR 0015 / 0020 / 0021)."""
    asset_type = feat.type if feat else None
    is_voice = asset_type == "voice"
    is_picture = asset_type in ("photo", "video")

    if role == "reflection":
        return 1, None, "clean_916", "none"

    if role == "hero_outro":
        if is_voice or not is_picture:
            # Voice-only slate promoted to hero_outro — keep clean_916, no motion.
            return 3, None, "clean_916", "none"
        return 3, "handheld_super8_drift", "super8_gate", "hard_cut"

    # Act 3 photo/clip cycle — Life on Film look on every montage picture.
    idx = montage_index or 0
    motion: CutMotionPreset | None = None
    if is_picture:
        # First slot after Reflection: punch-in; rest: gate weave / no Ken Burns.
        motion = "whip_punch_in" if idx == 0 else "handheld_super8_drift"
    transition: CutTransitionIn = "hard_cut"
    if idx == 0 or (idx > 0 and idx % 4 == 0):
        transition = "white_flash"
    return 3, motion, "super8_gate", transition


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


def _clamp_legal_end(start: int, max_end: int, n: int) -> int:
    """Largest exclusive end ≤ max_end that yields a legal span when possible."""
    end = min(max_end, n)
    if end <= start:
        return start + 1 if start + 1 <= n else end
    span = end - start
    if is_legal_span_beats(span):
        return end
    for legal in sorted((1, 2, 4), reverse=True):
        if start + legal <= end:
            return start + legal
    return end


def _plan_markers(
    *,
    grid: BeatGrid,
    reflection_end_beat: int,
    placements: list[PlannerPlacement],
) -> list[PlannerMarker]:
    n = len(grid.beat_ms)
    markers: list[PlannerMarker] = []
    if reflection_end_beat > 0:
        speech_end_ms = _beat_end_ms(grid, reflection_end_beat)
        j_cut_ms = max(0, speech_end_ms - int(J_CUT_PRE_ROLL_SEC * 1000))
        markers.append(
            PlannerMarker(type="j_cut", at_beat=_beat_at_or_before(grid, j_cut_ms))
        )
        drop_ms = speech_end_ms + int(DROP_OFFSET_AFTER_SPEECH_SEC * 1000)
        markers.append(
            PlannerMarker(
                type="chorus_drop",
                at_beat=_beat_at_or_after(grid, drop_ms),
            )
        )

    # Flash on first montage start; pairs with transitionIn white_flash (ADR 0021).
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

    # Clamp any marker that slipped past a tiny fixture grid.
    clamped: list[PlannerMarker] = []
    for m in markers:
        at = min(max(m.at_beat, 0), n - 1)
        clamped.append(PlannerMarker(type=m.type, at_beat=at, payload=m.payload))
    return clamped


def _beat_end_ms(grid: BeatGrid, exclusive_end_beat: int) -> int:
    n = len(grid.beat_ms)
    if exclusive_end_beat < n:
        return grid.beat_ms[exclusive_end_beat]
    if exclusive_end_beat == n and n >= 2:
        gaps = [grid.beat_ms[i + 1] - grid.beat_ms[i] for i in range(n - 1)]
        gaps.sort()
        return grid.beat_ms[-1] + max(gaps[len(gaps) // 2], 1)
    if n:
        return grid.beat_ms[-1]
    return 0


def _beat_at_or_before(grid: BeatGrid, target_ms: int) -> int:
    """Largest beat index with beat_ms[i] ≤ target_ms."""
    best = 0
    for i, ms in enumerate(grid.beat_ms):
        if ms <= target_ms:
            best = i
        else:
            break
    return best


def _beat_at_or_after(grid: BeatGrid, target_ms: int) -> int:
    """Smallest beat index with beat_ms[i] ≥ target_ms (clamped to last beat)."""
    n = len(grid.beat_ms)
    for i, ms in enumerate(grid.beat_ms):
        if ms >= target_ms:
            return i
    return max(n - 1, 0)
