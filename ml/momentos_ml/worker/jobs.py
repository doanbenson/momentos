"""cut-draft job payload parse + claim/complete against moment-service."""

from __future__ import annotations

import logging
import os
import socket
from typing import Any, Literal

import httpx
from pydantic import BaseModel, ConfigDict, Field

from momentos_ml.agents.draft_cut import draft_cut
from momentos_ml.agents.revise_cut import EditorPatch, revise_cut
from momentos_ml.config import Settings
from momentos_ml.features.extract import JobAsset
from momentos_ml.validate.cut_schema import Cut

logger = logging.getLogger(__name__)

JobMode = Literal["draft", "revise"]


class CutDraftJob(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    job_type: str = Field(default="cut-draft", alias="jobType")
    job_id: str | None = Field(default=None, alias="jobId")
    moment_id: str = Field(alias="momentId")
    song_pick_id: str = Field(
        default="live-a-life-you-will-remember", alias="songPickId"
    )
    edit_audio_url: str | None = Field(default=None, alias="editAudioUrl")
    mode: JobMode = "draft"
    assets: list[JobAsset] = Field(default_factory=list)
    reflection_asset_id: str | None = Field(default=None, alias="reflectionAssetId")
    reflection_question: str | None = Field(default=None, alias="reflectionQuestion")
    prior_cut: dict[str, Any] | None = Field(default=None, alias="priorCut")
    editor_patch: EditorPatch | None = Field(default=None, alias="editorPatch")


def parse_job(payload: dict[str, Any]) -> CutDraftJob:
    return CutDraftJob.model_validate(payload)


def parse_claim_response(data: dict[str, Any]) -> CutDraftJob:
    """Unwrap moment-service claim envelope: { id, momentId, payload, ... }."""
    import json

    inner = data.get("payload")
    if isinstance(inner, str):
        inner = json.loads(inner) if inner.strip() else {}
    if not isinstance(inner, dict):
        inner = {}

    merged = dict(inner)
    if "moment_id" in merged and "momentId" not in merged:
        merged["momentId"] = str(merged["moment_id"])

    envelope_moment = data.get("momentId") or data.get("moment_id")
    if envelope_moment and not merged.get("momentId"):
        merged["momentId"] = str(envelope_moment)
    if data.get("id") and not merged.get("jobId"):
        merged["jobId"] = str(data["id"])
    if data.get("jobType") and not merged.get("jobType"):
        merged["jobType"] = data["jobType"]
    if data.get("job_type") and not merged.get("jobType"):
        merged["jobType"] = data["job_type"]

    return parse_job(merged)


def process_job(job: CutDraftJob, settings: Settings) -> Cut:
    assets = [a.model_dump(by_alias=True) for a in job.assets]
    if job.mode == "revise":
        if job.prior_cut is None or job.editor_patch is None:
            raise ValueError("revise mode requires priorCut and editorPatch")
        return revise_cut(
            assets=assets,
            song_pick_id=job.song_pick_id,
            moment_id=job.moment_id,
            prior_cut=job.prior_cut,
            editor_patch=job.editor_patch,
            edit_audio_url=job.edit_audio_url,
            settings=settings,
            reflection_asset_id=job.reflection_asset_id,
            reflection_question=job.reflection_question,
        )
    return draft_cut(
        assets,
        job.song_pick_id,
        moment_id=job.moment_id,
        edit_audio_url=job.edit_audio_url,
        settings=settings,
        reflection_asset_id=job.reflection_asset_id,
        reflection_question=job.reflection_question,
    )


def _auth_headers(settings: Settings) -> dict[str, str]:
    headers = {"Content-Type": "application/json"}
    if settings.internal_service_token:
        headers["Authorization"] = f"Bearer {settings.internal_service_token}"
    return headers


def _worker_id() -> str:
    return os.getenv("WORKER_ID") or f"ml-{socket.gethostname()}"


def claim_job(settings: Settings) -> CutDraftJob | None:
    """Claim next cut-draft job: POST /v1/internal/jobs/claim."""
    base = settings.moment_service_url
    if not base:
        return None

    claim_url = f"{base.rstrip('/')}/v1/internal/jobs/claim"
    body = {"jobType": "cut-draft", "workerId": _worker_id()}

    try:
        with httpx.Client(timeout=30.0) as client:
            resp = client.post(claim_url, json=body, headers=_auth_headers(settings))
            if resp.status_code == 204:
                return None
            if resp.status_code == 404:
                logger.debug("claim endpoint not ready (404)")
                return None
            resp.raise_for_status()
            data = resp.json()
            if not data:
                return None
            return parse_claim_response(data)
    except (httpx.HTTPError, ValueError, TypeError) as e:
        logger.warning("claim_job failed: %s", e)
        return None
    except Exception as e:
        # Bad broker payloads must not kill the poll loop.
        logger.warning("claim_job parse failed: %s", e)
        return None


def complete_job(settings: Settings, job: CutDraftJob, cut: Cut) -> None:
    """POST /v1/internal/jobs/{jobId}/complete with ok + cut/clips."""
    base = settings.moment_service_url
    if not base or not job.job_id:
        logger.info(
            "MOMENT_SERVICE_URL/jobId unset — Cut ready (dry-run): moment=%s clips=%s",
            job.moment_id,
            len(cut.clips),
        )
        return

    url = f"{base.rstrip('/')}/v1/internal/jobs/{job.job_id}/complete"
    cut_body = cut.model_dump(by_alias=True)
    body = {
        "ok": True,
        "songPickId": job.song_pick_id,
        "cut": cut_body,
        "clips": cut_body.get("clips", []),
    }
    with httpx.Client(timeout=30.0) as client:
        resp = client.post(url, json=body, headers=_auth_headers(settings))
        resp.raise_for_status()


def fail_job(
    settings: Settings,
    job: CutDraftJob,
    *,
    message: str,
    attempts: int,
    error_code: str = "cut_draft_failed",
) -> None:
    """Complete job with ok=false → Moment failed."""
    base = settings.moment_service_url
    if not base or not job.job_id:
        logger.error(
            "MOMENT_SERVICE_URL/jobId unset — job failed (dry-run): moment=%s %s %s attempts=%s",
            job.moment_id,
            error_code,
            message,
            attempts,
        )
        return

    url = f"{base.rstrip('/')}/v1/internal/jobs/{job.job_id}/complete"
    body = {
        "ok": False,
        "errorCode": error_code,
        "errorMessage": f"{message} (attempts={attempts})",
    }
    with httpx.Client(timeout=30.0) as client:
        resp = client.post(url, json=body, headers=_auth_headers(settings))
        resp.raise_for_status()
