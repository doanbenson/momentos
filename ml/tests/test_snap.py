"""Tests for snap_to_grid (pure; no librosa)."""

import pytest

from momentos_ml.beat.grid import BeatGrid
from momentos_ml.beat.snap import PlannerPlacement, SnapError, snap_to_grid
from momentos_ml.features.extract import AssetFeature


def _grid(beats: list[int]) -> BeatGrid:
    return BeatGrid(
        song_pick_id="live-a-life-you-will-remember",
        audio_content_hash="test",
        beat_ms=beats,
        tempo_bpm=120.0,
    )


def test_snap_maps_half_open_beats_to_ms():
    grid = _grid([0, 500, 1000, 1500, 2000])
    placements = [
        PlannerPlacement(asset_id="a1", start_beat=0, end_beat=2),
        PlannerPlacement(asset_id="a2", start_beat=2, end_beat=4),
    ]
    clips = snap_to_grid(placements, grid)
    assert clips[0].start_ms == 0 and clips[0].end_ms == 1000
    assert clips[1].start_ms == 1000 and clips[1].end_ms == 2000
    assert [c.order for c in clips] == [0, 1]


def test_snap_extends_past_last_beat():
    grid = _grid([0, 500, 1000])
    placements = [PlannerPlacement(asset_id="a1", start_beat=1, end_beat=3)]
    clips = snap_to_grid(placements, grid)
    assert clips[0].start_ms == 500
    assert clips[0].end_ms == 1500  # last + median gap (500)


def test_snap_trims_video_to_source_duration():
    grid = _grid([0, 500, 1000, 1500, 2000])
    features = [
        AssetFeature(
            asset_id="v1", type="video", intake_order=0, duration_ms=600
        )
    ]
    placements = [PlannerPlacement(asset_id="v1", start_beat=0, end_beat=4)]
    clips = snap_to_grid(placements, grid, features)
    assert clips[0].start_ms == 0
    assert clips[0].end_ms == 600
    # Playback starts at file t=0 for the trimmed slot, not at timeline startMs.
    assert clips[0].source_in_ms == 0
    assert clips[0].source_out_ms == 600


def test_snap_rejects_hero_outro_shrunk_by_short_video():
    grid = _grid([0, 500, 1000, 1500, 2000])
    features = [
        AssetFeature(
            asset_id="v1", type="video", intake_order=0, duration_ms=600
        )
    ]
    placements = [
        PlannerPlacement(
            asset_id="v1", start_beat=0, end_beat=4, role="hero_outro"
        )
    ]
    with pytest.raises(SnapError, match="hero_outro"):
        snap_to_grid(placements, grid, features)


def test_snap_rejects_invalid_range():
    grid = _grid([0, 500, 1000])
    with pytest.raises(SnapError):
        snap_to_grid(
            [PlannerPlacement(asset_id="a1", start_beat=2, end_beat=2)],
            grid,
        )


def test_snap_requires_motion_preset_on_picture_clips():
    """Picture Assets must carry motionPreset when planner authors look fields (ADR 0015)."""
    grid = _grid([0, 500, 1000, 1500, 2000])
    features = [
        AssetFeature(asset_id="p1", type="photo", intake_order=0),
    ]
    placements = [
        PlannerPlacement(
            asset_id="p1",
            start_beat=0,
            end_beat=2,
            role="montage",
            act=2,
            motionPreset=None,
            visualStyle="clean_916",
            transitionIn="hard_cut",
        )
    ]
    with pytest.raises(SnapError, match="motionPreset"):
        snap_to_grid(placements, grid, features)


def test_snap_allows_null_motion_preset_on_voice():
    grid = _grid([0, 500, 1000, 1500, 2000])
    features = [
        AssetFeature(
            asset_id="v1", type="voice", intake_order=0, duration_ms=2000
        ),
    ]
    placements = [
        PlannerPlacement(
            asset_id="v1",
            start_beat=0,
            end_beat=4,
            role="hero_outro",
            act=3,
            motionPreset=None,
            visualStyle="clean_916",
            transitionIn="none",
        )
    ]
    clips = snap_to_grid(placements, grid, features)
    assert clips[0].motion_preset is None
    assert clips[0].visual_style == "clean_916"


def test_snap_photo_holds_full_beat_window_not_source_trimmed():
    """Photos (and 0ms sources) hold the target beat window — no SnapError."""
    grid = _grid([0, 500, 1000, 1500, 2000])
    features = [
        AssetFeature(
            asset_id="p1", type="photo", intake_order=0, duration_ms=10_000
        ),
        AssetFeature(
            asset_id="v0", type="video", intake_order=1, duration_ms=0
        ),
    ]
    placements = [
        PlannerPlacement(asset_id="p1", start_beat=0, end_beat=2, role="montage"),
        PlannerPlacement(
            asset_id="v0", start_beat=2, end_beat=4, role="hero_outro"
        ),
    ]
    clips = snap_to_grid(placements, grid, features)
    assert clips[0].start_ms == 0 and clips[0].end_ms == 1000
    assert clips[1].start_ms == 1000 and clips[1].end_ms == 2000
    # Photos and 0ms sources are holds, not file seeks at timeline startMs.
    assert clips[0].source_in_ms == 0 and clips[0].source_out_ms == 1000
    assert clips[1].source_in_ms == 0 and clips[1].source_out_ms == 1000
