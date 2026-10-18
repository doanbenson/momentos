"""Gemini montage-order parser + prompt contract (beats stay in Python)."""

from momentos_ml.agents.gemini_planner import parse_montage_order
from momentos_ml.agents.prompts import ORDER_SYSTEM_PROMPT
from momentos_ml.styles.song_pick import HERO_OUTRO_SPAN_BEATS


def test_order_system_prompt_forbids_beat_indices():
    assert "Do NOT emit beat indices" in ORDER_SYSTEM_PROMPT
    assert "storytelling order" in ORDER_SYSTEM_PROMPT


def test_parse_montage_order_appends_omitted_ids():
    allowed = {
        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
        "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
        "cccccccc-cccc-cccc-cccc-cccccccccccc",
    }
    raw = '{"asset_ids": ["bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"]}'
    ordered = parse_montage_order(raw, allowed_ids=allowed)
    assert ordered[0] == "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
    assert set(ordered) == allowed
    assert len(ordered) == 3


def test_hero_outro_span_constant_unchanged():
    # Deterministic director still uses the legal 4-beat finale.
    assert HERO_OUTRO_SPAN_BEATS == 4
