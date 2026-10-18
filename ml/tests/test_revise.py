"""Constrained revise order helpers."""

from momentos_ml.agents.revise_cut import EditorPatch, apply_patch_to_order


def test_remove_and_reorder():
    assets = [
        {"id": "a", "type": "photo", "order": 0},
        {"id": "b", "type": "photo", "order": 1},
        {"id": "c", "type": "photo", "order": 2},
    ]
    prior = {
        "id": "cut-1",
        "momentId": "m-1",
        "version": 1,
        "songPickId": "live-a-life-you-will-remember",
        "clips": [
            {"assetId": "a", "order": 0, "startMs": 0, "endMs": 500, "role": "montage"},
            {"assetId": "b", "order": 1, "startMs": 500, "endMs": 1000, "role": "montage"},
            {"assetId": "c", "order": 2, "startMs": 1000, "endMs": 1500, "role": "hero_outro"},
        ],
    }
    patch = EditorPatch.model_validate(
        {"removeAssetId": "b", "reorderAssetIds": ["c", "a"], "reDraft": True}
    )
    remaining, order, hint = apply_patch_to_order(assets, prior, patch)
    assert order == ["c", "a"]
    assert [r["id"] for r in remaining] == ["c", "a"]
    assert "c" in hint and "a" in hint
