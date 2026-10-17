"""Draft a Cut from Assets + fixed Style Profile / Vibe / Song Pick."""

from typing import Any


def draft_cut(assets: list[dict[str, Any]], song_pick_id: str) -> dict[str, Any]:
    """Return a Cut-shaped dict. Replace with real agent pipeline."""
    return {
        "version": 1,
        "song_pick_id": song_pick_id,
        "clips": [{"asset_id": a["id"], "order": i} for i, a in enumerate(assets)],
    }
