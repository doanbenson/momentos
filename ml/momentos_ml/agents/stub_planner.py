"""Legacy stub placements helper.

Prefer `momentos_ml.agents.director.plan_cut` for the full Life On Film path
(roles / markers / captions). Kept for unit tests that assert packing only.
"""

from __future__ import annotations

from momentos_ml.beat.grid import BeatGrid
from momentos_ml.beat.snap import PlannerPlacement
from momentos_ml.config import PHOTO_HOLD_BEATS
from momentos_ml.features.extract import AssetFeature


def plan_placements(
    features: list[AssetFeature],
    grid: BeatGrid,
    *,
    asset_order: list[str] | None = None,
) -> list[PlannerPlacement]:
    """
    Place each Asset on successive beat pairs (or PHOTO_HOLD_BEATS for photos).

    Uses intake_order unless asset_order is provided (revise / reorder).
    Drops trailing assets that do not fit the remaining grid.
    """
    n = len(grid.beat_ms)
    if n < 2:
        raise ValueError("BeatGrid needs at least 2 beats for stub planner")

    by_id = {f.asset_id: f for f in features}
    if asset_order is not None:
        ordered = [by_id[aid] for aid in asset_order if aid in by_id]
    else:
        ordered = sorted(features, key=lambda f: f.intake_order)

    placements: list[PlannerPlacement] = []
    cursor = 0

    for feat in ordered:
        span = _span_beats(feat, grid, cursor)
        if span is None:
            continue  # drop — not enough grid left
        start, end = span
        placements.append(
            PlannerPlacement(
                asset_id=feat.asset_id,
                start_beat=start,
                end_beat=end,
                role="montage",
            )
        )
        cursor = end
        if cursor >= n:
            break

    if not placements and ordered:
        # Guarantee at least one clip for validity when grid is tiny.
        feat = ordered[0]
        end = min(PHOTO_HOLD_BEATS, n)
        if end <= 0:
            end = 1
        placements.append(
            PlannerPlacement(
                asset_id=feat.asset_id,
                start_beat=0,
                end_beat=end,
                role="montage",
            )
        )

    return placements


def _span_beats(
    feat: AssetFeature,
    grid: BeatGrid,
    cursor: int,
) -> tuple[int, int] | None:
    n = len(grid.beat_ms)
    if cursor >= n:
        return None

    if feat.type == "photo":
        need = PHOTO_HOLD_BEATS
        end = cursor + need
        if end > n:
            # Allow exclusive end == n via snap extension.
            if cursor >= n:
                return None
            end = n
            if end <= cursor:
                return None
        return cursor, end

    # video / voice: grow exclusive end while window ≤ source duration
    if feat.duration_ms is not None and feat.duration_ms > 0:
        start_ms = grid.beat_ms[cursor]
        end = cursor + 1
        while end < n and (grid.beat_ms[end] - start_ms) <= feat.duration_ms:
            end += 1
        if end == cursor + 1 and end < n:
            # Single beat interval longer than source — still place 1-beat span;
            # snap_to_grid will trim endMs to duration.
            pass
        if end <= cursor:
            return None
        return cursor, end

    # Unknown duration: default photo-sized hold
    end = min(cursor + PHOTO_HOLD_BEATS, n)
    if end <= cursor:
        return None
    return cursor, end
