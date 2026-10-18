"""Reflection aligner: word cues + keep-segments (ADR 0013).

Default production backend is faster-whisper. CI / CUT_PLANNER=stub uses the
deterministic stub so tests never call Whisper or Gemini.
"""

from __future__ import annotations

import logging
from typing import Literal

from pydantic import BaseModel, Field

from momentos_ml.validate.cut_schema import CutCaption

logger = logging.getLogger(__name__)

AlignerBackend = Literal["stub", "faster_whisper"]


class KeepSegment(BaseModel):
    """Inclusive-start / exclusive-end window (ms) of Reflection audio to keep."""

    start_ms: int = Field(ge=0)
    end_ms: int = Field(ge=1)


class AlignmentResult(BaseModel):
    """Alignment relative to Reflection Asset t=0 (not yet timeline-offset)."""

    captions: list[CutCaption]
    keep_segments: list[KeepSegment]


def align_reflection(
    *,
    reflection_asset_id: str,
    reflection_question: str | None = None,
    duration_ms: int | None = None,
    audio_path: str | None = None,
    backend: AlignerBackend = "stub",
) -> AlignmentResult:
    """
    Align Reflection speech → caption cues + keep-segments.

    Captions are Asset-local ms. Director/snap offsets them onto the Cut timeline
    so re-render does not re-run Whisper (ADR 0009).
    """
    if backend == "faster_whisper":
        return _align_faster_whisper(
            audio_path=audio_path,
            duration_ms=duration_ms,
            reflection_question=reflection_question,
        )
    return _align_stub(
        reflection_asset_id=reflection_asset_id,
        reflection_question=reflection_question,
        duration_ms=duration_ms,
    )


def _align_stub(
    *,
    reflection_asset_id: str,
    reflection_question: str | None,
    duration_ms: int | None,
) -> AlignmentResult:
    """Deterministic captions for CI — no Whisper / network."""
    del reflection_asset_id  # id is for logging / future cache keys
    text = (reflection_question or "this week was mine").strip()
    words = [w for w in text.replace("?", "").split() if w]
    if not words:
        words = ["this", "week", "was", "mine"]

    total = duration_ms if duration_ms and duration_ms > 0 else max(len(words) * 400, 1200)
    slot = max(total // len(words), 1)
    captions: list[CutCaption] = []
    t = 0
    for i, word in enumerate(words):
        end = total if i == len(words) - 1 else min(t + slot, total)
        if end <= t:
            end = t + 1
        captions.append(CutCaption(text=word, startMs=t, endMs=end))
        t = end

    keep_end = max(total, captions[-1].end_ms if captions else 1)
    return AlignmentResult(
        captions=captions,
        keep_segments=[KeepSegment(start_ms=0, end_ms=keep_end)],
    )


_whisper_model = None


def _get_whisper_model():
    """Lazy singleton — avoid reloading weights on every cut-draft job."""
    global _whisper_model
    if _whisper_model is not None:
        return _whisper_model
    from faster_whisper import WhisperModel  # type: ignore[import-not-found]

    _whisper_model = WhisperModel("base", device="cpu", compute_type="int8")
    return _whisper_model


def _align_faster_whisper(
    *,
    audio_path: str | None,
    duration_ms: int | None,
    reflection_question: str | None,
) -> AlignmentResult:
    if not audio_path:
        raise ValueError(
            "faster_whisper requested without audio_path "
            "(Reflection Asset storageUrl missing or download failed)"
        )
    try:
        model = _get_whisper_model()
    except ImportError:
        logger.warning("faster-whisper not installed; falling back to stub aligner")
        return _align_stub(
            reflection_asset_id="unknown",
            reflection_question=reflection_question,
            duration_ms=duration_ms,
        )

    segments, _info = model.transcribe(audio_path, word_timestamps=True)
    captions: list[CutCaption] = []
    for seg in segments:
        words = getattr(seg, "words", None) or []
        for w in words:
            word = (getattr(w, "word", "") or "").strip()
            if not word:
                continue
            start = int(round(float(w.start) * 1000))
            end = int(round(float(w.end) * 1000))
            if end <= start:
                end = start + 1
            captions.append(CutCaption(text=word, startMs=start, endMs=end))

    if not captions:
        return _align_stub(
            reflection_asset_id="unknown",
            reflection_question=reflection_question,
            duration_ms=duration_ms,
        )

    if duration_ms and duration_ms > 0:
        captions = [
            c
            if c.end_ms <= duration_ms
            else CutCaption(text=c.text, startMs=c.start_ms, endMs=duration_ms)
            for c in captions
            if c.start_ms < duration_ms
        ]
        if not captions:
            return _align_stub(
                reflection_asset_id="unknown",
                reflection_question=reflection_question,
                duration_ms=duration_ms,
            )

    # Collapse silence gaps > 0.35s into keep_segments (PRD Act 1 trim).
    keep = _keep_segments_from_captions(captions, gap_ms=350)
    return AlignmentResult(captions=captions, keep_segments=keep)


def _keep_segments_from_captions(
    captions: list[CutCaption],
    *,
    gap_ms: int = 350,
) -> list[KeepSegment]:
    if not captions:
        return [KeepSegment(start_ms=0, end_ms=1)]
    segments: list[KeepSegment] = []
    start = captions[0].start_ms
    prev_end = captions[0].end_ms
    for c in captions[1:]:
        if c.start_ms - prev_end > gap_ms:
            segments.append(KeepSegment(start_ms=start, end_ms=max(prev_end, start + 1)))
            start = c.start_ms
        prev_end = max(prev_end, c.end_ms)
    segments.append(KeepSegment(start_ms=start, end_ms=max(prev_end, start + 1)))
    return segments
