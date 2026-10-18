"""Contract tests for Cut roles / captions / markers / hero_outro (feeds seam 1)."""

import pytest
from pydantic import ValidationError

from momentos_ml.validate.cut_schema import build_cut, validate_cut
from momentos_ml.validate.cut_schema import CutClip


def _base_clips(*, with_hero: bool = True) -> list[CutClip]:
    clips = [
        CutClip(
            assetId="11111111-1111-1111-1111-111111111111",
            order=0,
            startMs=0,
            endMs=2000,
            role="reflection",
            act=1,
            visualStyle="clean_916",
            transitionIn="none",
        ),
        CutClip(
            assetId="22222222-2222-2222-2222-222222222222",
            order=1,
            startMs=2000,
            endMs=3000,
            role="montage",
            act=2,
            motionPreset="whip_punch_in",
            visualStyle="clean_916",
            transitionIn="white_flash",
        ),
    ]
    if with_hero:
        clips.append(
            CutClip(
                assetId="33333333-3333-3333-3333-333333333333",
                order=2,
                startMs=3000,
                endMs=4905,  # ~4 beats @ 126 BPM (1905ms)
                role="hero_outro",
                act=3,
                motionPreset="handheld_super8_drift",
                visualStyle="super8_gate",
                transitionIn="hard_cut",
            )
        )
    return clips


def test_cut_accepts_roles_captions_markers():
    cut = build_cut(
        moment_id="44444444-4444-4444-4444-444444444444",
        song_pick_id="live-a-life-you-will-remember",
        clips=_base_clips(),
        captions=[{"text": "hello", "startMs": 0, "endMs": 500}],
        markers=[{"type": "chorus_drop", "atMs": 2000}],
    )
    validated = validate_cut(cut.model_dump(by_alias=True))
    assert [c.role for c in validated.clips] == ["reflection", "montage", "hero_outro"]
    assert validated.captions[0].text == "hello"
    assert validated.markers[0].type == "chorus_drop"
    assert "master_timeline" not in cut.model_dump(by_alias=True)


def test_cut_rejects_missing_hero_outro():
    with pytest.raises(ValidationError):
        build_cut(
            moment_id="44444444-4444-4444-4444-444444444444",
            song_pick_id="live-a-life-you-will-remember",
            clips=_base_clips(with_hero=False),
        )


def test_cut_rejects_short_hero_outro_span():
    clips = _base_clips(with_hero=False)
    clips.append(
        CutClip(
            assetId="33333333-3333-3333-3333-333333333333",
            order=2,
            startMs=3000,
            endMs=3500,  # ~1 beat — illegal for hero_outro
            role="hero_outro",
            act=3,
            motionPreset="handheld_super8_drift",
            visualStyle="super8_gate",
            transitionIn="hard_cut",
        )
    )
    with pytest.raises(ValidationError):
        build_cut(
            moment_id="44444444-4444-4444-4444-444444444444",
            song_pick_id="live-a-life-you-will-remember",
            clips=clips,
        )


def test_cut_accepts_act_motion_look_and_transition_in():
    cut = build_cut(
        moment_id="44444444-4444-4444-4444-444444444444",
        song_pick_id="live-a-life-you-will-remember",
        clips=_base_clips(),
    )
    validated = validate_cut(cut.model_dump(by_alias=True))
    assert [c.act for c in validated.clips] == [1, 2, 3]
    assert validated.clips[0].motion_preset is None
    assert validated.clips[1].motion_preset == "whip_punch_in"
    assert [c.visual_style for c in validated.clips] == [
        "clean_916",
        "clean_916",
        "super8_gate",
    ]
    assert [c.transition_in for c in validated.clips] == [
        "none",
        "white_flash",
        "hard_cut",
    ]
    dumped = cut.model_dump(by_alias=True)
    assert "master_timeline" not in dumped
    assert dumped["clips"][1]["motionPreset"] == "whip_punch_in"
    assert dumped["clips"][2]["visualStyle"] == "super8_gate"


def test_cut_rejects_missing_act():
    with pytest.raises(ValidationError):
        CutClip(
            assetId="22222222-2222-2222-2222-222222222222",
            order=0,
            startMs=0,
            endMs=1000,
            role="montage",
            motionPreset="hook_slow_push",
            visualStyle="clean_916",
            transitionIn="hard_cut",
        )
