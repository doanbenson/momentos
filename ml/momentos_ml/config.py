"""Runtime config from env. Mirrors mobile MVP Song Pick id."""

from __future__ import annotations

import os
from dataclasses import dataclass

# Keep in sync with apps/mobile/src/config/song-pick.ts and
# packages/contracts/mvp-constants.json (see styles.song_pick).
MVP_SONG_PICK_ID = "live-a-life-you-will-remember"
MVP_SONG_TITLE = "Live A Life You Will Remember"

# Photo hold duration in beats (legal span; see styles.song_pick.LEGAL_SPAN_BEATS).
PHOTO_HOLD_BEATS = 2

# Repair / retry budget after first attempt (GRILL Q5): ≤2 then fail.
MAX_REPAIR_ATTEMPTS = 2


@dataclass(frozen=True)
class Settings:
    cut_planner: str  # "gemini" | "stub"
    vision_scoring: str  # "metadata" | "gemini"
    gemini_api_key: str | None
    gemini_model: str
    moment_service_url: str | None
    job_queue_url: str | None
    internal_service_token: str | None
    edit_audio_path: str | None
    poll_interval_s: float
    beat_cache_dir: str


def load_settings() -> Settings:
    planner = os.getenv("CUT_PLANNER", "gemini").strip().lower()
    if planner not in ("gemini", "stub"):
        planner = "gemini"
    # Default metadata: Gemini placements do not consume vision scores today.
    vision = os.getenv("VISION_SCORING", "metadata").strip().lower()
    if vision not in ("metadata", "gemini"):
        vision = "metadata"
    return Settings(
        cut_planner=planner,
        vision_scoring=vision,
        gemini_api_key=os.getenv("GEMINI_API_KEY") or None,
        gemini_model=os.getenv("GEMINI_MODEL", "gemini-3.8-flash"),
        moment_service_url=os.getenv("MOMENT_SERVICE_URL") or None,
        job_queue_url=os.getenv("JOB_QUEUE_URL") or None,
        internal_service_token=os.getenv("INTERNAL_SERVICE_TOKEN") or None,
        edit_audio_path=os.getenv("EDIT_AUDIO_PATH") or None,
        poll_interval_s=float(os.getenv("WORKER_POLL_INTERVAL_S", "2.0")),
        beat_cache_dir=os.getenv("BEAT_CACHE_DIR", ".beat_cache"),
    )
