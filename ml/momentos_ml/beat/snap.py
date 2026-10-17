"""Pure snap: PlannerPlacement beat indices → startMs/endMs (ADR 0002)."""

from __future__ import annotations

from pydantic import BaseModel, Field

from momentos_ml.beat.grid import BeatGrid
from momentos_ml.features.extract import AssetFeature
from momentos_ml.validate.cut_schema import CutClip


class PlannerPlacement(BaseModel):
    """Planner DTO. end_beat is exclusive (half-open [start_beat, end_beat))."""

    asset_id: str
    start_beat: int = Field(ge=0)
    end_beat: int = Field(ge=1)


class SnapError(ValueError):
    """Illegal placement that cannot be repaired by clamping alone."""


def snap_to_grid(
    placements: list[PlannerPlacement],
    grid: BeatGrid,
    features: list[AssetFeature] | None = None,
) -> list[CutClip]:
    """
    Map beat indices → clip startMs/endMs.

    Rules:
    - start_beat / end_beat index into BeatGrid.beat_ms; end exclusive.
    - Durations must be > 0; timeline must be non-decreasing by order.
    - Video/voice windows trimmed to ≤ source duration_ms when known.
    """
    if not grid.beat_ms:
        raise SnapError("BeatGrid has no beats")

    by_id = {f.asset_id: f for f in (features or [])}
    n = len(grid.beat_ms)
    clips: list[CutClip] = []

    for order, p in enumerate(placements):
        if p.start_beat < 0 or p.end_beat <= p.start_beat:
            raise SnapError(
                f"asset {p.asset_id}: invalid beat range [{p.start_beat}, {p.end_beat})"
            )
        if p.start_beat >= n:
            raise SnapError(
                f"asset {p.asset_id}: start_beat {p.start_beat} past grid ({n} beats)"
            )

        # Clamp end to last available beat index + allow last beat as open end
        # by using grid duration extrapolation when end_beat == n.
        start_ms = grid.beat_ms[p.start_beat]
        if p.end_beat < n:
            end_ms = grid.beat_ms[p.end_beat]
        elif p.end_beat == n:
            # Exclusive past last beat: extend by median beat interval.
            end_ms = _extend_past_last(grid)
        else:
            raise SnapError(
                f"asset {p.asset_id}: end_beat {p.end_beat} past grid ({n} beats)"
            )

        if end_ms <= start_ms:
            raise SnapError(
                f"asset {p.asset_id}: non-positive duration ({start_ms}→{end_ms})"
            )

        feat = by_id.get(p.asset_id)
        if feat and feat.type in ("video", "voice") and feat.duration_ms is not None:
            max_end = start_ms + feat.duration_ms
            if end_ms > max_end:
                # Shrink beat window to fit source; if still invalid, fail.
                end_ms = max_end
                if end_ms <= start_ms:
                    raise SnapError(
                        f"asset {p.asset_id}: source duration {feat.duration_ms}ms "
                        "too short for any positive window"
                    )

        clips.append(
            CutClip(
                assetId=p.asset_id,
                order=order,
                startMs=start_ms,
                endMs=end_ms,
            )
        )

    # Monotonic timeline: each clip starts at/after previous start
    prev_start = -1
    for c in clips:
        assert c.start_ms is not None
        if c.start_ms < prev_start:
            raise SnapError(
                f"clip order {c.order}: startMs {c.start_ms} before prior {prev_start}"
            )
        prev_start = c.start_ms

    return clips


def _extend_past_last(grid: BeatGrid) -> int:
    beats = grid.beat_ms
    if len(beats) >= 2:
        gaps = [beats[i + 1] - beats[i] for i in range(len(beats) - 1)]
        gaps.sort()
        median = gaps[len(gaps) // 2]
        return beats[-1] + max(median, 1)
    return beats[-1] + 500
