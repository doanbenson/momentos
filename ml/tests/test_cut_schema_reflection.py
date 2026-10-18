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
        ),
        CutClip(
            assetId="22222222-2222-2222-2222-222222222222",
            order=1,
            startMs=2000,
            endMs=3000,
            role="montage",
        ),
    ]
    if with_hero:
        clips.append(
            CutClip(
                assetId="33333333-3333-3333-3333-333333333333",
                order=2,
                startMs=3000,
                endMs=5000,
                role="hero_outro",
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
