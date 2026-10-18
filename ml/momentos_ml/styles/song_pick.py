"""Song Pick / Style Profile constants for legal spans and chorus drop (ADR 0007).

Canonical numeric source: packages/contracts/mvp-constants.json (parity tests).
Keep Song Pick id in sync with apps/mobile/src/config/song-pick.ts.
"""

from __future__ import annotations

from momentos_ml.config import (
    MVP_SONG_PICK_ID,
    MVP_SONG_TITLE,
    PHOTO_HOLD_BEATS as _PHOTO_HOLD_BEATS,
)
from momentos_ml.styles.mvp import MVP_STYLE_PROFILE_ID, MVP_VIBE_ID

SONG_PICK_ID = MVP_SONG_PICK_ID
SONG_PICK_TITLE = MVP_SONG_TITLE
STYLE_PROFILE_ID = MVP_STYLE_PROFILE_ID
VIBE_ID = MVP_VIBE_ID

# Nominal BPM for this Song Pick when calibrating offline; Beat Grid still comes
# from librosa on Edit Audio (not a fake clock).
NOMINAL_BPM = 126

# Legal clip span lengths in beats (half-open beat indices).
LEGAL_SPAN_BEATS: tuple[int, ...] = (1, 2, 4)

# hero_outro must land on a legal 4-beat span.
HERO_OUTRO_SPAN_BEATS = 4

# Photo hold default (mirrored in momentos_ml.config.PHOTO_HOLD_BEATS).
PHOTO_HOLD_BEATS = _PHOTO_HOLD_BEATS

# Chorus drop as Beat Grid index for this Song Pick. Calibrate against Edit Audio
# offline when the stem is available (not in git — ADR 0004).
CHORUS_DROP_BEAT_INDEX = 96

# J-cut / post-speech drop timing (ADR 0019). ml plans markers; render mixes.
J_CUT_PRE_ROLL_SEC = 4.0
DROP_OFFSET_AFTER_SPEECH_SEC = 0.25


def is_legal_span_beats(span: int) -> bool:
    return span in LEGAL_SPAN_BEATS
