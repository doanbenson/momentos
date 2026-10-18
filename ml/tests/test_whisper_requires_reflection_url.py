"""Whisper path must hard-fail when Reflection storageUrl is missing."""

from unittest.mock import patch

import pytest

from momentos_ml.agents.draft_cut import draft_cut
from momentos_ml.beat.grid import BeatGrid
from momentos_ml.config import Settings


def _grid(n_beats: int = 32) -> BeatGrid:
    return BeatGrid(
        song_pick_id="live-a-life-you-will-remember",
        audio_content_hash="test",
        beat_ms=[i * 476 for i in range(n_beats)],
        tempo_bpm=126.0,
    )


def _settings(**kwargs) -> Settings:
    base = dict(
        cut_planner="gemini",
        vision_scoring="metadata",
        gemini_api_key="test-key",
        gemini_model="gemini-3.8-flash",
        moment_service_url=None,
        job_queue_url=None,
        internal_service_token=None,
        edit_audio_path=None,
        poll_interval_s=2.0,
        beat_cache_dir=".beat_cache",
    )
    base.update(kwargs)
    return Settings(**base)


def test_draft_cut_raises_when_whisper_lacks_reflection_url():
    assets = [
        {
            "id": "11111111-1111-1111-1111-111111111111",
            "type": "voice",
            "durationMs": 3000,
            "storageUrl": "",
        },
        {
            "id": "22222222-2222-2222-2222-222222222222",
            "type": "photo",
            "intakeOrder": 0,
        },
    ]
    with (
        patch(
            "momentos_ml.agents.draft_cut.load_edit_audio_bytes",
            return_value=b"fake-audio",
        ),
        patch(
            "momentos_ml.agents.draft_cut.build_beat_grid",
            return_value=_grid(),
        ),
        pytest.raises(ValueError, match="storageUrl_empty=True"),
    ):
        draft_cut(
            assets,
            "live-a-life-you-will-remember",
            moment_id="44444444-4444-4444-4444-444444444444",
            settings=_settings(),
            reflection_asset_id="11111111-1111-1111-1111-111111111111",
        )


def test_stub_planner_still_allows_missing_reflection_url():
    """CUT_PLANNER=stub uses stub aligner — empty Reflection URL is OK."""
    assets = [
        {
            "id": "11111111-1111-1111-1111-111111111111",
            "type": "voice",
            "durationMs": 3000,
        },
        {
            "id": "22222222-2222-2222-2222-222222222222",
            "type": "photo",
            "intakeOrder": 0,
        },
        {
            "id": "33333333-3333-3333-3333-333333333333",
            "type": "photo",
            "intakeOrder": 1,
        },
    ]
    with (
        patch(
            "momentos_ml.agents.draft_cut.load_edit_audio_bytes",
            return_value=b"fake-audio",
        ),
        patch(
            "momentos_ml.agents.draft_cut.build_beat_grid",
            return_value=_grid(120),
        ),
    ):
        cut = draft_cut(
            assets,
            "live-a-life-you-will-remember",
            moment_id="44444444-4444-4444-4444-444444444444",
            settings=_settings(cut_planner="stub"),
            reflection_asset_id="11111111-1111-1111-1111-111111111111",
        )
    assert any(c.role == "hero_outro" for c in cut.clips)
