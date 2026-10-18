"""Deterministic Cut draft: reflection + N zero-duration photos, no SnapError."""

from __future__ import annotations

from unittest.mock import patch

from momentos_ml.agents.draft_cut import draft_cut
from momentos_ml.beat.grid import BeatGrid
from momentos_ml.config import Settings
from momentos_ml.styles.song_pick import HERO_OUTRO_SPAN_BEATS, PHOTO_HOLD_BEATS
from momentos_ml.validate.cut_schema import validate_cut


def _grid(n_beats: int = 200, *, ms_per_beat: int = 476) -> BeatGrid:
    """~126 BPM grid (60_000/126 ≈ 476ms)."""
    return BeatGrid(
        song_pick_id="live-a-life-you-will-remember",
        audio_content_hash="test",
        beat_ms=[i * ms_per_beat for i in range(n_beats)],
        tempo_bpm=126.0,
    )


def _settings() -> Settings:
    return Settings(
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


def _uuid(n: int) -> str:
    return f"{n:08x}-aaaa-aaaa-aaaa-aaaaaaaaaaaa"


def test_draft_succeeds_on_reflection_plus_15_zero_ms_photos():
    """1 Reflection + 15 photo Assets with durationMs=0 → valid Cut, no SnapError."""
    reflection_id = _uuid(1)
    photo_ids = [_uuid(100 + i) for i in range(15)]
    assets = [
        {
            "id": reflection_id,
            "type": "voice",
            "order": 0,
            "durationMs": 4000,
        },
        *[
            {"id": pid, "type": "photo", "order": i + 1, "durationMs": 0}
            for i, pid in enumerate(photo_ids)
        ],
    ]

    with (
        patch(
            "momentos_ml.agents.draft_cut.load_edit_audio_bytes",
            return_value=b"fake-audio",
        ),
        patch(
            "momentos_ml.agents.draft_cut.build_beat_grid",
            return_value=_grid(200),
        ),
    ):
        cut = draft_cut(
            assets,
            "live-a-life-you-will-remember",
            moment_id="dddddddd-dddd-dddd-dddd-dddddddddddd",
            settings=_settings(),
            reflection_asset_id=reflection_id,
            reflection_question="What made this week yours?",
        )

    validated = validate_cut(cut.model_dump(by_alias=True))
    placed = [c.asset_id for c in validated.clips]
    assert placed[0] == reflection_id
    assert validated.clips[0].role == "reflection"
    assert validated.clips[-1].role == "hero_outro"
    # Every uploaded Asset appears at least once.
    assert set(photo_ids).issubset(set(placed))
    assert reflection_id in placed
    # Montage holds are deterministic 2-beat (final hero is 4-beat).
    body = [c for c in validated.clips if c.role == "montage"]
    for c in body:
        assert c.start_ms is not None and c.end_ms is not None
        span_ms = c.end_ms - c.start_ms
        assert span_ms == PHOTO_HOLD_BEATS * 476
    hero = validated.clips[-1]
    assert hero.end_ms - hero.start_ms == HERO_OUTRO_SPAN_BEATS * 476


def test_draft_cycles_montage_when_fewer_than_10_photos():
    """Fewer than 10 montage Assets → second cycle so montage fills music."""
    reflection_id = _uuid(1)
    photo_ids = [_uuid(200 + i) for i in range(4)]
    assets = [
        {
            "id": reflection_id,
            "type": "voice",
            "order": 0,
            "durationMs": 3000,
        },
        *[
            {"id": pid, "type": "photo", "order": i + 1, "durationMs": 0}
            for i, pid in enumerate(photo_ids)
        ],
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
            moment_id="dddddddd-dddd-dddd-dddd-dddddddddddd",
            settings=_settings(),
            reflection_asset_id=reflection_id,
        )

    validated = validate_cut(cut.model_dump(by_alias=True))
    placed = [c.asset_id for c in validated.clips if c.role != "reflection"]
    # First pass places all 4; cycle adds more so montage picture count > 4.
    assert len(placed) > len(photo_ids)
    assert set(photo_ids).issubset(set(placed))
