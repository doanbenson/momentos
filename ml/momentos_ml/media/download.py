"""Fetch signed Asset URLs to a temp file for Whisper / Gemini vision."""

from __future__ import annotations

import logging
import tempfile
from pathlib import Path

import httpx

logger = logging.getLogger(__name__)

_SUFFIX_BY_TYPE = {
    "photo": ".jpg",
    "video": ".mp4",
    "voice": ".m4a",
}

# Prefer Content-Type from the signed GET (Reflection talking-head is often video/quicktime).
_SUFFIX_BY_MIME = {
    "image/jpeg": ".jpg",
    "image/jpg": ".jpg",
    "image/png": ".png",
    "image/heic": ".heic",
    "image/heif": ".heif",
    "video/mp4": ".mp4",
    "video/quicktime": ".mov",
    "audio/mp4": ".m4a",
    "audio/m4a": ".m4a",
    "audio/mpeg": ".mp3",
    "audio/wav": ".wav",
    "audio/x-wav": ".wav",
    "audio/webm": ".webm",
    "video/webm": ".webm",
}


def _suffix_for(*, asset_type: str, content_type: str | None) -> str:
    if content_type:
        mime = content_type.split(";", 1)[0].strip().lower()
        if mime in _SUFFIX_BY_MIME:
            return _SUFFIX_BY_MIME[mime]
    return _SUFFIX_BY_TYPE.get(asset_type, ".bin")


def download_url_to_temp(
    url: str,
    *,
    suffix: str = ".bin",
    asset_type: str | None = None,
    timeout: float = 120.0,
) -> Path:
    """GET url → NamedTemporaryFile path (caller may unlink when done)."""
    if not url:
        raise ValueError("empty download url")
    with httpx.Client(timeout=timeout, follow_redirects=True) as client:
        resp = client.get(url)
        resp.raise_for_status()
        resolved = _suffix_for(
            asset_type=asset_type or "",
            content_type=resp.headers.get("content-type"),
        )
        if resolved != ".bin":
            suffix = resolved
        fd, name = tempfile.mkstemp(prefix="momentos-asset-", suffix=suffix)
        path = Path(name)
        try:
            with open(fd, "wb") as f:
                f.write(resp.content)
        except Exception:
            path.unlink(missing_ok=True)
            raise
        logger.info(
            "downloaded asset → %s (%s bytes, content-type=%s)",
            path,
            path.stat().st_size,
            resp.headers.get("content-type"),
        )
        return path


def materialize_asset(
    *,
    storage_url: str | None,
    asset_type: str,
) -> Path | None:
    """Download when storage_url is set; return None if unavailable."""
    if not storage_url:
        return None
    try:
        return download_url_to_temp(
            storage_url,
            suffix=_SUFFIX_BY_TYPE.get(asset_type, ".bin"),
            asset_type=asset_type,
        )
    except Exception as e:
        logger.warning("asset download failed (%s): %s", asset_type, e)
        return None
