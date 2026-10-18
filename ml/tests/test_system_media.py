"""Seam 1/2 hooks: system media catalog + local path resolution (not user Assets)."""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from momentos_ml.styles import system_media as sm

_CONTRACTS = (
    Path(__file__).resolve().parents[2]
    / "packages"
    / "contracts"
    / "mvp-constants.json"
)
_PACK_ROOT = (
    Path(__file__).resolve().parents[2]
    / "services"
    / "render-service"
    / "system-media"
)


def test_catalog_ids_match_contracts():
    expected = json.loads(_CONTRACTS.read_text(encoding="utf-8"))
    items = expected["systemMedia"]["items"]
    catalog = sm.catalog()
    assert set(catalog) == set(items)
    for media_id, meta in items.items():
        assert catalog[media_id].filename == meta["filename"]
        assert catalog[media_id].kind == meta["kind"]
        assert sm.storage_key(media_id) == (
            f"{expected['systemMedia']['storageKeyPrefix']}/{media_id}"
        )


def test_resolve_finds_packed_placeholders():
    paths = sm.resolve_all(_PACK_ROOT)
    assert set(paths) == {"flash", "sfx-shutter", "overlay-film-burn"}
    for media_id, path in paths.items():
        assert path.is_file(), f"missing packed system media: {media_id} -> {path}"
        assert path.stat().st_size > 0


def test_resolve_unknown_id_raises():
    with pytest.raises(sm.SystemMediaError, match="unknown"):
        sm.resolve(_PACK_ROOT, "not-a-real-media")


def test_resolve_missing_file_raises(tmp_path: Path):
    with pytest.raises(sm.SystemMediaError, match="not found"):
        sm.resolve(tmp_path, "flash")
