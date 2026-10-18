"""Primary CI seam: stub draft → validated Cut (roles, captions, markers, hero_outro)."""

from unittest.mock import patch

from momentos_ml.agents.draft_cut import draft_cut
from momentos_ml.beat.grid import BeatGrid
from momentos_ml.config import Settings
from momentos_ml.styles.song_pick import (
    CHORUS_DROP_BEAT_INDEX,
    HERO_OUTRO_SPAN_BEATS,
)
from momentos_ml.validate.cut_schema import validate_cut


def _grid(n_beats: int = 120) -> BeatGrid:
    return BeatGrid(
        song_pick_id="live-a-life-you-will-remember",
        audio_content_hash="test",
        beat_ms=[i * 500 for i in range(n_beats)],
        tempo_bpm=120.0,
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


def test_stub_draft_produces_validated_cut_with_roles_captions_markers():
    """Seam 1: Reflection + montage Assets → OpenAPI-valid Cut with Act grammar."""
    reflection_id = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
    photo_a = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
    photo_b = "cccccccc-cccc-cccc-cccc-cccccccccccc"
    assets = [
        {
            "id": reflection_id,
            "type": "voice",
            "order": 0,
            "durationMs": 4000,
        },
        {"id": photo_a, "type": "photo", "order": 1},
        {"id": photo_b, "type": "photo", "order": 2},
    ]
    fake_grid = _grid(120)

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
            moment_id="dddddddd-dddd-dddd-dddd-dddddddddddd",
            settings=_settings(),
            reflection_asset_id=reflection_id,
            reflection_question="What made this week yours?",
        )

    validated = validate_cut(cut.model_dump(by_alias=True))
    roles = [c.role for c in validated.clips]
    assert "reflection" in roles
    assert "montage" in roles
    assert roles[-1] == "hero_outro"
    assert validated.clips[0].asset_id == reflection_id
    assert validated.clips[0].role == "reflection"

    # hero_outro on a legal 4-beat span (snap owns ms; 500ms/beat in fixture)
    hero = validated.clips[-1]
    assert hero.start_ms is not None and hero.end_ms is not None
    assert hero.end_ms - hero.start_ms == HERO_OUTRO_SPAN_BEATS * 500

    assert len(validated.captions) >= 1
    assert all(c.end_ms > c.start_ms for c in validated.captions)
    # Captions sit on the Reflection Act window
    assert validated.captions[0].start_ms >= validated.clips[0].start_ms

    marker_types = {m.type for m in validated.markers}
    assert "chorus_drop" in marker_types
    chorus = next(m for m in validated.markers if m.type == "chorus_drop")
    assert chorus.at_ms == fake_grid.beat_ms[CHORUS_DROP_BEAT_INDEX]

    assert all(c.start_ms is not None and c.end_ms is not None for c in validated.clips)
    assert validated.song_pick_id == "live-a-life-you-will-remember"
