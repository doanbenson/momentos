"""Seam 1 hooks: Song Pick / Style Profile constants + parity with contracts."""

from __future__ import annotations

import json
from pathlib import Path

from momentos_ml.styles import song_pick as sp

_CONTRACTS = (
    Path(__file__).resolve().parents[2]
    / "packages"
    / "contracts"
    / "mvp-constants.json"
)


def _parity() -> dict:
    return json.loads(_CONTRACTS.read_text(encoding="utf-8"))


def test_mvp_song_pick_id_unchanged():
    assert sp.SONG_PICK_ID == "live-a-life-you-will-remember"
    assert sp.STYLE_PROFILE_ID == "life-on-film-v1"


def test_legal_span_beats_are_one_two_four():
    assert sp.LEGAL_SPAN_BEATS == (1, 2, 4)
    assert sp.is_legal_span_beats(1)
    assert sp.is_legal_span_beats(2)
    assert sp.is_legal_span_beats(4)
    assert not sp.is_legal_span_beats(3)
    assert not sp.is_legal_span_beats(0)


def test_hero_outro_requires_four_beat_span():
    assert sp.HERO_OUTRO_SPAN_BEATS == 4
    assert sp.is_legal_span_beats(sp.HERO_OUTRO_SPAN_BEATS)


def test_chorus_drop_beat_index_is_non_negative():
    assert sp.CHORUS_DROP_BEAT_INDEX >= 0
    assert sp.NOMINAL_BPM == 126


def test_constants_match_contracts_parity_file():
    expected = _parity()
    assert sp.SONG_PICK_ID == expected["songPickId"]
    assert sp.SONG_PICK_TITLE == expected["songPickTitle"]
    assert sp.STYLE_PROFILE_ID == expected["styleProfileId"]
    assert sp.VIBE_ID == expected["vibeId"]
    assert sp.NOMINAL_BPM == expected["nominalBpm"]
    assert list(sp.LEGAL_SPAN_BEATS) == expected["legalSpanBeats"]
    assert sp.HERO_OUTRO_SPAN_BEATS == expected["heroOutroSpanBeats"]
    assert sp.PHOTO_HOLD_BEATS == expected["photoHoldBeats"]
    assert sp.CHORUS_DROP_BEAT_INDEX == expected["chorusDropBeatIndex"]
