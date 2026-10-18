"""Primary CI seam: stub draft → validated Cut (place-all-once, motion/look, timing)."""

from __future__ import annotations

from unittest.mock import patch

import pytest

from momentos_ml.agents.draft_cut import draft_cut
from momentos_ml.beat.grid import BeatGrid
from momentos_ml.config import Settings
from momentos_ml.styles.song_pick import (
    DROP_OFFSET_AFTER_SPEECH_SEC,
    HERO_OUTRO_SPAN_BEATS,
    J_CUT_PRE_ROLL_SEC,
)
from momentos_ml.validate.cut_schema import validate_cut


def _grid(n_beats: int = 120, *, ms_per_beat: int = 500) -> BeatGrid:
    return BeatGrid(
        song_pick_id="live-a-life-you-will-remember",
        audio_content_hash="test",
        beat_ms=[i * ms_per_beat for i in range(n_beats)],
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


def _uuid(n: int) -> str:
    return f"{n:08x}-aaaa-aaaa-aaaa-aaaaaaaaaaaa"


def _draft(assets: list[dict], *, grid: BeatGrid, reflection_id: str):
    with (
        patch(
            "momentos_ml.agents.draft_cut.load_edit_audio_bytes",
            return_value=b"fake-audio",
        ),
        patch(
            "momentos_ml.agents.draft_cut.build_beat_grid",
            return_value=grid,
        ),
    ):
        return draft_cut(
            assets,
            "live-a-life-you-will-remember",
            moment_id="dddddddd-dddd-dddd-dddd-dddddddddddd",
            settings=_settings(),
            reflection_asset_id=reflection_id,
            reflection_question="What made this week yours?",
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

    cut = _draft(assets, grid=fake_grid, reflection_id=reflection_id)

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
    assert "j_cut" in marker_types
    assert "chorus_drop" in marker_types

    assert all(c.start_ms is not None and c.end_ms is not None for c in validated.clips)
    assert validated.song_pick_id == "live-a-life-you-will-remember"


def test_stub_draft_places_every_asset_once_shortening_holds():
    """Place-all-once: shrink photo holds to min legal spans before dropping (ADR 0017)."""
    reflection_id = _uuid(1)
    photo_ids = [_uuid(10 + i) for i in range(8)]
    assets = [
        {
            "id": reflection_id,
            "type": "voice",
            "order": 0,
            "durationMs": 2000,
        },
        *[{"id": pid, "type": "photo", "order": i + 1} for i, pid in enumerate(photo_ids)],
    ]
    # Tight grid: preferred 2-beat photo holds would silently drop; min 1-beat fits.
    fake_grid = _grid(20)

    cut = _draft(assets, grid=fake_grid, reflection_id=reflection_id)
    validated = validate_cut(cut.model_dump(by_alias=True))

    placed = [c.asset_id for c in validated.clips]
    assert set(placed) == {reflection_id, *photo_ids}
    assert len(placed) == len(set(placed))  # no multi-cycle reuse
    assert validated.clips[-1].role == "hero_outro"


def test_stub_draft_fails_clearly_when_placement_impossible():
    """When even min legal spans cannot fit every Asset, cut-draft fails (ADR 0017)."""
    reflection_id = _uuid(1)
    photo_ids = [_uuid(20 + i) for i in range(10)]
    assets = [
        {
            "id": reflection_id,
            "type": "voice",
            "order": 0,
            "durationMs": 2000,
        },
        *[{"id": pid, "type": "photo", "order": i + 1} for i, pid in enumerate(photo_ids)],
    ]
    # reflection (~4 beats) + 9 body (1) + hero (4) = 17 > 12 beats → impossible
    fake_grid = _grid(12)

    with pytest.raises((ValueError, RuntimeError), match=r"(?i)place|fit|impossible|asset"):
        _draft(assets, grid=fake_grid, reflection_id=reflection_id)


def test_stub_draft_authors_act_motion_look_and_transition_in():
    """Planner-authored Act / Motion Preset / Visual Style / transitionIn (ADR 0015/0020)."""
    reflection_id = _uuid(1)
    photo_a = _uuid(2)
    photo_b = _uuid(3)
    assets = [
        {
            "id": reflection_id,
            "type": "voice",
            "order": 0,
            "durationMs": 3000,
        },
        {"id": photo_a, "type": "photo", "order": 1},
        {"id": photo_b, "type": "photo", "order": 2},
    ]
    cut = _draft(assets, grid=_grid(40), reflection_id=reflection_id)
    validated = validate_cut(cut.model_dump(by_alias=True))

    by_role = {c.role: c for c in validated.clips}
    assert by_role["reflection"].act == 1
    assert by_role["reflection"].motion_preset is None
    assert by_role["reflection"].visual_style == "clean_916"
    assert by_role["reflection"].transition_in == "none"

    assert by_role["montage"].act == 2
    assert by_role["montage"].motion_preset is not None
    assert by_role["montage"].visual_style == "clean_916"
    # Intentional Act 2 accent — not the role→hard_cut default (ADR 0021).
    assert by_role["montage"].transition_in == "white_flash"

    assert by_role["hero_outro"].act == 3
    assert by_role["hero_outro"].motion_preset == "handheld_super8_drift"
    assert by_role["hero_outro"].visual_style == "super8_gate"

    # Picture clips must carry a Motion Preset (ADR 0015).
    for c in validated.clips:
        if c.role != "reflection":
            assert c.motion_preset is not None


def test_stub_draft_markers_honor_j_cut_and_drop_timing_constants():
    """j_cut / chorus_drop planned from shared timing constants; snap owns ms (ADR 0019)."""
    reflection_id = _uuid(1)
    photo = _uuid(2)
    assets = [
        {
            "id": reflection_id,
            "type": "voice",
            "order": 0,
            "durationMs": 5000,
        },
        {"id": photo, "type": "photo", "order": 1},
    ]
    ms_per_beat = 500
    fake_grid = _grid(40, ms_per_beat=ms_per_beat)
    cut = _draft(assets, grid=fake_grid, reflection_id=reflection_id)
    validated = validate_cut(cut.model_dump(by_alias=True))

    reflection = next(c for c in validated.clips if c.role == "reflection")
    assert reflection.end_ms is not None
    speech_end_ms = reflection.end_ms

    j_cut = next(m for m in validated.markers if m.type == "j_cut")
    expected_j = max(0, speech_end_ms - int(J_CUT_PRE_ROLL_SEC * 1000))
    # Marker lands on a beat ≤ expected pre-roll start (nearest grid beat).
    assert j_cut.at_ms <= expected_j + ms_per_beat
    assert abs(j_cut.at_ms - expected_j) <= ms_per_beat

    chorus = next(m for m in validated.markers if m.type == "chorus_drop")
    expected_drop = speech_end_ms + int(DROP_OFFSET_AFTER_SPEECH_SEC * 1000)
    assert abs(chorus.at_ms - expected_drop) <= ms_per_beat


def test_stub_draft_voice_only_reflection_clean_slate_no_motion():
    """Voice-only Reflection: clean_916 slate, no Motion Preset, no borrowed photo (ADR 0020)."""
    reflection_id = _uuid(1)
    assets = [
        {
            "id": reflection_id,
            "type": "voice",
            "order": 0,
            "durationMs": 3000,
        },
    ]
    cut = _draft(assets, grid=_grid(24), reflection_id=reflection_id)
    validated = validate_cut(cut.model_dump(by_alias=True))

    assert all(c.asset_id == reflection_id for c in validated.clips)
    assert len(validated.clips) == 1
    slate = validated.clips[0]
    assert slate.role == "hero_outro"  # Cut still requires hero_outro
    assert slate.visual_style == "clean_916"
    assert slate.motion_preset is None
