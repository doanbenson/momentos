"""Worker loop: claim → draft|revise → complete / fail (retry ≤2 inside draft)."""

from __future__ import annotations

import logging
import os
import time

from momentos_ml.config import MAX_REPAIR_ATTEMPTS, load_settings
from momentos_ml.worker.jobs import (
    claim_job,
    complete_job,
    fail_job,
    parse_job,
    process_job,
)

logger = logging.getLogger(__name__)


def run_once(
    payload: dict | None = None,
    *,
    settings=None,
) -> bool:
    """
    Process one job. If payload is given, skip claim (local/demo).
    Returns True if a job was handled.
    """
    settings = settings or load_settings()
    if payload is not None:
        job = parse_job(payload)
    else:
        job = claim_job(settings)
        if job is None:
            return False

    attempts = MAX_REPAIR_ATTEMPTS + 1
    try:
        cut = process_job(job, settings)
        complete_job(settings, job, cut)
        logger.info(
            "completed cut-draft moment=%s version=%s clips=%s",
            job.moment_id,
            cut.version,
            len(cut.clips),
        )
        return True
    except Exception as e:
        logger.exception("cut-draft failed moment=%s", job.moment_id)
        try:
            fail_job(
                settings,
                job,
                message=str(e),
                attempts=attempts,
            )
        except Exception:
            logger.exception("fail_job hook error")
        return True


def run() -> None:
    logging.basicConfig(
        level=os.getenv("LOG_LEVEL", "INFO"),
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )
    settings = load_settings()
    logger.info(
        "momentos-ml worker starting planner=%s moment_service=%s",
        settings.cut_planner,
        settings.moment_service_url or "(dry-run)",
    )

    # One-shot local job via CUT_DRAFT_JOB_JSON for demos without a broker
    oneshot = os.getenv("CUT_DRAFT_JOB_JSON")
    if oneshot:
        import json

        run_once(json.loads(oneshot), settings=settings)
        return

    if not settings.job_queue_url and not settings.moment_service_url:
        logger.warning(
            "No JOB_QUEUE_URL / MOMENT_SERVICE_URL — idle poll with empty claims. "
            "Set CUT_DRAFT_JOB_JSON for a one-shot local job, or CUT_PLANNER=stub + "
            "EDIT_AUDIO_PATH to exercise the pipeline in tests."
        )

    while True:
        handled = run_once(settings=settings)
        if not handled:
            time.sleep(settings.poll_interval_s)
