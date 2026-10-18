"""Tests for stub planner + draft path with synthetic BeatGrid."""

from unittest.mock import patch

from momentos_ml.agents.stub_planner import plan_placements
from momentos_ml.beat.grid import BeatGrid
from momentos_ml.beat.snap import snap_to_grid
from momentos_ml.config import Settings
from momentos_ml.features.extract import AssetFeature, extract_features
from momentos_ml.validate.cut_schema import validate_cut


def _grid(n_beats: int = 20) -> BeatGrid:
    return BeatGrid(
        song_pick_id="live-a-life-you-will-remember",
        audio_content_hash="test",
        beat_ms=[i * 500 for i in range(n_beats)],
        tempo_bpm=120.0,
    )


def test_stub_packs_photos_on_successive_holds():
    features = [
        AssetFeature(asset_id="p0", type="photo", intake_order=0),
        AssetFeature(asset_id="p1", type="photo", intake_order=1),
        AssetFeature(asset_id="p2", type="photo", intake_order=2),
    ]
    placements = plan_placements(features, _grid())
    assert len(placements) == 3
    assert placements[0].start_beat == 0 and placements[0].end_beat == 2
    assert placements[1].start_beat == 2 and placements[1].end_beat == 4
    assert placements[2].start_beat == 4 and placements[2].end_beat == 6


def test_stub_honors_asset_order():
    features = [
        AssetFeature(asset_id="a", type="photo", intake_order=0),
        AssetFeature(asset_id="b", type="photo", intake_order=1),
    ]
    placements = plan_placements(features, _grid(), asset_order=["b", "a"])
    assert [p.asset_id for p in placements] == ["b", "a"]


def test_stub_snap_validate_produces_openapi_cut():
    from momentos_ml.validate.cut_schema import build_cut

    assets = [
        {"id": "11111111-1111-1111-1111-111111111111", "type": "photo", "order": 0},
        {
            "id": "22222222-2222-2222-2222-222222222222",
            "type": "video",
            "order": 1,
            "durationMs": 2000,
        },
    ]
    features = extract_features(assets)
    grid = _grid()
    placements = plan_placements(features, grid)
    clips = snap_to_grid(placements, grid, features)
    cut = build_cut(
        moment_id="33333333-3333-3333-3333-333333333333",
        song_pick_id="live-a-life-you-will-remember",
        clips=clips,
    )
    validated = validate_cut(cut.model_dump(by_alias=True))
    assert validated.song_pick_id == "live-a-life-you-will-remember"
    assert all(c.start_ms is not None and c.end_ms is not None for c in validated.clips)


def test_draft_cut_with_stub_and_mock_audio():
    from momentos_ml.agents.draft_cut import draft_cut

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
    fake_grid = _grid(16)
    assets = [
        {"id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "type": "photo", "order": 0},
        {"id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "type": "photo", "order": 1},
    ]

    with (
        patch(
            "momentos_ml.agents.draft_cut.load_edit_audio_bytes",
            return_value=b"fake-audio",
        ),
        patch(
            "momentos_ml.agents.draft_cut.build_beat_grid",
            return_value=fake_grid,
        ),
    ):
        cut = draft_cut(
            assets,
            "live-a-life-you-will-remember",
            moment_id="cccccccc-cccc-cccc-cccc-cccccccccccc",
            settings=settings,
        )

    assert len(cut.clips) == 2
    assert cut.clips[0].start_ms == 0
    assert cut.version == 1
