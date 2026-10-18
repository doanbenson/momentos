"""Gemini montage-order failures fall back; deterministic beats still succeed."""

from __future__ import annotations

import tempfile
from pathlib import Path
from unittest.mock import patch

from momentos_ml.agents import gemini_planner
from momentos_ml.agents.draft_cut import draft_cut
from momentos_ml.align.aligner import AlignmentResult
from momentos_ml.beat.grid import BeatGrid
from momentos_ml.config import Settings


def _grid(n_beats: int = 120) -> BeatGrid:
    return BeatGrid(
        song_pick_id="live-a-life-you-will-remember",
        audio_content_hash="test",
        beat_ms=[i * 500 for i in range(n_beats)],
        tempo_bpm=120.0,
    )


def _gemini_settings() -> Settings:
    return Settings(
        cut_planner="gemini",
        vision_scoring="metadata",
        gemini_api_key="fake-key",
        gemini_model="gemini-3.8-flash",
        moment_service_url=None,
        job_queue_url=None,
        internal_service_token=None,
        edit_audio_path=None,
        poll_interval_s=0.1,
        beat_cache_dir=".beat_cache_test",
    )


def test_gemini_order_failure_falls_back_to_chronological():
    reflection_id = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
    photo_a = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
    photo_b = "5ec655c2-a0dc-41cc-809c-6e42c466a2af"
    assets = [
        {"id": reflection_id, "type": "voice", "order": 0, "durationMs": 3000},
        {"id": photo_a, "type": "photo", "order": 1},
        {"id": photo_b, "type": "photo", "order": 2},
    ]

    with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
        tmp.write(b"RIFF")
        tmp_path = Path(tmp.name)

    try:
        with (
            patch(
                "momentos_ml.agents.draft_cut.load_edit_audio_bytes",
                return_value=b"fake-audio",
            ),
            patch(
                "momentos_ml.agents.draft_cut.build_beat_grid",
                return_value=_grid(),
            ),
            patch(
                "momentos_ml.agents.draft_cut.materialize_asset",
                return_value=tmp_path,
            ),
            patch(
                "momentos_ml.agents.gemini_planner.plan_montage_order",
                side_effect=gemini_planner.GeminiPlannerError("timeout"),
            ),
            patch(
                "momentos_ml.agents.draft_cut.align_reflection",
                return_value=AlignmentResult(captions=[], keep_segments=[]),
            ),
        ):
            cut = draft_cut(
                assets,
                "live-a-life-you-will-remember",
                moment_id="dddddddd-dddd-dddd-dddd-dddddddddddd",
                settings=_gemini_settings(),
                reflection_asset_id=reflection_id,
            )
            assert {c.asset_id for c in cut.clips} >= {
                reflection_id,
                photo_a,
                photo_b,
            }
            # Chronological fallback: photo_a before photo_b among first-pass montage.
            montage = [c.asset_id for c in cut.clips if c.role == "montage"]
            assert montage[0] == photo_a
    finally:
        tmp_path.unlink(missing_ok=True)


def test_gemini_montage_order_is_honored():
    reflection_id = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
    photo_a = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
    photo_b = "5ec655c2-a0dc-41cc-809c-6e42c466a2af"
    assets = [
        {"id": reflection_id, "type": "voice", "order": 0, "durationMs": 3000},
        {"id": photo_a, "type": "photo", "order": 1},
        {"id": photo_b, "type": "photo", "order": 2},
    ]

    with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
        tmp.write(b"RIFF")
        tmp_path = Path(tmp.name)

    try:
        with (
            patch(
                "momentos_ml.agents.draft_cut.load_edit_audio_bytes",
                return_value=b"fake-audio",
            ),
            patch(
                "momentos_ml.agents.draft_cut.build_beat_grid",
                return_value=_grid(),
            ),
            patch(
                "momentos_ml.agents.draft_cut.materialize_asset",
                return_value=tmp_path,
            ),
            patch(
                "momentos_ml.agents.gemini_planner.plan_montage_order",
                return_value=[photo_b, photo_a],
            ),
            patch(
                "momentos_ml.agents.draft_cut.align_reflection",
                return_value=AlignmentResult(captions=[], keep_segments=[]),
            ),
        ):
            cut = draft_cut(
                assets,
                "live-a-life-you-will-remember",
                moment_id="dddddddd-dddd-dddd-dddd-dddddddddddd",
                settings=_gemini_settings(),
                reflection_asset_id=reflection_id,
            )
            first_picture = next(
                c for c in cut.clips if c.role in ("montage", "hero_outro")
            )
            assert first_picture.asset_id == photo_b
    finally:
        tmp_path.unlink(missing_ok=True)
