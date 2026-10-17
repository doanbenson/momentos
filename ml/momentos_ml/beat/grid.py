"""Deterministic Beat Grid from Edit Audio via librosa."""

from __future__ import annotations

import hashlib
import json
import logging
from pathlib import Path

from pydantic import BaseModel

logger = logging.getLogger(__name__)


class BeatGrid(BaseModel):
    song_pick_id: str
    audio_content_hash: str
    beat_ms: list[int]
    tempo_bpm: float | None = None


def _content_hash(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()[:32]


def _cache_path(cache_dir: Path, song_pick_id: str, content_hash: str) -> Path:
    safe = song_pick_id.replace("/", "_")
    return cache_dir / f"{safe}_{content_hash}.json"


def build_beat_grid(
    audio_bytes: bytes,
    *,
    song_pick_id: str,
    cache_dir: str | Path | None = None,
) -> BeatGrid:
    """
    Extract beat timestamps (ms) from Edit Audio.

    Deterministic for pinned librosa/numpy/scipy/soundfile versions.
    Caches by (songPickId, contentHash) when cache_dir is set.
    """
    content_hash = _content_hash(audio_bytes)
    if cache_dir is not None:
        path = _cache_path(Path(cache_dir), song_pick_id, content_hash)
        if path.is_file():
            return BeatGrid.model_validate_json(path.read_text(encoding="utf-8"))

    import io

    import librosa
    import numpy as np
    import soundfile as sf

    with io.BytesIO(audio_bytes) as buf:
        y, sr = sf.read(buf, always_2d=False)
    if getattr(y, "ndim", 1) > 1:
        y = np.mean(y, axis=1)
    y = np.asarray(y, dtype=np.float32)
    if sr != 22050:
        y = librosa.resample(y, orig_sr=sr, target_sr=22050)
        sr = 22050

    tempo, beat_frames = librosa.beat.beat_track(y=y, sr=sr, units="frames")
    beat_times = librosa.frames_to_time(beat_frames, sr=sr)
    beat_ms = [int(round(t * 1000)) for t in beat_times]

    # Ensure at least a trivial grid so stub/planner can place clips.
    if not beat_ms:
        duration_s = float(len(y)) / float(sr)
        step = 500  # 120 bpm-ish fallback
        beat_ms = list(range(0, max(int(duration_s * 1000), step), step))
        if len(beat_ms) < 2:
            beat_ms = [0, step]

    tempo_bpm: float | None
    if isinstance(tempo, np.ndarray):
        tempo_bpm = float(tempo.flat[0]) if tempo.size else None
    else:
        tempo_bpm = float(tempo) if tempo is not None else None

    grid = BeatGrid(
        song_pick_id=song_pick_id,
        audio_content_hash=content_hash,
        beat_ms=beat_ms,
        tempo_bpm=tempo_bpm,
    )

    if cache_dir is not None:
        path = _cache_path(Path(cache_dir), song_pick_id, content_hash)
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(grid.model_dump(), indent=2), encoding="utf-8")
        logger.debug("cached BeatGrid at %s", path)

    return grid


def load_edit_audio_bytes(
    *,
    edit_audio_url: str | None = None,
    edit_audio_path: str | None = None,
) -> bytes:
    """Load Edit Audio from signed URL or local path. Fail fast if missing."""
    if edit_audio_path:
        path = Path(edit_audio_path)
        if not path.is_file():
            raise FileNotFoundError(f"EDIT_AUDIO_PATH not found: {edit_audio_path}")
        return path.read_bytes()

    if edit_audio_url:
        import httpx

        with httpx.Client(timeout=60.0) as client:
            resp = client.get(edit_audio_url)
            resp.raise_for_status()
            return resp.content

    raise ValueError("Missing Edit Audio: set editAudioUrl on job or EDIT_AUDIO_PATH")
