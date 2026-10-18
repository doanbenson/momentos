"""Gemini enrich forces a 4-beat hero_outro before snap/validate."""

from momentos_ml.agents.draft_cut import _enrich_gemini_placements
from momentos_ml.agents.prompts import SYSTEM_PROMPT
from momentos_ml.beat.grid import BeatGrid
from momentos_ml.beat.snap import PlannerPlacement, snap_to_grid
from momentos_ml.styles.song_pick import HERO_OUTRO_SPAN_BEATS, NOMINAL_BPM
from momentos_ml.validate.cut_schema import build_cut, validate_cut


def test_system_prompt_requires_four_beat_finale():
    assert f"exclusive {HERO_OUTRO_SPAN_BEATS}-beat span" in SYSTEM_PROMPT
    assert "hero_outro" in SYSTEM_PROMPT


def _grid_126(n_beats: int = 16) -> BeatGrid:
    gap = 60_000 / NOMINAL_BPM
    return BeatGrid(
        song_pick_id="live-a-life-you-will-remember",
        audio_content_hash="test",
        beat_ms=[int(round(i * gap)) for i in range(n_beats)],
        tempo_bpm=float(NOMINAL_BPM),
    )


def test_enrich_expands_two_beat_finale_to_four_beats():
    reflection = "11111111-1111-1111-1111-111111111111"
    hero = "33333333-3333-3333-3333-333333333333"
    raw = [
        PlannerPlacement(asset_id=reflection, start_beat=0, end_beat=4),
        PlannerPlacement(asset_id="22222222-2222-2222-2222-222222222222", start_beat=4, end_beat=6),
        # Gemini often emits a 2-beat photo hold as the finale (~952ms @ 126 BPM).
        PlannerPlacement(asset_id=hero, start_beat=10, end_beat=12),
    ]
    grid = _grid_126()
    enriched = _enrich_gemini_placements(
        raw,
        reflection_asset_id=reflection,
        beat_count=len(grid.beat_ms),
    )
    last = enriched[-1]
    assert last.role == "hero_outro"
    assert last.end_beat - last.start_beat == HERO_OUTRO_SPAN_BEATS
    assert last.start_beat == 10
    assert last.end_beat == 14

    clips = snap_to_grid(enriched, grid)
    hero_clip = clips[-1]
    assert hero_clip.role == "hero_outro"
    span = hero_clip.end_ms - hero_clip.start_ms
    expected = int(round(HERO_OUTRO_SPAN_BEATS * (60_000 / NOMINAL_BPM)))
    assert abs(span - expected) <= int(0.5 * (60_000 / NOMINAL_BPM))

    cut = build_cut(
        moment_id="44444444-4444-4444-4444-444444444444",
        song_pick_id="live-a-life-you-will-remember",
        clips=clips,
    )
    validated = validate_cut(cut.model_dump(by_alias=True))
    assert validated.clips[-1].role == "hero_outro"


def test_enrich_clamps_hero_near_grid_end():
    hero = "33333333-3333-3333-3333-333333333333"
    raw = [
        PlannerPlacement(asset_id="a", start_beat=0, end_beat=2),
        PlannerPlacement(asset_id=hero, start_beat=14, end_beat=16),  # only 2 beats left
    ]
    beat_count = 16
    enriched = _enrich_gemini_placements(
        raw, reflection_asset_id=None, beat_count=beat_count
    )
    last = enriched[-1]
    assert last.role == "hero_outro"
    assert last.start_beat == beat_count - HERO_OUTRO_SPAN_BEATS
    assert last.end_beat == beat_count
