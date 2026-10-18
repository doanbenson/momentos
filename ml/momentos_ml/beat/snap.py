"""Pure snap: PlannerPlacement beat indices → startMs/endMs (ADR 0002)."""

from __future__ import annotations

from pydantic import BaseModel, Field

from momentos_ml.beat.grid import BeatGrid
from momentos_ml.features.extract import AssetFeature
from momentos_ml.validate.cut_schema import CutCaption, CutClip, CutClipRole, CutMarker


class PlannerPlacement(BaseModel):
    """Planner DTO. end_beat is exclusive (half-open [start_beat, end_beat))."""

    asset_id: str
    start_beat: int = Field(ge=0)
    end_beat: int = Field(ge=1)
    role: CutClipRole = "montage"


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
    - role is carried from the planner (director); snap does not invent roles
      except a last-clip hero_outro safety net for older planners.
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

        start_ms = grid.beat_ms[p.start_beat]
        if p.end_beat < n:
            end_ms = grid.beat_ms[p.end_beat]
        elif p.end_beat == n:
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
                role=p.role,
            )
        )

    prev_start = -1
    for c in clips:
        assert c.start_ms is not None
        if c.start_ms < prev_start:
            raise SnapError(
                f"clip order {c.order}: startMs {c.start_ms} before prior {prev_start}"
            )
        prev_start = c.start_ms

    # Safety net for older planners that omit hero_outro.
    if clips and not any(c.role == "hero_outro" for c in clips):
        last = clips[-1]
        clips[-1] = CutClip(
            assetId=last.asset_id,
            order=last.order,
            startMs=last.start_ms,
            endMs=last.end_ms,
            role="hero_outro",
        )

    return clips


def snap_markers(
    markers: list,
    grid: BeatGrid,
) -> list[CutMarker]:
    """Map planner markers (at_beat) → CutMarker atMs."""
    if not grid.beat_ms:
        raise SnapError("BeatGrid has no beats")
    n = len(grid.beat_ms)
    out: list[CutMarker] = []
    for m in markers:
        at_beat = int(m.at_beat)
        if at_beat < 0 or at_beat >= n:
            raise SnapError(f"marker {m.type}: at_beat {at_beat} out of grid ({n})")
        payload = getattr(m, "payload", None)
        out.append(
            CutMarker(
                type=m.type,
                atMs=grid.beat_ms[at_beat],
                payload=payload,
            )
        )
    return out


def offset_captions(
    captions: list[CutCaption] | list[dict],
    reflection_start_ms: int,
) -> list[CutCaption]:
    """Shift Asset-local caption cues onto the Cut timeline."""
    out: list[CutCaption] = []
    for c in captions:
        if isinstance(c, CutCaption):
            text, start, end = c.text, c.start_ms, c.end_ms
        else:
            text = str(c["text"])
            start = int(c.get("startMs", c.get("start_ms", 0)))
            end = int(c.get("endMs", c.get("end_ms", 0)))
        out.append(
            CutCaption(
                text=text,
                startMs=start + reflection_start_ms,
                endMs=end + reflection_start_ms,
            )
        )
    return out


def _extend_past_last(grid: BeatGrid) -> int:
    beats = grid.beat_ms
    if len(beats) >= 2:
        gaps = [beats[i + 1] - beats[i] for i in range(len(beats) - 1)]
        gaps.sort()
        median = gaps[len(gaps) // 2]
        return beats[-1] + max(median, 1)
    return beats[-1] + 500
