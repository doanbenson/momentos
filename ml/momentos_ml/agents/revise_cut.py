"""Constrained re-draft for EditorPatch (GRILL Q8)."""

from __future__ import annotations

from typing import Any

from pydantic import BaseModel, ConfigDict, Field

from momentos_ml.agents.draft_cut import draft_cut
from momentos_ml.config import Settings, load_settings
from momentos_ml.features.extract import JobAsset
from momentos_ml.validate.cut_schema import Cut


class EditorReplace(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    from_asset_id: str = Field(alias="fromAssetId")
    to_asset_id: str = Field(alias="toAssetId")


class EditorPatch(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    reorder_asset_ids: list[str] | None = Field(default=None, alias="reorderAssetIds")
    replace: EditorReplace | None = None
    remove_asset_id: str | None = Field(default=None, alias="removeAssetId")
    re_draft: bool | None = Field(default=None, alias="reDraft")
    re_render: bool | None = Field(default=None, alias="reRender")


def apply_patch_to_order(
    assets: list[JobAsset] | list[dict[str, Any]],
    prior_cut: Cut | dict[str, Any] | None,
    patch: EditorPatch | dict[str, Any],
) -> tuple[list[dict[str, Any]], list[str], str]:
    """
    Honor reorder / replace / remove. Returns (assets_for_features, order_ids, hint).

    Does not greenfield-regenerate: order is constrained by the patch + prior Cut.
    """
    p = patch if isinstance(patch, EditorPatch) else EditorPatch.model_validate(patch)
    job_assets = [
        a if isinstance(a, JobAsset) else JobAsset.model_validate(a) for a in assets
    ]
    by_id = {a.id: a for a in job_assets}

    # Start from prior Cut clip order when available
    if prior_cut is not None:
        cut = prior_cut if isinstance(prior_cut, Cut) else Cut.model_validate(prior_cut)
        order = [c.asset_id for c in sorted(cut.clips, key=lambda c: c.order)]
    else:
        order = [a.id for a in sorted(job_assets, key=lambda a: a.order)]

    if p.remove_asset_id:
        order = [aid for aid in order if aid != p.remove_asset_id]
        by_id.pop(p.remove_asset_id, None)

    if p.replace:
        order = [
            p.replace.to_asset_id if aid == p.replace.from_asset_id else aid
            for aid in order
        ]
        if p.replace.from_asset_id in by_id and p.replace.to_asset_id in by_id:
            # keep to; drop from from feature set if not elsewhere
            if p.replace.from_asset_id not in order:
                by_id.pop(p.replace.from_asset_id, None)

    if p.reorder_asset_ids:
        # Honor explicit reorder; keep only ids still present
        seen = set(by_id)
        order = [aid for aid in p.reorder_asset_ids if aid in seen]
        # Append any remaining assets not listed (stable)
        for aid in list(by_id):
            if aid not in order:
                order.append(aid)

    remaining = [by_id[aid].model_dump(by_alias=True) for aid in order if aid in by_id]
    hint = (
        f"Constrained revise. Required asset order: {order}. "
        "Preserve this sequence in placements; retarget beat spans only."
    )
    return remaining, order, hint


def revise_cut(
    *,
    assets: list[dict[str, Any]] | list[JobAsset],
    song_pick_id: str,
    moment_id: str,
    prior_cut: Cut | dict[str, Any],
    editor_patch: EditorPatch | dict[str, Any],
    edit_audio_url: str | None = None,
    edit_audio_path: str | None = None,
    settings: Settings | None = None,
) -> Cut:
    """Same BeatGrid path as draft; placements constrained by Editor order."""
    settings = settings or load_settings()
    remaining, order, hint = apply_patch_to_order(assets, prior_cut, editor_patch)
    if not remaining:
        raise ValueError("revise left no Assets")

    prior = prior_cut if isinstance(prior_cut, Cut) else Cut.model_validate(prior_cut)
    next_version = prior.version + 1

    return draft_cut(
        remaining,
        song_pick_id,
        moment_id=moment_id,
        edit_audio_url=edit_audio_url,
        edit_audio_path=edit_audio_path,
        settings=settings,
        version=next_version,
        prior_hint=hint,
        asset_order=order,
    )
