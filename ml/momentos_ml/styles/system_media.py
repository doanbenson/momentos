"""System media catalog + local resolution (SFX / flash / overlays).

Not user Assets. Pack lives under render-service/system-media/ (ADR 0010).
Storage keys mirror Edit Audio seeding: system/style-profiles/<style>/…
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path

from momentos_ml.styles.mvp import MVP_STYLE_PROFILE_ID

STORAGE_KEY_PREFIX = f"system/style-profiles/{MVP_STYLE_PROFILE_ID}"
STYLE_PROFILE_RELATIVE_ROOT = MVP_STYLE_PROFILE_ID


@dataclass(frozen=True)
class SystemMediaItem:
    id: str
    filename: str
    kind: str  # "sfx" | "overlay"
    mime: str


class SystemMediaError(LookupError):
    """Unknown id or missing packed file."""


_ITEMS: dict[str, SystemMediaItem] = {
    "flash": SystemMediaItem(
        id="flash",
        filename="flash.png",
        kind="overlay",
        mime="image/png",
    ),
    "sfx-shutter": SystemMediaItem(
        id="sfx-shutter",
        filename="sfx-shutter.wav",
        kind="sfx",
        mime="audio/wav",
    ),
    "overlay-film-burn": SystemMediaItem(
        id="overlay-film-burn",
        filename="overlay-film-burn.png",
        kind="overlay",
        mime="image/png",
    ),
}


def catalog() -> dict[str, SystemMediaItem]:
    return dict(_ITEMS)


def storage_key(media_id: str) -> str:
    if media_id not in _ITEMS:
        raise SystemMediaError(f"unknown system media id: {media_id}")
    return f"{STORAGE_KEY_PREFIX}/{media_id}"


def default_root() -> Path:
    """Prefer SYSTEM_MEDIA_DIR; else render-service pack relative to repo."""
    env = os.getenv("SYSTEM_MEDIA_DIR") or os.getenv("RENDER_SYSTEM_MEDIA_DIR")
    if env:
        return Path(env)
    # ml/tests → repo → services/render-service/system-media
    here = Path(__file__).resolve()
    candidate = here.parents[3] / "services" / "render-service" / "system-media"
    return candidate


def resolve(root: Path | str | None, media_id: str) -> Path:
    if media_id not in _ITEMS:
        raise SystemMediaError(f"unknown system media id: {media_id}")
    base = Path(root) if root is not None else default_root()
    path = base / STYLE_PROFILE_RELATIVE_ROOT / _ITEMS[media_id].filename
    if not path.is_file():
        raise SystemMediaError(f"system media not found: {media_id} ({path})")
    return path


def resolve_all(root: Path | str | None = None) -> dict[str, Path]:
    base = Path(root) if root is not None else default_root()
    return {media_id: resolve(base, media_id) for media_id in _ITEMS}
