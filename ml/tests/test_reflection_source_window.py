"""Reflection playback starts at the outer speech window, not file t=0."""

from __future__ import annotations

from unittest.mock import patch

import pytest

from momentos_ml.agents.director import plan_cut
from momentos_ml.agents.draft_cut import draft_cut
from momentos_ml.align.aligner import AlignmentResult, KeepSegment
from momentos_ml.beat.grid import BeatGrid
from momentos_ml.beat.snap import (
    PlannerPlacement,
    SnapError,
    offset_captions,
    snap_to_grid,
)
from momentos_ml.config import Settings
from momentos_ml.features.extract import AssetFeature
from momentos_ml.validate.cut_schema import CutCaption


def _grid(n_beats: int = 40, *, ms_per_beat: int = 500) -> BeatGrid:
    return BeatGrid(
        song_pick_id="live-a-life-you-will-remember",
        audio_content_hash="test",
        beat_ms=[i * ms_per_beat for i in range(n_beats)],
        tempo_bpm=120.0,
    )


def _speech() -> AlignmentResult:
    """Leading silence, then two keep-segments with a pause between them."""
    return AlignmentResult(
        captions=[
            CutCaption(text="hello", startMs=1500, endMs=2000),
            CutCaption(text="there", startMs=4000, endMs=4600),
        ],
        keep_segments=[
            KeepSegment(start_ms=1500, end_ms=2000),
            KeepSegment(start_ms=4000, end_ms=4600),
        ],
    )


def test_director_reflection_uses_outer_speech_span_not_max_segment():
    """Timeline length is last-keep minus first-keep, pauses included."""
    voice = AssetFeature(
        asset_id="voice", type="voice", intake_order=0, duration_ms=12_000
    )
    photo = AssetFeature(
        asset_id="photo", type="photo", intake_order=1, duration_ms=10_000
    )
    plan = plan_cut(
        [voice, photo],
        _grid(),
        alignment=_speech(),
        scores=[],
        reflection_asset_id="voice",
    )
    reflection = plan.placements[0]
    assert reflection.role == "reflection"
    assert reflection.source_in_ms == 1500
    assert reflection.source_out_ms == 4600
    # 3100ms outer span → 7 beats @ 500ms. Max segment alone is 600ms (2 beats).
    assert reflection.end_beat - reflection.start_beat == 7
    # Captions stay asset-local until draft_cut offsets them.
    assert plan.captions[0].start_ms == 1500
    assert plan.captions[1].start_ms == 4000


def test_director_reflection_video_keeps_act1_native_crop():
    video = AssetFeature(
        asset_id="vid", type="video", intake_order=0, duration_ms=8_000
    )
    photo = AssetFeature(asset_id="photo", type="photo", intake_order=1)
    alignment = AlignmentResult(
        captions=[CutCaption(text="hi", startMs=500, endMs=2500)],
        keep_segments=[KeepSegment(start_ms=500, end_ms=2500)],
    )
    plan = plan_cut(
        [video, photo],
        _grid(),
        alignment=alignment,
        scores=[],
        reflection_asset_id="vid",
    )
    reflection = plan.placements[0]
    assert reflection.role == "reflection"
    assert reflection.act == 1
    assert reflection.motion_preset == "native_action_crop"
    assert reflection.visual_style == "clean_916"
    assert reflection.transition_in == "none"
    assert reflection.source_in_ms == 500
    assert reflection.source_out_ms == 2500


def test_voice_only_hero_keeps_speech_window():
    voice = AssetFeature(
        asset_id="voice", type="voice", intake_order=0, duration_ms=9_000
    )
    alignment = AlignmentResult(
        captions=[CutCaption(text="hi", startMs=2000, endMs=7000)],
        keep_segments=[KeepSegment(start_ms=2000, end_ms=7000)],
    )
    plan = plan_cut(
        [voice],
        _grid(20),
        alignment=alignment,
        scores=[],
        reflection_asset_id="voice",
    )
    hero = plan.placements[0]
    assert hero.role == "hero_outro"
    assert hero.source_in_ms == 2000
    assert hero.source_out_ms == 7000
    assert hero.end_beat - hero.start_beat == 4


def test_snap_reflection_persists_speech_window_and_trims_timeline():
    grid = _grid(16)
    features = [
        AssetFeature(
            asset_id="voice", type="voice", intake_order=0, duration_ms=12_000
        )
    ]
    # Beats would land a 3500ms slot; speech window is 3100ms, so trim the slot.
    placements = [
        PlannerPlacement(
            asset_id="voice",
            start_beat=0,
            end_beat=7,
            role="reflection",
            act=1,
            motionPreset=None,
            visualStyle="clean_916",
            transitionIn="none",
            sourceInMs=1500,
            sourceOutMs=4600,
        )
    ]
    clips = snap_to_grid(placements, grid, features)
    clip = clips[0]
    assert clip.source_in_ms == 1500
    assert clip.source_out_ms == 4600
    assert clip.start_ms == 0
    assert clip.end_ms - clip.start_ms == 3100
    assert clip.source_out_ms - clip.source_in_ms == 3100


def test_snap_reflection_source_is_not_timeline_start():
    grid = _grid(12)
    features = [
        AssetFeature(
            asset_id="voice", type="voice", intake_order=0, duration_ms=12_000
        ),
        AssetFeature(
            asset_id="photo", type="photo", intake_order=1, duration_ms=10_000
        ),
    ]
    placements = [
        PlannerPlacement(
            asset_id="voice",
            start_beat=2,
            end_beat=6,
            role="reflection",
            act=1,
            motionPreset=None,
            visualStyle="clean_916",
            transitionIn="none",
            sourceInMs=1500,
            sourceOutMs=3100,
        ),
        PlannerPlacement(
            asset_id="photo",
            start_beat=6,
            end_beat=10,
            role="hero_outro",
            act=3,
            motionPreset="handheld_super8_drift",
            visualStyle="super8_gate",
            transitionIn="hard_cut",
        ),
    ]
    clips = snap_to_grid(placements, grid, features)
    voice, photo = clips
    assert voice.start_ms == 1000
    assert voice.source_in_ms == 1500
    assert voice.source_out_ms == 3100
    assert voice.source_in_ms != voice.start_ms
    # Window is 1600ms; the 2000ms beat slot is trimmed down to it.
    assert voice.end_ms - voice.start_ms == 1600
    assert photo.source_in_ms == 0
    assert photo.source_out_ms == photo.end_ms - photo.start_ms
    assert photo.start_ms != 0
    assert photo.source_in_ms != photo.start_ms


def test_snap_other_video_plays_from_file_zero_for_timeline_slot():
    grid = _grid(12)
    features = [
        AssetFeature(
            asset_id="vid", type="video", intake_order=0, duration_ms=5_000
        ),
        AssetFeature(
            asset_id="hero", type="video", intake_order=1, duration_ms=5_000
        ),
    ]
    placements = [
        PlannerPlacement(
            asset_id="vid",
            start_beat=2,
            end_beat=4,
            role="montage",
            act=3,
            motionPreset="whip_punch_in",
            visualStyle="super8_gate",
            transitionIn="hard_cut",
        ),
        PlannerPlacement(
            asset_id="hero",
            start_beat=4,
            end_beat=8,
            role="hero_outro",
            act=3,
            motionPreset="handheld_super8_drift",
            visualStyle="super8_gate",
            transitionIn="hard_cut",
        ),
    ]
    clips = snap_to_grid(placements, grid, features)
    montage = clips[0]
    assert montage.start_ms == 1000
    assert montage.end_ms - montage.start_ms == 1000
    assert montage.source_in_ms == 0
    assert montage.source_out_ms == 1000


def test_snap_keeps_explicit_window_on_non_reflection_video():
    """A planner-supplied source window is not replaced by timeline startMs."""
    grid = _grid(12)
    features = [
        AssetFeature(
            asset_id="vid", type="video", intake_order=0, duration_ms=8_000
        ),
        AssetFeature(
            asset_id="hero", type="photo", intake_order=1, duration_ms=10_000
        ),
    ]
    placements = [
        PlannerPlacement(
            asset_id="vid",
            start_beat=2,
            end_beat=6,
            role="montage",
            act=3,
            motionPreset="whip_punch_in",
            visualStyle="super8_gate",
            transitionIn="hard_cut",
            sourceInMs=400,
            sourceOutMs=1200,
        ),
        PlannerPlacement(
            asset_id="hero",
            start_beat=6,
            end_beat=10,
            role="hero_outro",
            act=3,
            motionPreset="handheld_super8_drift",
            visualStyle="super8_gate",
            transitionIn="hard_cut",
        ),
    ]
    clips = snap_to_grid(placements, grid, features)
    montage = clips[0]
    assert montage.start_ms == 1000
    assert montage.end_ms - montage.start_ms == 2000
    assert montage.source_in_ms == 400
    assert montage.source_out_ms == 1200


def test_snap_hero_outro_refuses_speech_window_shorter_than_legal_span():
    """File is long enough; the speech window is not. Do not shrink the hero."""
    grid = _grid(8)
    features = [
        AssetFeature(
            asset_id="voice", type="voice", intake_order=0, duration_ms=9_000
        )
    ]
    placements = [
        PlannerPlacement(
            asset_id="voice",
            start_beat=0,
            end_beat=4,
            role="hero_outro",
            act=3,
            motionPreset=None,
            visualStyle="clean_916",
            transitionIn="none",
            sourceInMs=200,
            sourceOutMs=800,
        )
    ]
    with pytest.raises(SnapError, match="hero_outro"):
        snap_to_grid(placements, grid, features)


def test_offset_captions_uses_timeline_start_minus_source_in():
    shifted = offset_captions(
        [
            CutCaption(text="hello", startMs=1500, endMs=2000),
            {"text": "there", "startMs": 4000, "endMs": 4600},
        ],
        reflection_start_ms=1000,
        source_in_ms=1500,
    )
    # T lands at timelineStart + (T - sourceInMs)
    assert shifted[0].start_ms == 1000
    assert shifted[0].end_ms == 1500
    assert shifted[1].start_ms == 3500
    assert shifted[1].end_ms == 4100


def test_draft_cut_burns_words_on_trimmed_voice():
    reflection_id = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
    photo_id = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
    assets = [
        {"id": reflection_id, "type": "voice", "order": 0, "durationMs": 12000},
        {"id": photo_id, "type": "photo", "order": 1},
    ]
    settings = Settings(
        cut_planner="stub",
        vision_scoring="metadata",
        gemini_api_key=None,
        gemini_model="gemini-3.8-flash",
        moment_service_url=None,
        job_queue_url=None,
        internal_service_token=None,
        edit_audio_path=None,
        poll_interval_s=0.1,
        beat_cache_dir=".beat_cache_test",
    )
    with (
        patch(
            "momentos_ml.agents.draft_cut.load_edit_audio_bytes",
            return_value=b"fake-audio",
        ),
        patch(
            "momentos_ml.agents.draft_cut.build_beat_grid",
            return_value=_grid(80),
        ),
        patch(
            "momentos_ml.agents.draft_cut.align_reflection",
            return_value=_speech(),
        ),
    ):
        cut = draft_cut(
            assets,
            "live-a-life-you-will-remember",
            moment_id="dddddddd-dddd-dddd-dddd-dddddddddddd",
            settings=settings,
            reflection_asset_id=reflection_id,
            reflection_question="ignored when align is patched",
        )

    reflection = next(c for c in cut.clips if c.role == "reflection")
    assert reflection.act == 1
    assert reflection.motion_preset is None
    assert reflection.visual_style == "clean_916"
    assert reflection.source_in_ms == 1500
    assert reflection.source_out_ms == 4600
    assert reflection.start_ms == 0
    assert reflection.end_ms is not None
    assert reflection.end_ms - reflection.start_ms == 3100

    photo = next(c for c in cut.clips if c.asset_id == photo_id)
    assert photo.source_in_ms == 0
    assert photo.source_out_ms == (photo.end_ms - photo.start_ms)

    # hello @ file 1500 → timeline 0; there @ file 4000 → timeline 2500.
    assert cut.captions[0].text == "hello"
    assert cut.captions[0].start_ms == 0
    assert cut.captions[0].end_ms == 500
    assert cut.captions[1].text == "there"
    assert cut.captions[1].start_ms == 2500
    assert cut.captions[1].end_ms == 3100
