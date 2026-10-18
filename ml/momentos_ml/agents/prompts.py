"""Gemini system/user templates for single-shot Cut planning."""

from __future__ import annotations

import json

from momentos_ml.config import MVP_SONG_TITLE, PHOTO_HOLD_BEATS
from momentos_ml.features.extract import AssetFeature
from momentos_ml.styles import mvp
from momentos_ml.styles.song_pick import HERO_OUTRO_SPAN_BEATS, NOMINAL_BPM


SYSTEM_PROMPT = """You are the Momentos Cut planner.
You place Week Intake Assets onto a Beat Grid for a Life On Film–inspired Moment.

Rules:
- Output ONLY valid JSON matching the schema (no markdown).
- Emit beat indices only — never milliseconds.
- end_beat is exclusive: clip covers beats [start_beat, end_beat).
- Prefer photo holds of about {photo_beats} beats for non-finale montage clips;
  shorten holds to the minimum legal span (1 beat) before giving up.
- Place EVERY intake Asset at least once. Do not drop assets. Do not reuse the
  same Asset multiple times. If placement is impossible even at minimum spans,
  the pipeline will fail — do not emit a partial plan.
- The FINAL placement must be an exclusive {hero_beats}-beat span
  [start_beat, start_beat+{hero_beats}) — this is the hero_outro; do not use a
  {photo_beats}-beat hold for the last clip.
- Video/voice spans must fit within the asset duration when duration_ms is given
  (approximate: assume ~{ms_per_beat}ms per beat at {bpm} BPM if unsure).
- Use assets in a sensible narrative order.
- Voice assets are VO over the Edit Audio bed — prefer not to truncate mid-word.
- Do not invent asset ids.
""".format(
    photo_beats=PHOTO_HOLD_BEATS,
    hero_beats=HERO_OUTRO_SPAN_BEATS,
    ms_per_beat=int(round(60_000 / NOMINAL_BPM)),
    bpm=NOMINAL_BPM,
)


def build_user_prompt(
    *,
    features: list[AssetFeature],
    beat_count: int,
    beat_ms_sample: list[int],
    song_pick_id: str,
    prior_hint: str | None = None,
) -> str:
    payload = {
        "styleProfile": mvp.MVP_STYLE,
        "vibe": mvp.MVP_VIBE,
        "songPick": {"id": song_pick_id, "title": MVP_SONG_TITLE},
        "beatCount": beat_count,
        "beatMsSample": beat_ms_sample,
        "assets": [f.model_dump() for f in features],
        "outputSchema": {
            "placements": [
                {
                    "asset_id": "uuid",
                    "start_beat": 0,
                    "end_beat": 2,
                }
            ]
        },
    }
    text = (
        "Plan placements for this Moment.\n"
        f"{json.dumps(payload, indent=2)}\n"
    )
    if prior_hint:
        text += f"\nConstraints from Editor revise:\n{prior_hint}\n"
    return text


REPAIR_SUFFIX = (
    "\nYour previous JSON was invalid or failed snap/validation. "
    "Return corrected placements only, same schema. Honor beatCount bounds. "
    f"If hero_outro failed, the last placement MUST span exactly "
    f"{HERO_OUTRO_SPAN_BEATS} beats [start, start+{HERO_OUTRO_SPAN_BEATS})."
)
